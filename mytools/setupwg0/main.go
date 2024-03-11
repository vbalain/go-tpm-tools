package main

import (
	"fmt"

	"github.com/google/go-tpm-tools/mytools/showwg0"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func SetupWgInterface() (string, error) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", fmt.Errorf("wgtypes: failed to generate private key: %v", err)
	}
	publicKey := privateKey.PublicKey()
	pskKey, err := wgtypes.GenerateKey()
	if err != nil {
		return "", fmt.Errorf("wgtypes: failed to generate psk key: %v", err)
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
		return "", fmt.Errorf("wgctrl: failed to create New wgctrl: %v", err)
	}

	device, err := wgctrlClient.Device("wg0")
	if err != nil {
		return "", fmt.Errorf("wgctrlClient: failed to get wg0 device: %v", err)
	}

	if err := wgctrlClient.ConfigureDevice(device.Name, cfg); err != nil {
		return "", fmt.Errorf("wgctrlClient: failed to configure on %q: %v", device.Name, err)
	}

	showwg0.ShowConfig()

	return publicKey.String(), nil
}

func main() {
	_, err := SetupWgInterface()
	if err != nil {
		fmt.Printf("setupwg0: failure: %v", err)
	}
}
