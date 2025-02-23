package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"
	"bytes"
	"os/exec"
	"strconv"
	"strings"
	
	messageproto "q1/protofiles"
	// "github.com/shirou/gopsutil/v3/cpu"
	// "github.com/shirou/gopsutil/v3/process"
	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)
	
func getAvaliablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0") // ":0" lets OS pick a free port
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func parseCmdOutput(output string) (float64, error){
	lines := strings.Split(output, "\n")
	if len(lines) < 7 {
		log.Fatalf("Unexpected output from top command")
	}
	processInfo := strings.Fields(lines[len(lines) - 2])
	if len(processInfo) < 9 {
		log.Fatalf("Failed to parse CPU usage")
	} 
	cpuUsage, err := strconv.ParseFloat(processInfo[8], 64)
	if err != nil {
		return 0.0, err
	}
	return cpuUsage, nil
}
func getCpuUsage()(float64, error){
	pid := os.Getpid()
	cmd := exec.Command("top", "-b", "-n", "1", "-p", strconv.Itoa(pid))
	var out bytes.Buffer
	cmd.Stdout = &out
	
	err := cmd.Run()
	if err != nil {
		return 0.0, err
	} 
	cpuUsage, err := parseCmdOutput(out.String())
	if err != nil{
		return 0.0, err
	}
	fmt.Println("Pid:",pid , "CPU Usage:", cpuUsage, "%")
	return cpuUsage, nil
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
	lbServerAddr   = "localhost:50311"
	ttl            = 2                   // TTL in seconds for etcd lease
)

type BackendServer struct {
	messageproto.UnimplementedBackendServiceServer
}

func (s *BackendServer) BackendRPC(ctx context.Context, req *messageproto.BackendRequest) (*messageproto.BackendResponse, error) {
	tasktype, num := req.GetTaskType(), req.GetNum()
	log.Printf("Task Received : %d, N : %d\n", tasktype, num)
	result := executeTask(tasktype, num)
	return &messageproto.BackendResponse{Output: result}, nil
}

func ReportLoadStatus(client messageproto.ReportLoadServiceClient, serverAddr string){
	for{
		load, err := getCpuUsage()
		if err != nil {
			log.Fatalf("Error while getting CPU load: %v", err)
		}
		loadStatus := &messageproto.LoadStatus{ServerAddr:serverAddr, Load: float32(load)}
		_, err = client.ReportLoadRPC(context.Background(), loadStatus)
		if err != nil{
			log.Fatalf("Error while send Load Status: %v", err)
		}
		time.Sleep(1 * time.Second)
	}
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := etcdClient.Status(ctx, etcdServerAddr)
	if err != nil {
		log.Fatalf("Etcd is not running or unreachable: %v", err)
	}

	log.Println("Etcd is running! Version:", resp.Version)
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

	// cpuLoad, err := getCpuLoad()
	// if err != nil {
	// 	log.Fatalf("Error while getting CPU usage: %v", err)
	// }
	// log.Println("Current CPU Usage: ", cpuLoad)

	conn, err := grpc.Dial(lbServerAddr, grpc.WithInsecure())
	reportLoadClient := messageproto.NewReportLoadServiceClient(conn)
	go ReportLoadStatus(reportLoadClient, serverAddr)
	
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	backendServer := grpc.NewServer()
	messageproto.RegisterBackendServiceServer(backendServer, &BackendServer{})

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
