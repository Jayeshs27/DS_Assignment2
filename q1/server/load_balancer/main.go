package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	// "math/rand"
	"sync"
	"time"

	messageproto "q1/protofiles"

	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

const (
	etcdServerAddr = "localhost:2379"
	etcdKeyPrefix  = "/services/backend/"
	lbServerAddr   = "localhost:50311" // Change per instance
)

var (
	loadBalancingPolicy = 0
)

type BackendServerInfo struct {
	availableServers    []string
	mutexLock            sync.Mutex
	loadStatus           map[string]float32
	rrIndex				 int                       
}

type LoadBalancingServer struct {
	messageproto.UnimplementedLoadBalancingServiceServer
}

type ReportLoadServer struct {
	messageproto.UnimplementedReportLoadServiceServer
}

var backendServersInfo = &BackendServerInfo{}


func (s *LoadBalancingServer) LoadBalancerRPC(ctx context.Context, req *messageproto.LoadBalancerRequest) (*messageproto.LoadBalancerResponse, error) {
	tasktype := req.GetTaskType()
	log.Println("Load Balancer - Task Received from Client:", tasktype)
	backendAddr, err := getBackendServer()
	if err != nil{
		return &messageproto.LoadBalancerResponse{BestServer: ""}, err
	}
	// time.Sleep(5 * time.Second)
	return &messageproto.LoadBalancerResponse{BestServer: backendAddr}, nil
}

func (s *ReportLoadServer) ReportLoadRPC(ctx context.Context, req *messageproto.LoadStatus) (*messageproto.Empty, error) {
	serverAddr, load := req.GetServerAddr(), req.GetLoad()
	// log.Printf("Load Balancer - Load Status Received from Backend Server-%s, Load:%f\n", serverAddr, load)
	
	// for i := 0 ; i < len(backendServersInfo.availableServers) ; i++ {
	// 	if backendServersInfo.availableServers[i] == serverAddr {
	// 		backendServersInfo.backendServersLoad[i] = load
	// 		break
	// 	}
	// }
	
	backendServersInfo.mutexLock.Lock()
	_, exists := backendServersInfo.loadStatus[serverAddr]
	if exists {
		backendServersInfo.loadStatus[serverAddr] = load
	}
	backendServersInfo.mutexLock.Unlock()

	return &messageproto.Empty{}, nil
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
		updatedLoadStatus := make(map[string]float32)
		
		fmt.Print("CPU loads: ")
		for _, serverAddr := range servers {
			load, exists := backendServersInfo.loadStatus[serverAddr]
			if exists {
				updatedLoadStatus[serverAddr] = load
				fmt.Print(" ", load)
			} else {
				updatedLoadStatus[serverAddr] = 0.0
				fmt.Print(" 0.0")
			}
		}
		fmt.Print("\n")

		backendServersInfo.loadStatus = updatedLoadStatus
		backendServersInfo.availableServers = servers
		backendServersInfo.mutexLock.Unlock()

		// log.Printf("Updated backend servers: %v", backendServersInfo.availableServers)
		time.Sleep(2 * time.Second)
	}
}	
func usePickFirst() (string){
	return backendServersInfo.availableServers[0]
}

func useRoundRobin() (string){
	index := backendServersInfo.rrIndex
	for {
		server := backendServersInfo.availableServers[index]
		_, exists := backendServersInfo.loadStatus[server]
		if exists {
			index = (index + 1) % len(backendServersInfo.availableServers)
			break;
		}
		index = (index + 1) % len(backendServersInfo.availableServers)
	}
	reqServer := backendServersInfo.availableServers[index + 1]
	return reqServer
}

func useLeastLoad() (string){
	reqServer := backendServersInfo.availableServers[0]
	minLoad := backendServersInfo.loadStatus[reqServer]
	for _, server := range backendServersInfo.availableServers {
		if backendServersInfo.loadStatus[server] < minLoad {
			minLoad = backendServersInfo.loadStatus[server]
			reqServer = server
		}
	}
	return reqServer
}

func getBackendServer() (string, error){
	// to - do : implement all three policies here
	if len(backendServersInfo.availableServers) == 0{
		return "", fmt.Errorf("no available backend servers")
	}
	if loadBalancingPolicy == 0 {  // Pick First Policy
		return usePickFirst(), nil
	} else if loadBalancingPolicy == 1 {
		return useRoundRobin(), nil
	} else {
		return useLeastLoad(), nil
	}
}

// func handleClientRequests() {

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
// }

func main() {
	args := os.Args[1:]
	if len(args) != 1{
		log.Fatalf("Invalid number of command line arguments, expected 1")
	}
	lbPolicy, err1 := strconv.Atoi(args[0])
	fmt.Println("Policy:", args[0])
	if err1 != nil {
		log.Fatalf("Invalid command line arguments")
	}
	if lbPolicy > 3 || lbPolicy < 0 {
		log.Fatalf("Invalid load balancing policy")
	}
	loadBalancingPolicy = lbPolicy

	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdServerAddr},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}
	defer etcdClient.Close()

	// backendInfo := BackendServerInfo{}

	// to do lb handles -> requests from client as well as backends
	// dynamic server discovery is separate from monitoring availability and load reporting

	go discoverBackends(etcdClient)

	listener, err := net.Listen("tcp", lbServerAddr)
	if err != nil {
		log.Fatalf("Load Balancing Server Failed to listen: %v", err)
	}
	lbServer := grpc.NewServer()

	messageproto.RegisterLoadBalancingServiceServer(lbServer, &LoadBalancingServer{})
	messageproto.RegisterReportLoadServiceServer(lbServer, &ReportLoadServer{})

	log.Println("Load Balancing Server is running on", lbServerAddr)
	if err := lbServer.Serve(listener); err != nil {
		log.Fatalf("Load Balancing Server Failed to serve: %v", err)
	}
	// Handle client requests dynamically
	// handleClientRequests()
}
