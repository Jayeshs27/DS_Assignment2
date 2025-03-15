package main

import (
	"context"
	"fmt"
	// "go/scanner"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	mapReducepb "q2/protofiles"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TaskType int

const (
	COUNT_FREQUENCY TaskType = 0
	INVERTED_INDEX TaskType = 1
)

const (
	masterServerAddr = "localhost:50411"
)

func getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0") // ":0" lets OS pick a free port
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}


type MasterServer struct {
	mapReducepb.UnimplementedWorkerServiceServer
	mapReducepb.UnimplementedSubmitResultServiceServer

	totalMappers int
	totalReducers int
	mapCounter int
	reduceCounter int
	mu sync.Mutex
	mapDone chan bool
	reduceDone chan bool
}

func NewMasterServer(totalMappers int, totalReducers int) *MasterServer {
	return &MasterServer{
		totalMappers: totalMappers,
		totalReducers: totalReducers,
		mapCounter: 0,
		reduceCounter: 0,
		mapDone: make(chan bool, 1),
		reduceDone: make(chan bool, 1),
	}
}

func sendMapRequest(workerAddr string, filePath string, numReduce int, id int, tasktype TaskType){
	conn, err := grpc.NewClient(workerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to worker %d: %v", id, err)
	}
	defer conn.Close()
	client := mapReducepb.NewWorkerServiceClient(conn)

	log.Printf("Master - Sending Map request to Worker %d", id)
	_, err = client.MapRPC(context.Background(), &mapReducepb.MapRequest{
		Inputfile: filePath,
		NumReduce: int32(numReduce),
		MapperId:  int32(id),
		TaskType: int32(tasktype),
	})
	if err != nil {
		log.Fatalf("Failed to send Map request to Worker %d: %v", id, err)
	}
}

func sendReduceRequest(workerAddr string, numMappers int, id int, tasktype TaskType){
	conn, err := grpc.NewClient(workerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to worker for Reduce task %d: %v", id, err)
	}
	defer conn.Close()
	client := mapReducepb.NewWorkerServiceClient(conn)

	log.Printf("Master - Sending Reduce request to Worker %d", id)
	_, err = client.ReduceRPC(context.Background(), &mapReducepb.ReduceRequest{
		NumMappers: int32(numMappers),
		ReducerId:  int32(id),
		TaskType: int32(tasktype),
	})
	if err != nil {
		log.Fatalf("Failed to send Reduce request to Worker %d: %v", id, err)
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Invalid Arguments, Usage: go run master.go <data-directory> <numReduce>")
	}
	directory := os.Args[1]
	numReduce, err := strconv.Atoi(os.Args[2])

	if err != nil {
		log.Fatalf("Invalid number of reducers: %v", err)
	}
	files, err := os.ReadDir(directory)
	if err != nil {
		log.Fatalf("Failed to read directory: %v", err)
	}

	var taskType int
	fmt.Print("Enter Task Type(0-Count Word Frequency, 1-Inverted Index):")
	fmt.Scan(&taskType)
	if taskType > 2 || taskType < 0 {
		fmt.Println("Invalid taskType, Usage: enter 0 for count frequency, 1 for inverted index")
	}

	listener, err := net.Listen("tcp", masterServerAddr)
	if err != nil {
		log.Fatalf("Master Server Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	master := NewMasterServer(len(files), numReduce)
	mapReducepb.RegisterWorkerServiceServer(grpcServer, master)
	mapReducepb.RegisterSubmitResultServiceServer(grpcServer, master)

	log.Println("Master Server is running on", masterServerAddr)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Master Server Failed to serve: %v", err)
		}
	}()

	workerPorts := make([]int, len(files))
	for i, file := range files {
		if file.IsDir() {
			continue 
		}
		workerPort, err := getAvailablePort()
		if err != nil {
			log.Fatalf("Failed to get Free port %v",err)
		}
		workerPorts[i] = workerPort

		cmd := exec.Command("go", "run", "./worker", strconv.Itoa(workerPort))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			log.Fatalf("Failed to start worker %d: %v", i, err)
		}
		log.Printf("Started worker %d on port %d\n", i, workerPort)
		time.Sleep(500 * time.Millisecond)
	}

	// TODO here wait for mapper to iniatized 

	for i, file := range files {
		if file.IsDir() {
			continue
		}
		filePath := filepath.Join(directory, file.Name())
		workerAddr := fmt.Sprintf("localhost:%d", workerPorts[i])
		sendMapRequest(workerAddr, filePath, numReduce, i, TaskType(taskType))
	}

	log.Println("Waiting for Mapper...")
	<-master.mapDone

	numMappers := len(files)
	numReducers := numReduce
	if numReducers > numMappers {
		extraWorkers := numReducers - numMappers
		log.Printf("Spawning %d extra workers for reducers", extraWorkers)
		for i := range extraWorkers{
			workerPort, err := getAvailablePort()
			if err != nil {
				log.Fatalf("Failed to get free port: %v", err)
			}
			workerPorts = append(workerPorts, workerPort)
	
			cmd := exec.Command("go", "run", "./worker", strconv.Itoa(workerPort))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Start()
			if err != nil {
				log.Fatalf("Failed to start extra worker %d: %v", i, err)
			}
			log.Printf("Started extra worker %d on port %d for reducers\n", numMappers+i, workerPort)
			time.Sleep(500 * time.Millisecond)
		}
	}

	for i := range numReducers {
		workerAddr := fmt.Sprintf("localhost:%d", workerPorts[i % numMappers])
		sendReduceRequest(workerAddr, numMappers, i, TaskType(taskType))
	}

	log.Println("Waiting for Reducer...")
	<-master.reduceDone

	log.Println("Master - MapReduce Job Completed Successfully!")

	for _, Ports := range workerPorts {
		workerAddr := fmt.Sprintf("localhost:%d", Ports)
		conn, err := grpc.NewClient(workerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("Failed to connect to worker %s: %v", workerAddr, err)
			continue
		}
		defer conn.Close()
	
		client := mapReducepb.NewWorkerServiceClient(conn)
		_, err = client.ExitRPC(context.Background(), &mapReducepb.ExitRequest{})
		if err != nil {
			log.Printf("Failed to shut down worker %s: %v", workerAddr, err)
		} 
	}
}
