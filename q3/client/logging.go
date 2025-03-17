package main

import (
	"errors"
	// "fmt"
	"q3/common"
	"time"

	// "bufio"
	"context"

	"google.golang.org/grpc"
)

var (
	maxRetries = 5
	timeoutInterval = 5
)


func loggingInterceptor(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
    timeout := time.Duration(timeoutInterval) * time.Second 
    var err error
    for i := range maxRetries {
        retryCtx, cancel := context.WithTimeout(ctx, timeout)
        defer cancel() 

        clientLogger.PrintLog("Attempt %d - Request: %v", i+1, req)
        err = invoker(retryCtx, method, req, reply, cc, opts...)

        if errors.Is(err, common.ErrSuccess) {
			clientLogger.PrintLog("Response : Success")
            return common.ErrSuccess
        }
		
		if i == maxRetries - 1 {
			clientLogger.PrintLog("RPC Call Failed after %d attempts with error: %v", maxRetries, err)
			if common.IsEqual(retryCtx.Err(), context.DeadlineExceeded) || common.IsEqual(err, common.ErrTransactionInProgress){
				return common.ErrTimeOut
			}
			
		}
		
        if common.IsEqual(retryCtx.Err(), context.DeadlineExceeded) || common.IsEqual(err, common.ErrTransactionInProgress) {
			clientLogger.PrintLog("Request Timeout (Attempt %d): Server taking too long | Retrying ..", i+1)
		} else {
            clientLogger.PrintLog("RPC failed (Attempt %d) with error: %v", i+1, err)
			if common.IsEqual(err, common.ErrTransactionInProgress){
				return common.ErrTimeOut
			}
			return err
        }
        time.Sleep(time.Duration(i + 1) * time.Second)
    }
    return err
}

