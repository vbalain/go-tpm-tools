package comm_client

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/google/go-tpm-tools/launcher/mytools/communication/proto/connect"
	"github.com/google/go-tpm-tools/launcher/mytools/showwg0"
)

func RequestPSK(serverAddr string) {
	fmt.Println("(wg)Exchange Pre-shared Keys and Certs")
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	fmt.Println("client(wg): dialing to: ", serverAddr)
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewWgConnectClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	req := pb.PskRequest{}
	defer cancel()
	res, err := client.GetPSK(ctx, &req)
	if err != nil {
		fmt.Printf("failed to receive response from secure server: %v", err)
	}
	fmt.Println("client(wg): respone: received PSK key: ", *(res.Key))
	showwg0.ShowConfig()
}

func SharePublicKeyWithPrimary(serverAddr string, myPublicKey string) (*string, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	fmt.Println("client: dialing to: ", serverAddr)
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	client := pb.NewDefaultConnectClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	myInstanceId := "companion1"
	myIp := getOutboundIP()
	fmt.Println("client: request: public key: ", myPublicKey)
	fmt.Println("client: request: instance id: ", myInstanceId)
	fmt.Println("client: request: instance ip: ", myIp)
	req := pb.ExchangeRequest{Key: &myPublicKey, InstanceId: &myInstanceId}
	res, err := client.SharePublicKey(ctx, &req)
	defer conn.Close()
	defer cancel()
	if err != nil {
		return nil, fmt.Errorf("failed to receive response from insecure server: %v", err)
	}
	fmt.Println("client: respone: received public key: ", *(res.Key))

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
