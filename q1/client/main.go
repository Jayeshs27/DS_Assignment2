package main

import (
	"context"
	"fmt"
	"log"

	messageproto "q1/protofiles"

	"google.golang.org/grpc"
	// "google.golang.org/grpc/credentials/insecure"
)

const (
	lbServerAddr = "localhost:50311"
)

func sendRequestToLoadBalancer(client messageproto.LoadBalancingServiceClient) (string, error){
	req := &messageproto.LoadBalancerRequest{Load: 300}

	resp, err := client.LoadBalancerRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("Error while calling LoadBalancerRPC: %v", err)
		return "", err
	}

	fmt.Println("Response From Load Balancing Server: ", resp.GetBestServer())
	return resp.GetBestServer(), nil
}

func sendRequestToBackendServer(client messageproto.BackendServiceClient){
	req := &messageproto.BackendRequest{Load: 300}

	resp, err := client.BackendRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("Error while calling RPC")
	}

	fmt.Println("Response From Backend Server: ", resp.GetFeedback())
}

func main(){
	conn, err := grpc.Dial(lbServerAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Client - Could not connet to Load Balancing server")
	}
	defer conn.Close()

	lbClient := messageproto.NewLoadBalancingServiceClient(conn)

	backendAddr, err := sendRequestToLoadBalancer(lbClient)
	if err != nil{
		log.Fatalf("Client - Error while requesting for backend server: %v", err)
	}

	conn2, err2 := grpc.Dial(backendAddr, grpc.WithInsecure())
	if err2 != nil {
		log.Fatalf("Client - Could not connet to Backend server with Addr: %s, error: %v", backendAddr, err)
	}

	backendClient := messageproto.NewBackendServiceClient(conn2)
	sendRequestToBackendServer(backendClient)
}