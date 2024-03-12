package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "github.com/google/go-tpm-tools/mytools/communication/insecure/proto/connect"
)

var (
	// tls      = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	// certFile = flag.String("cert_file", "", "The TLS cert file")
	// keyFile  = flag.String("key_file", "", "The TLS key file")
	port = flag.Int("port", 51820, "The server port")
)

type ConnectServer struct {
	pb.UnimplementedConnectServer
}

func (s *ConnectServer) ExchangePublicKeys(ctx context.Context, req *pb.ExchangeRequest) (*pb.ExchangeResponse, error) {
	result := true
	key := "xwHuPhl5gw5rUhOToxCB2UEuI3JhQWOi8kVuxcI4inY=" // dummy key string for now
	fmt.Println("server: response: ending public key: ", key)
	fmt.Println("server: request: public key: ", *(req.Key))
	fmt.Println("server: request: instance id: ", *(req.InstanceId))
	fmt.Println("server: request: ip: ", *(req.Ip))
	return &pb.ExchangeResponse{Success: &result, Key: &key}, nil
}

func newServer() *ConnectServer {
	s := &ConnectServer{}
	return s
}

func StartServer(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("server is listening to port: ", port)
	fmt.Println("...")
	fmt.Println("...")
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterConnectServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}

func main() {
	flag.Parse()
	StartServer(*port)
}
