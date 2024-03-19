package setupwg0

import (
	"fmt"
	"os/exec"

	"github.com/google/go-tpm-tools/launcher/mytools/showwg0"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var debugEnabled bool

func runShellCommand(cmd string) {
	execCmd := exec.Command("/bin/sh", "-c", cmd)
	fmt.Println("running shell command:", execCmd)
	out, err := execCmd.CombinedOutput()
	if debugEnabled && err != nil {
		fmt.Println(out, err)
	}
}

func initCommands(wgSubnet string, wgPort int) {
	runShellCommand("sudo ip link add dev wg0 type wireguard")
	runShellCommand(fmt.Sprintf("sudo ip address add dev wg0 %s", wgSubnet))
	runShellCommand("sudo iptables -I INPUT 1 -i wg0 -j ACCEPT")
	runShellCommand(fmt.Sprintf("sudo /sbin/iptables -A INPUT -p udp --dport %d -j ACCEPT", wgPort))
	runShellCommand("sudo ip link set up dev wg0")
}

func SetupWgInterface(wgSubnet string, wgPort int, debug bool) (*string, error) {
	fmt.Println("SetupWgInterface")
	debugEnabled = debug

	initCommands(wgSubnet, wgPort)

	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("wgtypes: failed to generate private key: %v", err)
	}
	publicKey := privateKey.PublicKey()
	listenPort := 51820
	fireWallMask := 0
	cfg := wgtypes.Config{
		PrivateKey:   &privateKey,
		ListenPort:   &listenPort,
		FirewallMark: &fireWallMask,
		ReplacePeers: false,
	}

	wgctrlClient, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("wgctrl: failed to create New wgctrl: %v", err)
	}

	device, err := wgctrlClient.Device("wg0")
	if err != nil {
		return nil, fmt.Errorf("wgctrlClient: failed to get wg0 device: %v", err)
	}

	if err := wgctrlClient.ConfigureDevice(device.Name, cfg); err != nil {
		return nil, fmt.Errorf("wgctrlClient: failed to configure on %q: %v", device.Name, err)
	}

	showwg0.ShowConfig()
	key := publicKey.String()

	return &key, nil
}
