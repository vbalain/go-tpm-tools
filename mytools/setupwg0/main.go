package setupwg0

import (
	"fmt"
	"os/exec"

	"github.com/google/go-tpm-tools/mytools/showwg0"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func SetupWgInterface(wgSubnet string, wgPort int) (*string, error) {
	fmt.Println("SetupWgInterface")
	cmd := exec.Command("/bin/sh", "-c", "sudo ip link add dev wg0 type wireguard")
	fmt.Println("running cmd:", cmd)
	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo ip address add dev wg0 %s", wgSubnet))
	fmt.Println("running cmd:", cmd)
	cmd = exec.Command("/bin/sh", "-c", "sudo iptables -I INPUT 1 -i wg0 -j ACCEPT")
	fmt.Println("running cmd:", cmd)
	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo /sbin/iptables -A INPUT -p udp --dport %d -j ACCEPT", wgPort))
	fmt.Println("running cmd:", cmd)
	cmd = exec.Command("/bin/sh", "-c", "sudo ip link set up dev wg0")
	fmt.Println("running cmd:", cmd)

	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("wgtypes: failed to generate private key: %v", err)
	}
	publicKey := privateKey.PublicKey()
	pskKey, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("wgtypes: failed to generate psk key: %v", err)
	}
	fmt.Println("(1) publicKey, privateKey, pskKey: ", publicKey, privateKey, pskKey)
	fmt.Println("(1s) publicKey: ", publicKey.String())

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
