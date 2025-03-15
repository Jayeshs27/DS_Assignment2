package main

import (
	"context"
	"log"

	mapReducepb "q2/protofiles"
)


// func (masterServer *MasterServer) RegisterRPC(ctx context.Context, req *mapReducepb.RegisterRequest) (*mapReducepb.RegisterResponse, error) {
// 	masterServer.mu.Lock()
// 	masterServer.mapCounter++
// 	log.Printf("Master - Received Map task completion (%d/%d)", masterServer.mapCounter, masterServer.totalMappers)

// 	if masterServer.mapCounter == masterServer.totalMappers {
// 		masterServer.mapDone <- true
// 		log.Println("Master - All Map tasks completed, proceeding to Reduce phase")
// 	}
// 	masterServer.mu.Unlock()

// 	return &mapReducepb.RegisterResponse{}, nil
// }

func (masterServer *MasterServer) MapResultRPC(ctx context.Context, req *mapReducepb.MapResult) (*mapReducepb.MapResultResponse, error) {
	masterServer.mu.Lock()
	masterServer.mapCounter++
	log.Printf("Master - Received Map task completion (%d/%d)", masterServer.mapCounter, masterServer.totalMappers)

	if masterServer.mapCounter == masterServer.totalMappers {
		masterServer.mapDone <- true
		log.Println("Master - All Map tasks completed, proceeding to Reduce phase")
	}
	masterServer.mu.Unlock()

	return &mapReducepb.MapResultResponse{}, nil
}

func (masterServer *MasterServer) ReduceResultRPC(ctx context.Context, req *mapReducepb.ReduceResult) (*mapReducepb.ReduceResultResponse, error) {
	masterServer.mu.Lock()
	masterServer.reduceCounter++
	log.Printf("Master - Received Reduce task completion (%d/%d)", masterServer.reduceCounter, masterServer.totalReducers)

	if masterServer.reduceCounter == masterServer.totalReducers {
		masterServer.reduceDone <- true
		log.Println("Master - All Reduce tasks completed, MapReduce Job Done!")
	}
	masterServer.mu.Unlock()

	return &mapReducepb.ReduceResultResponse{}, nil
}