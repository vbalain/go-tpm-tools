package comm_server

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"

	pb "github.com/google/go-tpm-tools/mytools/communication/common/proto/connect"
	"github.com/google/go-tpm-tools/mytools/showwg0"
)

// var (
// 	// tls      = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
// 	// certFile = flag.String("cert_file", "", "The TLS cert file")
// 	// keyFile  = flag.String("key_file", "", "The TLS key file")
// 	addr = flag.String("addr", ":80", "server address")
// )

var companions = make(map[string]string) // key-value pair: instance_id-public key
var grpcServer *grpc.Server
var pkExchangeDone bool

func GetCompanionPK(key string) (*string, error) {
	val, ok := companions[key]
	if !ok {
		return nil, fmt.Errorf("companion key not found")
	}
	return &val, nil
}

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

	_, ok := companions[*(req.InstanceId)]
	if !ok {
		result = false
		return &pb.ExchangeResponse{Success: &result}, nil
	}
	companions[*(req.InstanceId)] = *(req.Key)
	key := "xwHuPhl5gw5rUhOToxCB2UEuI3JhQWOi8kVuxcI4inY=" // dummy key string for now
	fmt.Println("server: response: ending public key: ", key)
	pkExchangeDone = true
	go StopSecureServerAfter(10)
	return &pb.ExchangeResponse{Success: &result, Key: &key}, nil
}

func newInsecureConnectServer() *InsecureConnectServer {
	s := &InsecureConnectServer{}
	return s
}

func StartInsecureConnectServer(addr string) {
	fmt.Println("StartInsecureConnectServer")
	pkExchangeDone = false
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("server is listening to: ", addr)
	fmt.Println("...")
	fmt.Println("...")
	var opts []grpc.ServerOption
	grpcServer = grpc.NewServer(opts...)
	pb.RegisterInsecureConnectServer(grpcServer, newInsecureConnectServer())
	grpcServer.Serve(lis)
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
	if pkExchangeDone {
		fmt.Println("StartSecureConnectServer")
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		fmt.Println("server is listening to: ", addr)
		fmt.Println("...")
		fmt.Println("...")
		var opts []grpc.ServerOption
		grpcServer = grpc.NewServer(opts...)
		pb.RegisterSecureConnectServer(grpcServer, newSecureConnectServer())
		grpcServer.Serve(lis)
		return
	}

	fmt.Println("PK exchange wasn't done")
}

func StopServer() {
	grpcServer.GracefulStop()
}

func AddCompanion(instance_id string, instance_ip string) {
	companions[instance_id] = instance_ip
}

func StopSecureServerAfter(delay int) {
	fmt.Println("server: stopping insecure server after", delay, "seconds")
	time.Sleep(time.Duration(delay) * time.Second)

	grpcServer.GracefulStop()
	fmt.Println("server: stopped insecure server")
	// StartSecureConnectServer(":80")
}
