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
    if !common.IsEqual(err, common.ErrSuccess) {
        bSLogger.PrintLog("RPC failed with error: %v", common.ErrorMessage(err))
    }
	bSLogger.PrintLog("Method: %s, Response: %v, Status: Sucess", info.FullMethod, resp)
	
    return resp, err
}

func pgRegisterInterceptor(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	bSLogger.PrintLog("Method: %s, Request: %v", method, req)
	err := invoker(ctx, method, req, reply, cc, opts...)
	if !common.IsEqual(err, common.ErrSuccess) {
		bSLogger.PrintLog("Failedd to send the request, %v", err)
	}
	bSLogger.PrintLog("Method: %s, Response: %v, status: Success", method, reply)
	return err
}

