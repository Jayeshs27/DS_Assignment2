package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	// "math/rand"
	"sync"
	"time"

	lbproto "q1/protofiles"

	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

const (
	etcdServerAddr = "localhost:2379"
	etcdKeyPrefix  = "/services/backend/"
	lbServerAddr   = "localhost:50319" 
)

var (
	loadBalancingPolicy = "PF"
)

type BackendServerInfo struct {
	availableServers    []string
	mutexLock            sync.Mutex
	loadStatus           map[string]float32
	rrIndex				 int                       
}

type LoadBalancingServer struct {
	lbproto.UnimplementedLoadBalancingServiceServer
}

type ReportLoadServer struct {
	lbproto.UnimplementedReportLoadServiceServer
}

var backendServersInfo = &BackendServerInfo{}


func (s *LoadBalancingServer) LoadBalancerRPC(ctx context.Context, req *lbproto.LoadBalancerRequest) (*lbproto.LoadBalancerResponse, error) {
	tasktype := req.GetTaskType()
	log.Println("Load Balancer - Task Received from Client:", tasktype)
	backendAddr, err := getBackendServer()
	if err != nil{
		return &lbproto.LoadBalancerResponse{BestServer: ""}, err
	}
	return &lbproto.LoadBalancerResponse{BestServer: backendAddr}, nil
}

func (s *ReportLoadServer) ReportLoadRPC(ctx context.Context, req *lbproto.LoadStatus) (*lbproto.Empty, error) {
	serverAddr, load := req.GetServerAddr(), req.GetLoad()
	// log.Printf("Load Balancer - Load Status Received from Backend Server-%s, Load:%f\n", serverAddr, load)
	backendServersInfo.mutexLock.Lock()
	_, exists := backendServersInfo.loadStatus[serverAddr]
	if exists {
		backendServersInfo.loadStatus[serverAddr] = load
	}
	backendServersInfo.mutexLock.Unlock()

	return &lbproto.Empty{}, nil
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
		
		for _, serverAddr := range servers {
			load, exists := backendServersInfo.loadStatus[serverAddr]
			if exists {
				updatedLoadStatus[serverAddr] = load
			} else {
				updatedLoadStatus[serverAddr] = 0.0
			}
		}

		backendServersInfo.loadStatus = updatedLoadStatus
		backendServersInfo.availableServers = servers
		backendServersInfo.mutexLock.Unlock()

		// log.Printf("Updated backend servers: %v", backendServersInfo.availableServers)
		time.Sleep(2 * time.Second)
	}
}	

func usePickFirst() (string){
	backendServersInfo.mutexLock.Lock()
	reqServer := backendServersInfo.availableServers[0]
	backendServersInfo.mutexLock.Unlock()
	return reqServer
}

func useRoundRobin() (string){
	backendServersInfo.mutexLock.Lock()

	index := backendServersInfo.rrIndex
	num_servers := len(backendServersInfo.availableServers)

	for {
		server := backendServersInfo.availableServers[index]
		_, exists := backendServersInfo.loadStatus[server]
		if exists {
			index = (index + 1) % num_servers
			break;
		}
		index = (index + 1) % num_servers
	}

	reqServer := backendServersInfo.availableServers[index]
	backendServersInfo.rrIndex = (index + 1) % num_servers

	backendServersInfo.mutexLock.Unlock()
	return reqServer
}

func useLeastLoad() (string){
	backendServersInfo.mutexLock.Lock()

	reqServer := backendServersInfo.availableServers[0]
	minLoad := backendServersInfo.loadStatus[reqServer]

	for _, server := range backendServersInfo.availableServers {
		if backendServersInfo.loadStatus[server] < minLoad {
			minLoad = backendServersInfo.loadStatus[server]
			reqServer = server
		}
	}

	backendServersInfo.mutexLock.Unlock()
	return reqServer
}

func getBackendServer() (string, error){
	if len(backendServersInfo.availableServers) == 0{
		return "", fmt.Errorf("no available backend servers")
	}

	if loadBalancingPolicy == "PF" {  // Pick First Policy
		return usePickFirst(), nil
	} else if loadBalancingPolicy == "RR" {  // Round Robin Policy
		return useRoundRobin(), nil
	} else {
		return useLeastLoad(), nil   // Least Load Policy
	}
}

func main() {
	args := os.Args[1:]

	if len(args) == 1 {
		if args[0] != "RR" && args[0] != "PF" && args[0] != "LL"{
			log.Fatalf("Invalid load balancing policy. Use 'RR' or 'PF' or 'LL'.")
		}
		loadBalancingPolicy = args[0] 
	}

	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdServerAddr},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}
	defer etcdClient.Close()

	go discoverBackends(etcdClient)

	listener, err := net.Listen("tcp", lbServerAddr)
	if err != nil {
		log.Fatalf("Load Balancing Server Failed to listen: %v", err)
	}
	defer listener.Close()

	lbServer := grpc.NewServer()

	lbproto.RegisterLoadBalancingServiceServer(lbServer, &LoadBalancingServer{})
	lbproto.RegisterReportLoadServiceServer(lbServer, &ReportLoadServer{})

	log.Println("Load Balancing Server is running on", lbServerAddr)
	if err := lbServer.Serve(listener); err != nil {
		log.Fatalf("Load Balancing Server Failed to serve: %v", err)
	}
}
