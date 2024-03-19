// package main is a program that will start a container with attestation.
package main

import (
	"context"
	"errors"
	"flag"
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
	comm_client "github.com/google/go-tpm-tools/launcher/mytools/communication/client"
	comm_server "github.com/google/go-tpm-tools/launcher/mytools/communication/server"
	"github.com/google/go-tpm-tools/launcher/mytools/configurewg0"
	"github.com/google/go-tpm-tools/launcher/mytools/setupwg0"
	"github.com/google/go-tpm-tools/launcher/spec"
	"github.com/google/go-tpm/legacy/tpm2"
)

var (
	stage = flag.String("stage", "l1", "l1: launcher stage 1: p1: primary instance stage 1; c1: companion instance stage 1/2")
	debug = flag.Bool("debug", false, "debug mode on or off")
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

var wg_port int
var my_ip string
var my_public_key string

func main() {
	flag.Parse()

	if *stage == "p1" {
		fmt.Println("Instance: Primary")

		// Step 1: Setup VPN wireguard interface so the public key is readily available.
		fmt.Println("Step 1: Setup VPN wireguard interface so the public key is readily available.")
		wg_port = 51820
		my_ip = getOutboundIP()
		ppk, err := setupwg0.SetupWgInterface("192.168.0.1/24", wg_port, *debug)
		my_public_key = *ppk
		if err != nil {
			fmt.Printf("%v", err)
			return
		}
		fmt.Println("primary's public key and its IP", my_public_key, my_ip)

		// Step 2: Open TCP port 80
		fmt.Println("Step 2: Open TCP port 80")
		my_port := 80
		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo iptables -A INPUT -p tcp --dport %d -j ACCEPT", my_port))
		fmt.Println("running cmd:", cmd)
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(out, err)
		}

		// Step 3: Start gRPC server(insecure) to exchange public keys.
		// gRPC server(insecure) will be closed after an exchange between primary and companion.
		// comm_server.AddCompanion(companion_instance_id, "")
		fmt.Println("Step 3: Start gRPC server(insecure) to exchange public keys.")
		comm_server.StartInsecureConnectServer(fmt.Sprintf(":%d", my_port), my_public_key)
	} else if *stage == "c1" {
		fmt.Println("Instance: Companion")

		// Step 1: Setup VPN wireguard interface so the public key is readily available.
		fmt.Println("Step 1: Setup VPN wireguard interface so the public key is readily available.")
		wg_port = 51820
		ppk, err := setupwg0.SetupWgInterface("192.168.0.2/24", wg_port, *debug)
		my_public_key = *ppk
		my_ip := getOutboundIP()
		if err != nil {
			fmt.Printf("%v", err)
			return
		}
		fmt.Println("companion's public key and its IP", my_public_key, my_ip)
	}

	if *stage == "l1" {
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

		if err := verifyFsAndMount(); err != nil {
			logger.Printf("failed to verify filesystem and mounts: %v\n", err)
			exitCode = rebootRC
			logger.Printf("%s, exit code: %d (%s)\n", exitMessage, exitCode, rcMessage[exitCode])
			return
		}

		// Get RestartPolicy and IsHardened from spec
		mdsClient = metadata.NewClient(nil)
		launchSpec, err := spec.GetLaunchSpec(mdsClient)
		if err != nil {
			logger.Printf("failed to get launchspec, make sure you're running inside a GCE VM: %v\n", err)
			// if cannot get launchSpec, exit directly
			exitCode = failRC
			logger.Printf("%s, exit code: %d (%s)\n", exitMessage, exitCode, rcMessage[exitCode])
			// return
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
		if err = startLauncher(launchSpec, serialConsole, my_public_key, my_ip); err != nil {
			logger.Println(err)
		}

		exitCode = getExitCode(launchSpec.Hardened, launchSpec.RestartPolicy, err)
	}

	if *stage == "p1" {
		// Step 4: StartLauncher -> container_runner -> companion_manager server should have written companion instance ID/Name and IPs.
		fmt.Println("Step 4: StartLauncher -> container_runner -> companion_manager server should have written companion instance ID/Name and IPs.")

		// Step 5: Start gRPC server to exchange PSK and Certificates etc.
		fmt.Println("Step 5: Start gRPC server to exchange PSK and Certificates etc.")
		comm_server.StartSecureConnectServer(fmt.Sprintf(":%d", wg_port))
	} else if *stage == "c1" {
		// Step 2: Share Companion's public key with the Primary instance.
		fmt.Println("Step 2: Share Companion's public key with the Primary instance.")
		// Primary instance public key, IP etc. should be available from metadata when launching companion instances.
		fmt.Println("Primary instance public key, IP etc. should be available from metadata when launching companion instances.")
		primary_allowed_ips := "192.168.0.1/32"
		primary_ip := "10.128.0.14"                       // to be fetched from metadata
		primary_public_key := "_dummyPrimaryPublicKey123" // should be fetched from metadata
		insecure_server_addr := fmt.Sprintf("%s:80", primary_ip)
		// Ideally, the primary instance's primary key should be part of metadata but unless gcloud APIs are not used,
		// we can do it this way where primary's public key is returned as part of RPC response.
		ppk, err := comm_client.SharePublicKeyWithPrimary(insecure_server_addr, my_public_key)
		primary_public_key = *ppk
		if err != nil {
			fmt.Printf("%v", err)
			return
		}
		fmt.Println("fetch from metadata - primary public key and its IP...", primary_public_key, primary_ip)

		// Step 3: Configure VPN wireguard by adding peer.
		fmt.Println("Step 3: Configure VPN wireguard by adding peer.")
		configurewg0.ConfigurePeer(primary_public_key, primary_ip, wg_port, primary_allowed_ips, true)

		// VPM wireguard subnet decided by us. x.x.x.1 for primary instance and subsequent for companion instances.
		secure_server_addr := fmt.Sprintf("192.168.0.1:%d", wg_port)
		// Step 4: Request PSK key, certificates etc. from server(primary instance)
		fmt.Println("sleeping for 5 secs before requesting PSK")
		time.Sleep(time.Duration(5) * time.Second)
		fmt.Println("Step 4: Request PSK key, certificates etc. from server(primary instance)")
		comm_client.RequestPSK(secure_server_addr)
	}
}

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
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

func startLauncher(launchSpec spec.LaunchSpec, serialConsole *os.File, arg_optional ...string) error {
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

	return r.Run(ctx, arg_optional...)
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
