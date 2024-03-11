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
)

var (
	serverAddr = flag.String("addr", "192.168.0.1:51820", "The server address in the format of host:port")
)

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.Dial(*serverAddr, opts...)
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
	fmt.Println("received PSK res.key: ", res.Key)
	key := *(res.Key)
	fmt.Println("received PSK key: ", key)
}
