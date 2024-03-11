package main

import (
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/go-tpm-tools/mytools/showwg0"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var (
	peer_public_key  = flag.String("public_key", "1asi7uykjhasAkuh1asi7uykjhasAkuh", "peer public key")
	peer_ip          = flag.String("ip", "10.128.0.8", "instance-svm-1") // instance-svm-2 10.128.0.7
	peer_port        = flag.Int("port", 51820, "port no.")
	peer_allowed_ips = flag.String("allowed_ips", "", "subnet") // 10.99.0.0/32,10.99.0.1/32 etc.
	replace_peers    = flag.Bool("replace_peers", true, "replace peers in wg0 config")
)

func main() {
	flag.Parse()
	fmt.Println("flags: public key, ip, port, allowed ips: ")
	ConfigurePeer(*peer_public_key, *peer_ip, *peer_port, *peer_allowed_ips, *replace_peers)
}

func ConfigurePeer(publicKey string, ip string, port int, allowedIps string, refreshPeers bool) {
	var peerConfigs []wgtypes.PeerConfig
	var peerAllowedIPs []net.IPNet
	var peerPublicKey wgtypes.Key
	dur := 25 * time.Second
	peerPublicKey, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		fmt.Printf("wgtypes: Unable to parse key: %v", err)
		return
	}
	peerIP := net.ParseIP(ip)
	peerPort := port

	ips := allowedIps
	ipList := strings.Split(ips, ",")
	for _, element := range ipList {
		if element == "" {
			continue
		}
		ip, ipnet, err := net.ParseCIDR(element)
		if err != nil {
			fmt.Printf("net: ParseCIDR invalid: %v", err)
			return
		}
		peerAllowedIPs = append(peerAllowedIPs, net.IPNet{IP: ip, Mask: ipnet.Mask})
	}

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
		ReplacePeers: refreshPeers,
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

	showwg0.ShowConfig()
}
