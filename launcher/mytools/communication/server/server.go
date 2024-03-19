package comm_server

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"

	pb "github.com/google/go-tpm-tools/launcher/mytools/communication/proto/connect"
	"github.com/google/go-tpm-tools/launcher/mytools/configurewg0"
	"github.com/google/go-tpm-tools/launcher/mytools/showwg0"
)

var grpcInsecureServer *grpc.Server
var grpcSecureServer *grpc.Server
var primaryPublicKey string

type SecureConnectServer struct {
	pb.UnimplementedSecureConnectServer
}

type InsecureConnectServer struct {
	pb.UnimplementedInsecureConnectServer
}

func (s *InsecureConnectServer) ExchangePublicKeys(ctx context.Context, req *pb.ExchangeRequest) (*pb.ExchangeResponse, error) {
	result := true
	fmt.Println("server: request: public key: ", *(req.Key))
	fmt.Println("server: request: instance id: ", *(req.InstanceId))

	// Read from file(written by companion_manager server) companion IP via instance ID
	peer_public_key := *(req.Key)
	peer_ip := "10.128.0.14"

	fmt.Println("Step 5: Configure VPN wireguard by adding peer/companion.")
	wg_port := 51820
	configurewg0.ConfigurePeer(peer_public_key, peer_ip, wg_port, "192.168.0.2/32", true)

	StopInsecureServerAfter(10)

	key := primaryPublicKey
	fmt.Println("server: response: ending public key: ", key)
	return &pb.ExchangeResponse{Success: &result, Key: &key}, nil
}

func newInsecureConnectServer() *InsecureConnectServer {
	s := &InsecureConnectServer{}
	return s
}

func StartInsecureConnectServer(addr string, ppk_optional ...string) {
	if len(ppk_optional) > 0 {
		primaryPublicKey = ppk_optional[0]
	} else {
		primaryPublicKey = ""
	}

	fmt.Println("StartInsecureConnectServer")
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("server is listening to: ", addr)
	fmt.Println("...")
	fmt.Println("...")
	var opts []grpc.ServerOption
	grpcInsecureServer = grpc.NewServer(opts...)
	pb.RegisterInsecureConnectServer(grpcInsecureServer, newInsecureConnectServer())
	grpcInsecureServer.Serve(lis)
}

func (s *SecureConnectServer) GetPSK(ctx context.Context, request *pb.PskRequest) (*pb.PskResponse, error) {
	result := true
	key := "xwHuPhl5gw5rUhOToxCB2UEuI3JhQWOi8kVuxcI4inY=" // dummy key string for now
	fmt.Println("server: sending PSK key: ", key)
	showwg0.ShowConfig()
	return &pb.PskResponse{Success: &result, Key: &key}, nil
}

func newSecureConnectServer() *SecureConnectServer {
	s := &SecureConnectServer{}
	return s
}

func StartSecureConnectServer(addr string) {
	fmt.Println("StartSecureConnectServer")
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("server is listening to: ", addr)
	fmt.Println("...")
	fmt.Println("...")
	var opts []grpc.ServerOption
	grpcSecureServer = grpc.NewServer(opts...)
	pb.RegisterSecureConnectServer(grpcSecureServer, newSecureConnectServer())
	grpcSecureServer.Serve(lis)
}

func StopInsecureServerAfter(delay int) {
	fmt.Println("server: stopping insecure server after", delay, "seconds")
	time.Sleep(time.Duration(delay) * time.Second)

	grpcInsecureServer.GracefulStop()
	fmt.Println("server: stopped insecure server")
}
