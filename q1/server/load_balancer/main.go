package main

import (
	"context"
	"log"
	"net"
	"fmt"
	// "math/rand"
	"sync"
	"time"

	"google.golang.org/grpc"
	"go.etcd.io/etcd/client/v3"
	messageproto "q1/protofiles"
)

const (
	etcdServerAddr = "localhost:2379"
	etcdKeyPrefix  = "/services/backend/"
	lbServerAddr   = "localhost:50311" // Change per instance
)

// var (
// 	mu           sync.Mutex
// 	backendNodes []string
// )

type BackendServerInfo struct {
	availableServers    []string
	mutexLock            sync.Mutex
	backendServersLoad    []int
}

type LoadBalancingServer struct {
	messageproto.UnimplementedLoadBalancingServiceServer
}

var backendServersInfo = &BackendServerInfo{}


func (s *LoadBalancingServer) LoadBalancerRPC(ctx context.Context, req *messageproto.LoadBalancerRequest) (*messageproto.LoadBalancerResponse, error) {
	message := req.GetLoad()
	log.Println("Load Balancer - Message Received from Client:", message)
	backendAddr, err := getBackendServer()
	if err != nil{
		return &messageproto.LoadBalancerResponse{BestServer: ""}, err
	}
	// time.Sleep(5 * time.Second)
	return &messageproto.LoadBalancerResponse{BestServer: backendAddr}, nil
}

func discoverBackends(client *clientv3.Client) {
	for {
		resp, err := client.Get(context.Background(), etcdKeyPrefix, clientv3.WithPrefix())
		if err != nil {
			log.Printf("Failed to fetch backend servers: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var servers []string
		for _, kv := range resp.Kvs {
			servers = append(servers, string(kv.Value))
		}

		backendServersInfo.mutexLock.Lock()
		backendServersInfo.availableServers = servers
		backendServersInfo.mutexLock.Unlock()

		// log.Printf("Updated backend servers: %v", backendServersInfo.availableServers)
		time.Sleep(5 * time.Second)
	}
}	

func getBackendServer() (string, error){
	// to - do : implement all three policies here
	if len(backendServersInfo.availableServers) == 0{
		return "", fmt.Errorf("no available backend servers")
	}
	return backendServersInfo.availableServers[0], nil
}

func handleClientRequests() {
	listener, err := net.Listen("tcp", lbServerAddr)
	if err != nil {
		log.Fatalf("Load Balancing Server Failed to listen: %v", err)
	}
	lbServer := grpc.NewServer()
	messageproto.RegisterLoadBalancingServiceServer(lbServer, &LoadBalancingServer{})

	log.Println("Load Balancing Server is running on", lbServerAddr)
	if err := lbServer.Serve(listener); err != nil {
		log.Fatalf("Load Balancing Server Failed to serve: %v", err)
	}

	// for {
	// 	backend := getBackend()
	// 	if backend == "" {
	// 		log.Println("No available backend servers")
	// 		time.Sleep(3 * time.Second)
	// 		continue
	// 	}

	// 	conn, err := grpc.Dial(backend, grpc.WithInsecure())
	// 	if err != nil {
	// 		log.Printf("Failed to connect to backend %s: %v", backend, err)
	// 		time.Sleep(3 * time.Second)
	// 		continue
	// 	}
	// 	defer conn.Close()

	// 	// client := messageproto.NewMessageServiceClient(conn)
	// 	// req := &messageproto.MessageRequest{Message: "Hello from Load Balancer"}
	// 	// resp, err := client.MessageRPC(context.Background(), req)
	// 	// if err != nil {
	// 	// 	log.Printf("RPC failed to backend %s: %v", backend, err)
	// 	// 	time.Sleep(3 * time.Second)
	// 	// 	continue
	// 	// }

	// 	// log.Printf("Response from backend %s: %s", backend, resp.GetResponse())
	// 	// time.Sleep(3 * time.Second) // Simulating continuous client requests
	// }
}

func main() {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdServerAddr},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}
	defer etcdClient.Close()

	// backendInfo := BackendServerInfo{}

	go discoverBackends(etcdClient)

	// Handle client requests dynamically
	handleClientRequests()
}
