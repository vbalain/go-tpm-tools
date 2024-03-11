package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/google/go-tpm-tools/mytools/communication/proto/connect"
	"github.com/google/go-tpm-tools/mytools/showwg0"
)

var (
	serverAddr = flag.String("addr", "192.168.0.1:51820", "The server address in the format of host:port")
)

func RequestPSK(serverAddr string) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewConnectClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	req := pb.PskRequest{}
	defer cancel()
	res, err := client.GetPSK(ctx, &req)
	if err != nil {
		fmt.Printf("failed to receive response from server: %v", err)
	}
	fmt.Println("client: received PSK key: ", *(res.Key))
	showwg0.ShowConfig()
}

func main() {
	flag.Parse()
	RequestPSK(*serverAddr)
}
