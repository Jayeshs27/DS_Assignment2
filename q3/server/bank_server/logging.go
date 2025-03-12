package main

import (
	"context"
	// "fmt"
	// "strings"
	// "time"

	// "github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	// "google.golang.org/grpc/metadata"
	common "q3/common"
)

func bankLoggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, 
	handler grpc.UnaryHandler) (any, error) {
	
	bSLogger.PrintLog("Method: %s, Request: %v", info.FullMethod, req)
    resp, err := handler(ctx, req)
    if err != common.ErrSuccess {
        bSLogger.PrintLog("RPC failed with error: %v", err)
    }
	bSLogger.PrintLog("Method: %s, Response: %v", info.FullMethod, resp)
	
    return resp, err
}
