package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "github.com/google/go-tpm-tools/mytools/communication/proto/connect"
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

func (s *ConnectServer) GetPSK(ctx context.Context, request *pb.PskRequest) (*pb.PskResponse, error) {
	result := true
	key := "xwHuPhl5gw5rUhOToxCB2UEuI3JhQWOi8kVuxcI4inY=" // dummy key string for now
	return &pb.PskResponse{Success: &result, Key: &key}, nil
}

func newServer() *ConnectServer {
	s := &ConnectServer{}
	return s
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterConnectServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}
