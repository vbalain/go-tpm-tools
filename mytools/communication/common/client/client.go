package comm_client

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/google/go-tpm-tools/mytools/communication/common/proto/connect"
	"github.com/google/go-tpm-tools/mytools/showwg0"
)

// var (
// 	secureAddr   = flag.String("addr", "192.168.0.1:80", "VPN Wireguard subnet/server address in the format of host:port")
// 	insecureAddr = flag.String("addr2", "10.128.0.14:80", "server address in the format of host:port")
// )

func RequestPSK(serverAddr string) {
	fmt.Println("Exchange Pre-shared Keys and Certs")
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	fmt.Println("client: dialing to: ", serverAddr)
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewSecureConnectClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	req := pb.PskRequest{}
	defer cancel()
	res, err := client.GetPSK(ctx, &req)
	if err != nil {
		fmt.Printf("failed to receive response from server: %v", err)
	}
	fmt.Println("client(secure): received PSK key: ", *(res.Key))
	showwg0.ShowConfig()
}

func SharePublicKeyWithPrimary(serverAddr string) (*string, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	fmt.Println("client: dialing to: ", serverAddr)
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	client := pb.NewInsecureConnectClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	myPublicKey := "aqMpWik5gw5rUhOToxCB2UEuI3JhQWOi8kVuxcO8u9R="
	myInstanceId := "companion1"
	myIp := getOutboundIP()
	fmt.Println("client: request: public key: ", myPublicKey)
	fmt.Println("client: request: instance id: ", myInstanceId)
	fmt.Println("client: request: instance ip: ", myIp)
	req := pb.ExchangeRequest{Key: &myPublicKey, InstanceId: &myInstanceId}
	res, err := client.ExchangePublicKeys(ctx, &req)
	defer conn.Close()
	defer cancel()
	if err != nil {
		return nil, fmt.Errorf("failed to receive response from server: %v", err)
	}
	fmt.Println("client: received public key: ", *(res.Key))

	return res.Key, nil
}

// Get preferred outbound ip of this machine
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

// func main() {
// 	flag.Parse()
// 	RequestPublicKeyFromPrimary(*insecureAddr)
// 	RequestPSK(*secureAddr)
// }
