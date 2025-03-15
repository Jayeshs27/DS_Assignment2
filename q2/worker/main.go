package main

import (
	"context"
	// "fmt"
	"log"
	"net"
	"fmt"
	"os"

	mapReducepb "q2/protofiles"
	"google.golang.org/grpc"
)

const (
	masterServerAddr = "localhost:50411"
)
var (
	workerAddr = ""
)

type TaskType int

const (
	COUNT_FREQUENCY TaskType = 0
	INVERTED_INDEX TaskType = 1
)


type WorkerServiceServer struct {
	mapReducepb.UnimplementedWorkerServiceServer
}

func sendMapResults(client mapReducepb.SubmitResultServiceClient) (error){
	req := &mapReducepb.MapResult{}
	_, err := client.MapResultRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("Error while calling RPC")
	}
	return nil
}

func sendReduceResults(client mapReducepb.SubmitResultServiceClient) (error){
	req := &mapReducepb.ReduceResult{}
	_, err := client.ReduceResultRPC(context.Background(), req)
	if err != nil {
		log.Fatalf("Error while calling RPC")
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: go run master.go <port>")
	}
	workerAddr = fmt.Sprintf("localhost:%s",os.Args[1])
	
	listener, err := net.Listen("tcp", workerAddr)
	if err != nil {
		log.Fatalf("Worker Server Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	mapReducepb.RegisterWorkerServiceServer(grpcServer, &WorkerServiceServer{})

	log.Println("Worker Server is running on", workerAddr)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Worker Server Failed to serve: %v", err)
	}
}
