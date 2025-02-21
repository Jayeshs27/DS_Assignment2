package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"go.etcd.io/etcd/client/v3"
	messageproto "q1/protofiles"
)

func getAvaliablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0") // ":0" lets OS pick a free port
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func getAvaliableAddress() (string, error) {
	port, err := getAvaliablePort()
	if err != nil {
		return "", err
	}
	addr := fmt.Sprintf("localhost:%v", port)
	return addr, nil
}

const (
	etcdServerAddr = "localhost:2379"
	etcdKeyPrefix  = "/services/backend/"
	ttl            = 2                   // TTL in seconds for etcd lease
)

type server struct {
	messageproto.UnimplementedBackendServiceServer
}

func (s *server) BackendRPC(ctx context.Context, req *messageproto.BackendRequest) (*messageproto.BackendResponse, error) {
	message := req.GetLoad()
	fmt.Println("Message Received from Client:", message)
	resp := "Hi Client, This is Backend Server"
	return &messageproto.BackendResponse{Feedback: resp}, nil
}

// Register server with etcd
func registerWithEtcd(client *clientv3.Client, leaseID clientv3.LeaseID, serverAddr string) {
	etcdKey := fmt.Sprintf("%s%s", etcdKeyPrefix, serverAddr)
	_, err := client.Put(context.Background(), etcdKey, serverAddr, clientv3.WithLease(leaseID))
	if err != nil {
		log.Fatalf("Failed to register backend: %v", err)
	}
	log.Printf("Registered %s with etcd", serverAddr)
}

// Maintain heartbeat with etcd
func keepAlive(client *clientv3.Client, leaseID clientv3.LeaseID) {
	ch, err := client.KeepAlive(context.Background(), leaseID)
	if err != nil {
		log.Fatalf("Failed to keep alive: %v", err)
	}
	// fmt.Println(ch)
	for range ch {}  // to consumed keepalive responses
}

func main() {
	// Initialize etcd client
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdServerAddr},
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}
	defer etcdClient.Close()

	// Create a lease
	leaseResp, err := etcdClient.Grant(context.Background(), ttl)
	if err != nil {
		log.Fatalf("Failed to create lease: %v", err)
	}
	leaseID := leaseResp.ID

	serverAddr, err := getAvaliableAddress()
	if err != nil {
		log.Fatalf("Failed to get avaliable address: %v", err)
	}

	// Register and keep alive
	registerWithEtcd(etcdClient, leaseID, serverAddr)
	log.Println("Backend server registered with etcd")

	go keepAlive(etcdClient, leaseID)


	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	backendServer := grpc.NewServer()
	messageproto.RegisterBackendServiceServer(backendServer, &server{})

	log.Println("Backend gRPC server is running on", serverAddr)
	if err := backendServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

/*

docker pull gcr.io/etcd-development/etcd:v3.5.9

docker run -d -p 2379:2379 --name etcd-instance gcr.io/etcd-development/etcd:v3.5.9 \
    etcd --advertise-client-urls http://localhost:2379 \
         --listen-client-urls http://0.0.0.0:2379


Check current running containers

docker ps -a

To remove docker container

docker rm etcd-instance

docker stop etcd-instance

*/
