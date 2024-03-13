package configurewg0

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/go-tpm-tools/mytools/showwg0"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func ConfigurePeer(publicKey string, ip string, port int, allowedIps string, refreshPeers bool) error {
	fmt.Println("ConfigurePeer")
	var peerConfigs []wgtypes.PeerConfig
	var peerAllowedIPs []net.IPNet
	var peerPublicKey wgtypes.Key
	dur := 25 * time.Second
	peerPublicKey, err := wgtypes.ParseKey(publicKey)
	if err != nil {
		return fmt.Errorf("wgtypes: Unable to parse key: %v", err)
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
			return fmt.Errorf("net: ParseCIDR invalid: %v", err)
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
		return fmt.Errorf("wgctrl: failed to create New wgctrl: %v", err)
	}

	device, err := wgctrlClient.Device("wg0")
	if err != nil {
		return fmt.Errorf("wgctrlClient: failed to get wg0 device: %v", err)
	}

	if err := wgctrlClient.ConfigureDevice(device.Name, newCfg); err != nil {
		return fmt.Errorf("wgctrlClient: failed to configure on %q: %v", device.Name, err)
	}

	showwg0.ShowConfig()
	return nil
}
