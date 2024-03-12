package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/google/go-tpm-tools/mytools/communication/insecure/proto/connect"
)

var (
	serverAddr = flag.String("addr", "10.128.0.14:51821", "The server address in the format of host:port")
)

func RequestPublicKeyFromPrimary(serverAddr string) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	fmt.Println("client: dialing to: ", serverAddr)
	conn, err := grpc.Dial(serverAddr, opts...)
	fmt.Println("vaibhav 1")
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	fmt.Println("vaibhav 2")
	defer conn.Close()
	client := pb.NewConnectClient(conn)
	fmt.Println("vaibhav 3")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	myPublicKey := "aqMpWik5gw5rUhOToxCB2UEuI3JhQWOi8kVuxcO8u9R="
	myInstanceId := "companion123"
	myIp := GetOutboundIP()
	fmt.Println("client: request: public key: ", myPublicKey)
	fmt.Println("client: request: instance id: ", myInstanceId)
	fmt.Println("client: request: ip: ", myIp)
	fmt.Println("vaibhav 4")
	req := pb.ExchangeRequest{Key: &myPublicKey, InstanceId: &myInstanceId, Ip: &myIp}
	defer cancel()
	res, err := client.ExchangePublicKeys(ctx, &req)
	fmt.Println("vaibhav 5")
	if err != nil {
		fmt.Printf("failed to receive response from server: %v", err)
	}
	fmt.Println("client: received public key: ", *(res.Key))
	fmt.Println("vaibhav 6")
}

// Get preferred outbound ip of this machine
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func main() {
	flag.Parse()
	RequestPublicKeyFromPrimary(*serverAddr)
}
