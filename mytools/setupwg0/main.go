package main

import (
	"fmt"
	"net"
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func main() {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		fmt.Printf("wgtypes: failed to generate private key: %v", err)
		return
	}
	publicKey := privateKey.PublicKey()
	pskKey, err := wgtypes.GenerateKey()
	if err != nil {
		fmt.Printf("wgtypes: failed to generate psk key: %v", err)
		return
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
		fmt.Printf("wgctrl: failed to create New wgctrl: %v", err)
		return
	}

	device, err := wgctrlClient.Device("wg0")
	if err != nil {
		fmt.Printf("wgctrlClient: failed to get wg0 device: %v", err)
		return
	}

	if err := wgctrlClient.ConfigureDevice(device.Name, cfg); err != nil {
		fmt.Printf("wgctrlClient: failed to configure on %q: %v", device.Name, err)
		return
	}

	newDevice, err := wgctrlClient.Device(device.Name)
	if err != nil {
		fmt.Printf("wgctrlClient: failed to get updated device: %v", err)
		return
	}

	printDevice(newDevice)

	for _, p := range newDevice.Peers {
		printPeer(p)
	}
}

func printDevice(d *wgtypes.Device) {
	const f = `interface: %s (%s)
  public key: %s
  private key: %s
  listening port: %d
  peers count: %d`

	fmt.Printf(
		f,
		d.Name,
		d.Type.String(),
		d.PublicKey,
		d.PrivateKey,
		d.ListenPort,
		len(d.Peers))
	fmt.Println("**********")
}

func printPeer(p wgtypes.Peer) {
	const f = `peer: %s
  endpoint: %s
  allowed ips: %s
  latest handshake: %s
  transfer: %d B received, %d B sent`

	fmt.Printf(
		f,
		p.PublicKey,
		// TODO(mdlayher): get right endpoint with getnameinfo.
		p.Endpoint.String(),
		ipsString(p.AllowedIPs),
		p.LastHandshakeTime.String(),
		p.ReceiveBytes,
		p.TransmitBytes,
	)
	fmt.Println("**********")
}

func ipsString(ipns []net.IPNet) string {
	ss := make([]string, 0, len(ipns))
	for _, ipn := range ipns {
		ss = append(ss, ipn.String())
	}

	return strings.Join(ss, ", ")
}
