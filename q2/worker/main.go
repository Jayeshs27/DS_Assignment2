package main

import (
	"context"
	// "fmt"
	"log"
	"net"
	// "time"
	"bufio"
	"fmt"
	"hash/fnv"
	"os"
	"strings"
	"strconv"

	mapReducepb "q2/protofiles"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

)

// Merges intermediate files and directly stores reduced output
func mergeAndReduce(numMappers int, reducerId int) {
	wordCounts := make(map[string]int) // Dictionary to store aggregated counts

	// Read all intermediate files for reducer r
	for m := 0; m < numMappers; m++ {
		inputFile := fmt.Sprintf("map-%d-%d.txt", m, reducerId)
		file, err := os.Open(inputFile)
		if err != nil {
			log.Printf("Skipping missing file: %s\n", inputFile)
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			parts := strings.Fields(scanner.Text())
			if len(parts) != 2 {
				continue // Ignore malformed lines
			}

			word := parts[0]
			count, err := strconv.Atoi(parts[1])
			if err != nil {
				log.Printf("Invalid count for word %s: %v\n", word, err)
				continue
			}

			// Aggregate counts
			wordCounts[word] += count
		}
		file.Close()
	}
	// Write reduced output directly to final reduce-%d.txt
	outputFile, err := os.Create(fmt.Sprintf("reduce-%d.txt", reducerId))
	if err != nil {
		log.Fatalf("Failed to create reduce output file: %v", err)
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	for word, count := range wordCounts {
		fmt.Fprintf(writer, "%s %d\n", word, count)
	}
	writer.Flush()

	log.Printf("Reducer %d completed: Output stored in reduce-%d.txt\n", reducerId, reducerId)
	
}

func processReduceTask(numMappers int, reducerId int) {
	mergeAndReduce(numMappers, reducerId)

	conn, err := grpc.NewClient(masterServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Error while connecting to Master :%v", err)
	}
	masterclient := mapReducepb.NewSubmitResultServiceClient(conn)
	err = sendReduceResults(masterclient)
}



// Get reducer index using hash function
func getReducerIndex(word string, numReduce int) int {
	hash := fnv.New32a()
	hash.Write([]byte(word))
	return int(hash.Sum32()) % numReduce
}

// Mapper function
func processMapTask(mapperID int, inputFile string, numReduce int) {
	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Println("Error opening input file:", err)
		return
	}
	defer file.Close()

	// Create intermediate files for each reducer
	intermediateFiles := make([]*os.File, numReduce)
	for i := 0; i < numReduce; i++ {
		filename := fmt.Sprintf("map-%d-%d.txt", mapperID, i) // Unique for each Mapper
		intermediateFiles[i], _ = os.Create(filename)
		defer intermediateFiles[i].Close()
	}

	// Process input file
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		words := strings.Fields(scanner.Text()) // Split line into words
		for _, word := range words {
			reducerIndex := getReducerIndex(word, numReduce)
			fmt.Fprintf(intermediateFiles[reducerIndex], "%s 1\n", word) // Write safely
		}
	}

	conn, err := grpc.NewClient(masterServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Error while connecting to Master :%v", err)
	}
	masterclient := mapReducepb.NewSubmitResultServiceClient(conn)
	err = sendMapResults(masterclient)

}


const (
	workerAddr = "localhost:23111"
	masterServerAddr = "localhost:50411"
)

type WorkerServiceServer struct {
	mapReducepb.UnimplementedWorkerServiceServer
}

// Map Task - Returns Immediately, Runs in Background
func (s *WorkerServiceServer) MapRPC(ctx context.Context, req *mapReducepb.MapRequest) (*mapReducepb.MapResponse, error) {
	inputFile, numReduce, mapperId := req.GetInputfile(), req.GetNumReduce(), req.GetMapperId()
	log.Println("Worker - Task Received: Mapper, inputFile:", inputFile, ", numReduce:", numReduce)

	// Run Map Task in a separate goroutine
	go func() {
		processMapTask(int(mapperId), inputFile, int(numReduce))
	}()

	// Return immediately to prevent blocking the master
	return &mapReducepb.MapResponse{}, nil
}

// Reduce Task - Returns Immediately, Runs in Background
func (s *WorkerServiceServer) ReduceRPC(ctx context.Context, req *mapReducepb.ReduceRequest) (*mapReducepb.ReduceResponse, error) {
	numMappers, reducerId := req.GetNumMappers(), req.GetReducerId()
	log.Println("Worker - Task Received: Reducer, reducerId:", reducerId)

	// Run Reduce Task in a separate goroutine
	go func() {
		processReduceTask(int(numMappers), int(reducerId))
	}()

	// Return immediately to prevent blocking the master
	return &mapReducepb.ReduceResponse{}, nil
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

// Simulated Map Task
// func processMapTask(inputFile string, numReduce int32) {
// 	log.Printf("Processing Map Task for file %s with %d reducers...\n", inputFile, numReduce)
// 	time.Sleep(1 * time.Second) // Simulate computation time
// 	log.Printf("Map Task Completed for file %s\n", inputFile)
	
// 	conn, err := grpc.NewClient(masterServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	if err != nil {
// 		log.Fatalf("Error while connecting to Master :%v", err)
// 	}
// 	masterclient := mapReducepb.NewSubmitResultServiceClient(conn)
// 	err = sendMapResults(masterclient)
// }

// Simulated Reduce Task

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: go run master.go <port>")
	}
	// Get directory and numReduce from command-line arguments
	workerAddr := fmt.Sprintf("localhost:%s",os.Args[1])
	
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
