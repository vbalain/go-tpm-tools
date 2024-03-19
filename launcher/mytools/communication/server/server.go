package comm_server

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"google.golang.org/grpc"

	pb "github.com/google/go-tpm-tools/launcher/mytools/communication/proto/connect"
	"github.com/google/go-tpm-tools/launcher/mytools/configurewg0"
	"github.com/google/go-tpm-tools/launcher/mytools/showwg0"
)

var grpcDefaultServer *grpc.Server
var grpcWgServer *grpc.Server
var primaryPublicKey string

type DefaultConnectServer struct {
	pb.UnimplementedDefaultConnectServer
}

func newDefaultConnectServer() *DefaultConnectServer {
	s := &DefaultConnectServer{}
	return s
}

func (s *DefaultConnectServer) SharePublicKey(ctx context.Context, req *pb.ExchangeRequest) (*pb.ExchangeResponse, error) {
	result := true
	fmt.Println("default server: request: public key: ", *(req.Key))
	fmt.Println("default server: request: instance id: ", *(req.InstanceId))

	// Read from file(written by companion_manager server) companion IP via instance ID
	peer_public_key := *(req.Key)
	peer_ip := "10.128.0.14"

	fmt.Printf("\nSTEP 5: Configure VPN wireguard by adding peer/companion.\n\n")
	wg_port := 51820
	configurewg0.ConfigurePeer(peer_public_key, peer_ip, wg_port, "192.168.0.2/32", true)

	key := primaryPublicKey
	fmt.Println("default server: sending public key: ", key)
	return &pb.ExchangeResponse{Success: &result, Key: &key}, nil
}

func StartDefaultConnectServer(addr string, ppk_optional ...string) {
	if len(ppk_optional) > 0 {
		primaryPublicKey = ppk_optional[0]
	} else {
		primaryPublicKey = ""
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("default server is listening to: ", addr)
	fmt.Println("...")
	fmt.Println("...")
	var opts []grpc.ServerOption
	grpcDefaultServer = grpc.NewServer(opts...)
	pb.RegisterDefaultConnectServer(grpcDefaultServer, newDefaultConnectServer())
	grpcDefaultServer.Serve(lis)
}

type WgConnectServer struct {
	pb.UnimplementedWgConnectServer
}

func newWgConnectServer() *WgConnectServer {
	s := &WgConnectServer{}
	return s
}

func (s *WgConnectServer) GetPSK(ctx context.Context, request *pb.PskRequest) (*pb.PskResponse, error) {
	result := true
	pskKey, err := wgtypes.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("wgtypes: failed to generate psk key: %v", err)
	}
	key := pskKey.String()
	fmt.Println("wg server: sending PSK key: ", key)
	showwg0.ShowConfig()
	return &pb.PskResponse{Success: &result, Key: &key}, nil
}

func StartWgConnectServer(addr string) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("wg server is listening to: ", addr)
	fmt.Println("...")
	fmt.Println("...")
	var opts []grpc.ServerOption
	grpcWgServer = grpc.NewServer(opts...)
	pb.RegisterWgConnectServer(grpcWgServer, newWgConnectServer())
	grpcWgServer.Serve(lis)
}

func StopDefaultServerAfter(delay int) {
	fmt.Println("default server: stopping server after", delay, "seconds")
	time.Sleep(time.Duration(delay) * time.Second)

	grpcDefaultServer.GracefulStop()
	fmt.Println("default server: stopped server")
}
