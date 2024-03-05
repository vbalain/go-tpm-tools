package main

import (
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	// "github.com/google/go-sev-guest/tools/lib/cmdline"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var (
	peer_public_key = flag.String("public_key", "1asi7uykjhasAkuh1asi7uykjhasAkuh", "peer public key")
	peer_ip         = flag.String("ip", "34.171.172.249", "svm-instance-1") // 10.128.0.8/32
	peer_port       = flag.Int("port", 51820, "port no.")
	peer_allowed_ip = flag.String("allowed_ip", "10.99.0.0/32", "subnet")
)

func main() {
	flag.Parse()
	fmt.Println("flags: public key, ip, port, allowed ip: ", *peer_public_key, *peer_ip, *peer_port, *peer_allowed_ip)
	dur := 25 * time.Second
	var peerConfigs []wgtypes.PeerConfig
	var peerAllowedIPs []net.IPNet
	var peerPublicKey wgtypes.Key
	peerPublicKey, err := wgtypes.ParseKey(*peer_public_key)
	if err != nil {
		fmt.Printf("wgtypes: Unable to parse key: %v", err)
		return
	}
	peerIP := net.ParseIP(*peer_ip)
	// peerIP := net.ParseIP("34.70.64.176")  // svm-instance-2: 10.128.0.7/32
	peerPort := *peer_port
	ip, ipnet, err := net.ParseCIDR(*peer_allowed_ip)
	if err != nil {
		fmt.Printf("net: ParseCIDR invalid: %v", err)
		return
	}
	peerAllowedIPs = append(peerAllowedIPs, net.IPNet{IP: ip, Mask: ipnet.Mask})

	peerConfig := wgtypes.PeerConfig{
		PublicKey:         peerPublicKey,
		ReplaceAllowedIPs: true,
		Endpoint: &net.UDPAddr{
			IP:   peerIP,
			Port: peerPort,
		},
		PersistentKeepaliveInterval: &dur,
		AllowedIPs:                  peerAllowedIPs,
	}
	peerConfigs = append(peerConfigs, peerConfig)

	newCfg := wgtypes.Config{
		ReplacePeers: true,
		Peers:        peerConfigs,
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

	if err := wgctrlClient.ConfigureDevice(device.Name, newCfg); err != nil {
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
