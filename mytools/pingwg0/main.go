package main

import (
	"flag"
	"fmt"
	"net"
	"os/exec"
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var (
	ping_ip = flag.String("ip", "10.128.0.8", "instance-svm-1") // instance-svm-2 10.128.0.7
)

func main() {
	flag.Parse()

	wgctrlClient, err := wgctrl.New()
	if err != nil {
		fmt.Printf("wgctrl: failed to create New wgctrl: %v", err)
		return
	}

	ping(*ping_ip)

	device, err := wgctrlClient.Device("wg0")
	if err != nil {
		fmt.Printf("wgctrlClient: failed to get wg0 device: %v", err)
		return
	}

	printDevice(device)

	for _, p := range device.Peers {
		printPeer(p)
	}
}

func ping(ip string) {
	out, _ := exec.Command("ping", ip, "-c 5", "-i 3", "-w 10").Output()
	fmt.Println("ping output: ", out)
	if strings.Contains(string(out), "Destination Host Unreachable") {
		fmt.Println("TANGO DOWN")
	} else {
		fmt.Println("IT'S ALIVEEE")
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
