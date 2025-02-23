package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	mapReducepb "q2/protofiles"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	masterServerAddr = "localhost:50411"
	baseWorkerPort   = 23000 // Base port for workers
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

func (s *MasterServer) MapResultRPC(ctx context.Context, req *mapReducepb.MapResult) (*mapReducepb.MapResultResponse, error) {
	s.mu.Lock()
	s.mapCounter++
	log.Printf("Master - Received Map task completion (%d/%d)", s.mapCounter, s.totalMappers)

	// Only mark Map phase as done when all Mappers finish
	if s.mapCounter == s.totalMappers {
		s.mapDone <- true
		log.Println("Master - All Map tasks completed, proceeding to Reduce phase")
	}
	s.mu.Unlock()

	return &mapReducepb.MapResultResponse{}, nil
}

func (s *MasterServer) ReduceResultRPC(ctx context.Context, req *mapReducepb.ReduceResult) (*mapReducepb.ReduceResultResponse, error) {
	s.mu.Lock()
	s.reduceCounter++
	log.Printf("Master - Received Reduce task completion (%d/%d)", s.reduceCounter, s.totalReducers)

	// Only mark Reduce phase as done when all Reducers finish
	if s.reduceCounter == s.totalReducers {
		s.reduceDone <- true
		log.Println("Master - All Reduce tasks completed, MapReduce Job Done!")
	}
	s.mu.Unlock()

	return &mapReducepb.ReduceResultResponse{}, nil
}

func main() {
	// Ensure correct usage
	if len(os.Args) < 3 {
		log.Fatalf("Usage: go run master.go <directory> <numReduce>")
	}

	// Get directory and numReduce from command-line arguments
	directory := os.Args[1]
	numReduce, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalf("Invalid number of reducers: %v", err)
	}

	// Read files from the directory
	files, err := os.ReadDir(directory)
	if err != nil {
		log.Fatalf("Failed to read directory: %v", err)
	}

	// Start the gRPC master server
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

	// Start workers for each input file
	workerPorts := make([]int, len(files))
	for i, file := range files {
		if file.IsDir() {
			continue // Skip directories
		}
		// Get an available port for the worker
		workerPort, err := getAvailablePort()
		if err != nil {
			log.Fatalf("Failed to get Free port %v",err)
		}
		workerPorts[i] = workerPort

		// Start worker process
		cmd := exec.Command("go", "run", "worker/main.go", strconv.Itoa(workerPort))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Start()
		if err != nil {
			log.Fatalf("Failed to start worker %d: %v", i, err)
		}
		log.Printf("Started worker %d on port %d\n", i, workerPort)
	}
	time.Sleep(2 * time.Second)
	// Connect to workers and assign Map tasks
	for i, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(directory, file.Name())
		workerAddr := fmt.Sprintf("localhost:%d", workerPorts[i])
		conn, err := grpc.NewClient(workerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("Failed to connect to worker %d: %v", i, err)
		}
		defer conn.Close()
		client := mapReducepb.NewWorkerServiceClient(conn)

		log.Printf("Master - Sending Map request to Worker %d", i)
		_, err = client.MapRPC(context.Background(), &mapReducepb.MapRequest{
			Inputfile: filePath,
			NumReduce: int32(numReduce),
			MapperId:  int32(i),
		})
		if err != nil {
			log.Fatalf("Failed to send Map request to Worker %d: %v", i, err)
		}
	}

	// Wait for all Map tasks to complete
	log.Println("Waiting for Mapper...")
	<-master.mapDone
	// Determine the number of reducers to spawn
	numMappers := len(files)
	numReducers := numReduce
	if numReducers > numMappers {
		extraWorkers := numReducers - numMappers
		log.Printf("Spawning %d extra workers for reducers", extraWorkers)
		for i := 0; i < extraWorkers; i++ {
			workerPort, err := getAvailablePort()
			if err != nil {
				log.Fatalf("Failed to get free port: %v", err)
			}
			workerPorts = append(workerPorts, workerPort)

			cmd := exec.Command("go", "run", "worker.go", strconv.Itoa(workerPort))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Start()
			if err != nil {
				log.Fatalf("Failed to start extra worker %d: %v", i, err)
			}
			log.Printf("Started extra worker %d on port %d for reducers\n", numMappers+i, workerPort)
		}
	}

	// Connect to workers and assign Reduce tasks
	for i := 0; i < numReducers; i++ {
		workerAddr := fmt.Sprintf("localhost:%d", workerPorts[i%numMappers])
		conn, err := grpc.NewClient(workerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("Failed to connect to worker for Reduce task %d: %v", i, err)
		}
		defer conn.Close()
		client := mapReducepb.NewWorkerServiceClient(conn)

		log.Printf("Master - Sending Reduce request to Worker %d", i)
		_, err = client.ReduceRPC(context.Background(), &mapReducepb.ReduceRequest{
			NumMappers: int32(numMappers),
			ReducerId:  int32(i),
		})
		if err != nil {
			log.Fatalf("Failed to send Reduce request to Worker %d: %v", i, err)
		}
	}
	// Wait for all Reduce tasks to complete
	log.Println("Waiting for Mapper...")
	<-master.reduceDone

	log.Println("Master - MapReduce Job Completed Successfully!")
}
