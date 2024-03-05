// package main is a program that will start a container with attestation.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/namespaces"
	"github.com/google/go-tpm-tools/client"
	"github.com/google/go-tpm-tools/launcher"
	"github.com/google/go-tpm-tools/launcher/internal/experiments"
	"github.com/google/go-tpm-tools/launcher/launcherfile"
	"github.com/google/go-tpm-tools/launcher/spec"
	"github.com/google/go-tpm/legacy/tpm2"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	successRC = 0 // workload successful (no reboot)
	failRC    = 1 // workload or launcher internal failed (no reboot)
	// panic() returns 2
	rebootRC = 3 // reboot
	holdRC   = 4 // hold
	// experimentDataFile defines where the experiment sync output data is expected to be.
	experimentDataFile = "experiment_data"
	// binaryPath contains the path to the experiments binary.
	binaryPath = "/usr/share/oem/confidential_space/confidential_space_experiments"
)

var rcMessage = map[int]string{
	successRC: "workload finished successfully, shutting down the VM",
	failRC:    "workload or launcher error, shutting down the VM",
	rebootRC:  "rebooting VM",
	holdRC:    "VM remains running",
}

var logger *log.Logger
var mdsClient *metadata.Client

var welcomeMessage = "TEE container launcher initiating"
var exitMessage = "TEE container launcher exiting"

func main() {
	var exitCode int // by default exit code is 0
	var err error

	logger = log.Default()
	// log.Default() outputs to stderr; change to stdout.
	log.SetOutput(os.Stdout)
	defer func() {
		os.Exit(exitCode)
	}()

	serialConsole, err := os.OpenFile("/dev/console", os.O_WRONLY, 0)
	if err != nil {
		logger.Printf("failed to open serial console for writing: %v\n", err)
		exitCode = failRC
		logger.Printf("%s, exit code: %d (%s)\n", exitMessage, exitCode, rcMessage[exitCode])
		return
	}
	defer serialConsole.Close()
	logger.SetOutput(io.MultiWriter(os.Stdout, serialConsole))

	logger.Println(welcomeMessage)

	// if err := verifyFsAndMount(); err != nil {
	// 	logger.Printf("failed to verify filesystem and mounts: %v\n", err)
	// 	exitCode = rebootRC
	// 	logger.Printf("%s, exit code: %d (%s)\n", exitMessage, exitCode, rcMessage[exitCode])
	// 	return
	// }

	// Get RestartPolicy and IsHardened from spec
	mdsClient = metadata.NewClient(nil)
	launchSpec, err := spec.GetLaunchSpec(mdsClient)
	if err != nil {
		logger.Printf("failed to get launchspec, make sure you're running inside a GCE VM: %v\n", err)
		// if cannot get launchSpec, exit directly
		exitCode = failRC
		logger.Printf("%s, exit code: %d (%s)\n", exitMessage, exitCode, rcMessage[exitCode])
		return
	}

	if err := os.MkdirAll(launcherfile.HostTmpPath, 0744); err != nil {
		logger.Printf("failed to create %s: %v", launcherfile.HostTmpPath, err)
	}
	experimentsFile := path.Join(launcherfile.HostTmpPath, experimentDataFile)

	args := fmt.Sprintf("-output=%s", experimentsFile)
	err = exec.Command(binaryPath, args).Run()
	if err != nil {
		logger.Printf("failure during experiment sync: %v\n", err)
	}

	e, err := experiments.New(experimentsFile)
	if err != nil {
		logger.Printf("failed to read experiment file: %v\n", err)
		// do not fail if experiment retrieval fails
	}
	launchSpec.Experiments = e

	defer func() {
		// Catch panic to attempt to output to Cloud Logging.
		if r := recover(); r != nil {
			logger.Println("Panic:", r)
			exitCode = 2
		}
		msg, ok := rcMessage[exitCode]
		if ok {
			logger.Printf("%s, exit code: %d (%s)\n", exitMessage, exitCode, msg)
		} else {
			logger.Printf("%s, exit code: %d\n", exitMessage, exitCode)
		}
	}()
	if err = startLauncher(launchSpec, serialConsole); err != nil {
		logger.Println(err)
	}

	exitCode = getExitCode(launchSpec.Hardened, launchSpec.RestartPolicy, err)
}

func getExitCode(isHardened bool, restartPolicy spec.RestartPolicy, err error) int {
	exitCode := 0

	// if in a debug image, will always hold
	if !isHardened {
		return holdRC
	}

	if err != nil {
		switch err.(type) {
		default:
			// non-retryable error
			exitCode = failRC
		case *launcher.RetryableError, *launcher.WorkloadError:
			if restartPolicy == spec.Always || restartPolicy == spec.OnFailure {
				exitCode = rebootRC
			} else {
				exitCode = failRC
			}
		}
	} else {
		// if no error
		if restartPolicy == spec.Always {
			exitCode = rebootRC
		} else {
			exitCode = successRC
		}
	}

	return exitCode
}

func startLauncher(launchSpec spec.LaunchSpec, serialConsole *os.File) error {
	logger.Printf("Launch Spec: %+v\n", launchSpec)
	containerdClient, err := containerd.New(defaults.DefaultAddress)
	if err != nil {
		return &launcher.RetryableError{Err: err}
	}
	defer containerdClient.Close()

	tpm, err := tpm2.OpenTPM("/dev/tpmrm0")
	if err != nil {
		return &launcher.RetryableError{Err: err}
	}
	defer tpm.Close()

	// check AK (EK signing) cert
	gceAk, err := client.GceAttestationKeyECC(tpm)
	if err != nil {
		return err
	}
	if gceAk.Cert() == nil {
		return errors.New("failed to find AKCert on this VM: try creating a new VM or contacting support")
	}
	gceAk.Close()

	token, err := launcher.RetrieveAuthToken(mdsClient)
	if err != nil {
		logger.Printf("failed to retrieve auth token: %v, using empty auth for image pulling\n", err)
	}

	ctx := namespaces.WithNamespace(context.Background(), namespaces.Default)
	r, err := launcher.NewRunner(ctx, containerdClient, token, launchSpec, mdsClient, tpm, logger, serialConsole)
	if err != nil {
		return err
	}
	defer r.Close(ctx)

	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return fmt.Errorf("wgtypes: failed to generate private key: %v", err)
	}
	publicKey := privateKey.PublicKey()

	pskKey, err := wgtypes.GenerateKey()
	if err != nil {
		return fmt.Errorf("wgtypes: failed to generate psk key: %v", err)
	}
	/*
		sudo ip link add dev wg0 type wireguard
		sudo ip address add dev wg0 10.27.0.0/24
		sudo iptables -I INPUT 1 -i wg0 -j ACCEPT
		sudo /sbin/iptables -A INPUT -p udp --dport 51820 -j ACCEPT
		sudo ip link set up dev wg0
	*/

	dur := 10 * time.Second
	var peerConfigs []wgtypes.PeerConfig
	var peerAllowedIPs []net.IPNet
	// peerIP := net.IPv4(146, 148, 82, 244)
	peerIP := net.IPv4(34, 70, 64, 176)
	peerPort := 51820
	peerAllowedIPs = append(peerAllowedIPs, net.IPNet{IP: net.ParseIP("10.27.0.0"), Mask: net.CIDRMask(128, 128)})
	peerConfig := wgtypes.PeerConfig{
		PublicKey:         publicKey,
		PresharedKey:      &pskKey,
		ReplaceAllowedIPs: true,
		Endpoint: &net.UDPAddr{
			IP:   peerIP,
			Port: peerPort,
		},
		PersistentKeepaliveInterval: &dur,
		AllowedIPs:                  peerAllowedIPs,
	}
	peerConfigs = append(peerConfigs, peerConfig)

	listenPort := 51820
	fireWallMask := 0
	cfg := wgtypes.Config{
		PrivateKey:   &privateKey,
		ListenPort:   &listenPort,
		FirewallMark: &fireWallMask,
		ReplacePeers: true,
		Peers:        peerConfigs,
	}

	wgctrlClient, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("wgctrl: failed to create New wgctrl: %v", err)
	}

	device, err := wgctrlClient.Device("wg0")
	if err != nil {
		return fmt.Errorf("wgctrl: failed to get wg0 device: %v", err)
	}

	if err := wgctrlClient.ConfigureDevice(device.Name, cfg); err != nil {
		return fmt.Errorf("failed to configure on %q: %v", device.Name, err)
	}

	_, err = wgctrlClient.Device(device.Name)
	if err != nil {
		return fmt.Errorf("failed to get updated device: %v", err)
	}

	printDevice(device)

	for _, p := range device.Peers {
		printPeer(p)
	}

	return r.Run(ctx)
}

func printDevice(d *wgtypes.Device) {
	const f = `interface: %s (%s)
  public key: %s
  private key: (hidden)
  listening port: %d

`

	fmt.Printf(
		f,
		d.Name,
		d.Type.String(),
		d.PublicKey.String(),
		d.ListenPort)
}

func printPeer(p wgtypes.Peer) {
	const f = `peer: %s
  endpoint: %s
  allowed ips: %s
  latest handshake: %s
  transfer: %d B received, %d B sent

`

	fmt.Printf(
		f,
		p.PublicKey.String(),
		// TODO(mdlayher): get right endpoint with getnameinfo.
		p.Endpoint.String(),
		ipsString(p.AllowedIPs),
		p.LastHandshakeTime.String(),
		p.ReceiveBytes,
		p.TransmitBytes,
	)
}

func ipsString(ipns []net.IPNet) string {
	ss := make([]string, 0, len(ipns))
	for _, ipn := range ipns {
		ss = append(ss, ipn.String())
	}

	return strings.Join(ss, ", ")
}

// verifyFsAndMount checks the partitions/mounts are as expected, based on the command output reported by OS.
// These checks are not a security guarantee.
func verifyFsAndMount() error {
	dmLsOutput, err := exec.Command("dmsetup", "ls").Output()
	if err != nil {
		return fmt.Errorf("failed to call `dmsetup ls`: %v %s", err, string(dmLsOutput))
	}

	dmDevs := strings.Split(string(dmLsOutput), "\n")
	devNameToDevNo := make(map[string]string)
	for _, dmDev := range dmDevs {
		if dmDev == "" {
			continue
		}
		devFields := strings.Fields(dmDev)
		if len(devFields) != 2 {
			continue
		}
		devMajorMinor := strings.ReplaceAll(strings.ReplaceAll(devFields[1], "(", ""), ")", "")
		devNameToDevNo[devFields[0]] = devMajorMinor
	}
	var cryptNo, zeroNo string
	var ok bool
	if _, ok = devNameToDevNo["protected_stateful_partition"]; !ok {
		return fmt.Errorf("failed to find /dev/mapper/protected_stateful_partition: %s", string(dmLsOutput))
	}
	if cryptNo, ok = devNameToDevNo["protected_stateful_partition_crypt"]; !ok {
		return fmt.Errorf("failed to find /dev/mapper/protected_stateful_partition_crypt: %s", string(dmLsOutput))
	}
	if zeroNo, ok = devNameToDevNo["protected_stateful_partition_zero"]; !ok {
		return fmt.Errorf("failed to find /dev/mapper/protected_stateful_partition_zero: %s", string(dmLsOutput))
	}

	dmTableCloneOutput, err := exec.Command("dmsetup", "table", "/dev/mapper/protected_stateful_partition").Output()
	if err != nil {
		return fmt.Errorf("failed to check /dev/mapper/protected_stateful_partition status: %v %s", err, string(dmTableCloneOutput))
	}
	cloneTable := strings.Fields(string(dmTableCloneOutput))
	// https://docs.kernel.org/admin-guide/device-mapper/dm-clone.html
	if len(cloneTable) < 7 {
		return fmt.Errorf("clone table does not match expected format: %s", string(dmTableCloneOutput))
	}
	if cloneTable[2] != "clone" {
		return fmt.Errorf("protected_stateful_partition is not a dm-clone device: %s", string(dmTableCloneOutput))
	}
	if cloneTable[4] != cryptNo {
		return fmt.Errorf("protected_stateful_partition does not have protected_stateful_partition_crypt as a destination device: %s", string(dmTableCloneOutput))
	}
	if cloneTable[5] != zeroNo {
		return fmt.Errorf("protected_stateful_partition protected_stateful_partition_zero as a source device: %s", string(dmTableCloneOutput))
	}

	// Check protected_stateful_partition_crypt is encrypted and is on integrity protection.
	dmTableCryptOutput, err := exec.Command("dmsetup", "table", "/dev/mapper/protected_stateful_partition_crypt").Output()
	if err != nil {
		return fmt.Errorf("failed to check /dev/mapper/protected_stateful_partition_crypt status: %v %s", err, string(dmTableCryptOutput))
	}
	matched := regexp.MustCompile(`integrity:28:aead`).FindString(string(dmTableCryptOutput))
	if len(matched) == 0 {
		return fmt.Errorf("stateful partition is not integrity protected: \n%s", dmTableCryptOutput)
	}
	matched = regexp.MustCompile(`capi:gcm\(aes\)-random`).FindString(string(dmTableCryptOutput))
	if len(matched) == 0 {
		return fmt.Errorf("stateful partition is not using the aes-gcm-random cipher: \n%s", dmTableCryptOutput)
	}

	// Make sure /var/lib/containerd is on protected_stateful_partition.
	findmountOutput, err := exec.Command("findmnt", "/dev/mapper/protected_stateful_partition").Output()
	if err != nil {
		return fmt.Errorf("failed to findmnt /dev/mapper/protected_stateful_partition: %v %s", err, string(findmountOutput))
	}
	matched = regexp.MustCompile(`/var/lib/containerd\s+/dev/mapper/protected_stateful_partition\[/var/lib/containerd\]\s+ext4\s+rw,nosuid,nodev,relatime,commit=30`).FindString(string(findmountOutput))
	if len(matched) == 0 {
		return fmt.Errorf("/var/lib/containerd was not mounted on the protected_stateful_partition: \n%s", findmountOutput)
	}
	matched = regexp.MustCompile(`/var/lib/google\s+/dev/mapper/protected_stateful_partition\[/var/lib/google\]\s+ext4\s+rw,nosuid,nodev,relatime,commit=30`).FindString(string(findmountOutput))
	if len(matched) == 0 {
		return fmt.Errorf("/var/lib/google was not mounted on the protected_stateful_partition: \n%s", findmountOutput)
	}

	// Check /tmp is on tmpfs.
	findmntOutput, err := exec.Command("findmnt", "tmpfs").Output()
	if err != nil {
		return fmt.Errorf("failed to findmnt tmpfs: %v %s", err, string(findmntOutput))
	}
	matched = regexp.MustCompile(`/tmp\s+tmpfs\s+tmpfs`).FindString(string(findmntOutput))
	if len(matched) == 0 {
		return fmt.Errorf("/tmp was not mounted on the tmpfs: \n%s", findmntOutput)
	}

	// Check verity status on vroot and oemroot.
	cryptSetupOutput, err := exec.Command("cryptsetup", "status", "vroot").Output()
	if err != nil {
		return fmt.Errorf("failed to check vroot status: %v %s", err, string(cryptSetupOutput))
	}
	if !strings.Contains(string(cryptSetupOutput), "/dev/mapper/vroot is active and is in use.") {
		return fmt.Errorf("/dev/mapper/vroot was not mounted correctly: \n%s", cryptSetupOutput)
	}
	cryptSetupOutput, err = exec.Command("cryptsetup", "status", "oemroot").Output()
	if err != nil {
		return fmt.Errorf("failed to check oemroot status: %v %s", err, string(cryptSetupOutput))
	}
	if !strings.Contains(string(cryptSetupOutput), "/dev/mapper/oemroot is active and is in use.") {
		return fmt.Errorf("/dev/mapper/oemroot was not mounted correctly: \n%s", cryptSetupOutput)
	}

	return nil
}
