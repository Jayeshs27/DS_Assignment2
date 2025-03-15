package main

import (
	"context"
	"time"
	// "fmt"
	"log"
	"os"

	mapReducepb "q2/protofiles"
)


func (s *WorkerServiceServer) MapRPC(ctx context.Context, req *mapReducepb.MapRequest) (*mapReducepb.MapResponse, error) {
	inputFile, numReduce, mapperId, taskType := req.GetInputfile(), req.GetNumReduce(), req.GetMapperId(), req.GetTaskType()
	log.Println("Worker - Task Received: Mapper, inputFile:", inputFile, ", numReduce:", numReduce)
	go func() {
		processMapTask(int(mapperId), inputFile, int(numReduce), TaskType(taskType))
	}()
	return &mapReducepb.MapResponse{}, nil
}


func (s *WorkerServiceServer) ReduceRPC(ctx context.Context, req *mapReducepb.ReduceRequest) (*mapReducepb.ReduceResponse, error) {
	numMappers, reducerId, taskType := req.GetNumMappers(), req.GetReducerId(), req.GetTaskType()
	log.Println("Worker - Task Received: Reducer, reducerId:", reducerId)
	go func() {
		processReduceTask(int(numMappers), int(reducerId), TaskType(taskType))
	}()
	return &mapReducepb.ReduceResponse{}, nil
}

func (s *WorkerServiceServer) ExitRPC(ctx context.Context, req *mapReducepb.ExitRequest) (*mapReducepb.ExitResponse, error) {
	log.Println("WorkerAddr:",workerAddr,", Received Exit Request, Exiting...")
	go func() {
		time.Sleep(time.Second)
		os.Exit(0)
	}()
	return &mapReducepb.ExitResponse{}, nil
}