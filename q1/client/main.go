package main

import (
	"context"
	"fmt"
	"log"
	"os"
	lbproto "q1/protofiles"
	"strconv"

	"google.golang.org/grpc"
	// "google.golang.org/grpc/credentials/insecure"
)

const (
	lbServerAddr = "localhost:50319"
)

func sendRequestToLoadBalancer(client lbproto.LoadBalancingServiceClient, tasktype int) (string, error){
	req := &lbproto.LoadBalancerRequest{TaskType: int32(tasktype)}

	resp, err := client.LoadBalancerRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("Error while calling LoadBalancerRPC: %v", err)
		return "", err
	}

	fmt.Println("Response From Load Balancing Server: ", resp.GetBestServer())
	return resp.GetBestServer(), nil
}

func sendRequestToBackendServer(client lbproto.BackendServiceClient, taskType int, num int64){
	req := &lbproto.BackendRequest{TaskType: int32(taskType), Num: num}

	resp, err := client.BackendRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("Error while calling RPC")
	}

	fmt.Println("Response From Backend Server: ", resp.GetOutput())
}

func main(){
	args := os.Args[1:]
	if len(args) != 1{
		log.Fatalf("Invalid command line arguments, expected 1")
	}
	tasktype, err1 := strconv.Atoi(args[0])
	if err1 != nil {
		log.Fatalf("Invalid command line arguments")
	}
	if tasktype > 3 || tasktype < 0 {
		log.Fatalf("Invalid tasktype")
	}

	conn, err := grpc.Dial(lbServerAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Client - Could not connet to Load Balancing server")
	}
	defer conn.Close()

	lbClient := lbproto.NewLoadBalancingServiceClient(conn)

	backendAddr, err := sendRequestToLoadBalancer(lbClient, tasktype)
	if err != nil{
		log.Fatalf("Client - Error while requesting for backend server: %v", err)
	}
	conn2, err2 := grpc.Dial(backendAddr, grpc.WithInsecure())
	if err2 != nil {
		log.Fatalf("Client - Could not connet to Backend server with Addr: %s, error: %v", backendAddr, err)
	}
	
	backendClient := lbproto.NewBackendServiceClient(conn2)

	if tasktype == 0{
		sendRequestToBackendServer(backendClient, tasktype, 1e9)
	} else if tasktype == 1{
		sendRequestToBackendServer(backendClient, tasktype, 1e6)
	}else{
		sendRequestToBackendServer(backendClient, tasktype, 45)
	}
}