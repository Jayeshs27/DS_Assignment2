package main

import (
	// "fmt"
	// "log"
	// "os"
	// "sync"
	"errors"
	"fmt"
	"q3/common"
	"time"

	// "bufio"
	"context"

	"google.golang.org/grpc"
)

var (
	maxRetries = 3
	timeoutInterval = 5
)
// Global variables for logging
// var (
// 	logFile  *os.File
// 	logMutex sync.Mutex // Ensures thread-safe writes
// )

// Initialize logging (Creates a new file per session)
// type Logger struct {
// 	logFile  *os.File
// 	logMutex sync.Mutex
// }

// // NewLogger initializes a new logger instance
// func NewLogger(folderPath string) *Logger {
// 	timestamp := time.Now().Format("2006-01-02_15-04-05")
// 	filename := fmt.Sprintf("%s/session_%s.log", folderPath, timestamp)

// 	// Ensure the logs directory exists
// 	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
// 		log.Fatalf("Failed to create logs directory: %v", err)
// 	}

// 	// Create the log file
// 	logFile, err := os.Create(filename)
// 	if err != nil {
// 		log.Fatalf("Failed to open log file: %v", err)
// 	}

// 	return &Logger{logFile: logFile}
// }

// // PrintLog safely writes log messages with timestamp
// func (l *Logger) PrintLog(format string, a ...any) {
// 	l.logMutex.Lock() // Ensure thread safety
// 	defer l.logMutex.Unlock()

// 	timestamp := time.Now().Format("2006-01-02 15:04:05") // YYYY-MM-DD HH:MM:SS
// 	writer := bufio.NewWriter(l.logFile)
// 	fmt.Fprintf(writer, "[%s] %s\n", timestamp, fmt.Sprintf(format, a...))
// 	writer.Flush()
// }

// // Close properly closes the log file
// func (l *Logger) Close() {
// 	if l.logFile != nil {
// 		l.logFile.Close()
// 	}
// }

func loggingInterceptor(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
    timeout := time.Duration(timeoutInterval) * time.Second 
    var err error
    for i := range maxRetries {
        retryCtx, cancel := context.WithTimeout(ctx, timeout)
        defer cancel() 

        startTime := time.Now()
        clientLogger.PrintLog("Attempt %d - Request: %v", i+1, req)

        err = invoker(retryCtx, method, req, reply, cc, opts...)
        elapsed := time.Since(startTime)

        if err == common.ErrSuccess {
            clientLogger.PrintLog("Response: %v", reply)
            clientLogger.PrintLog("RPC Call Succeeded in %v", elapsed)
            return common.ErrSuccess
        }
		
		if i == maxRetries -1 {
			clientLogger.PrintLog("RPC Call Failed after %d attempts", maxRetries)
			return err
		}

        if errors.Is(retryCtx.Err(), context.DeadlineExceeded) {
			clientLogger.PrintLog("Request Timeout (Attempt %d): Server taking too long | Retrying ..", i+1)
		} else {
			fmt.Println("error - ", err)
            clientLogger.PrintLog("RPC failed (Attempt %d) with error: %v", i+1, err)
			return err
        }

        time.Sleep(time.Duration(i + 1) * time.Second)
    }
    return err
}


// func unaryInterceptor(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
// 	var credsConfigured bool
// 	for _, o := range opts {
// 		_, ok := o.(grpc.PerRPCCredsCallOption)
// 			credsConfigured = true
// 			break
// 		}
// 	}
// 	if !credsConfigured {
// 		opts = append(opts, grpc.PerRPCCredentials(oauth.TokenSource{
// 			TokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: fallbackToken}),
// 		}))
// 	}
// 	start := time.Now()
// 	err := invoker(ctx, method, req, reply, cc, opts...)
// 	end := time.Now()
// 	logger("RPC: %s, start time: %s, end time: %s, err: %v", method, start.Format("Basic"), end.Format(time.RFC3339), err)
// 	return err
// }

// func main() {
// 	initLogger()
// 	defer closeLogger() // Ensure the log file is closed on exit

// 	// Example log entries
// 	logMessage("INFO", "Server started successfully")
// 	logMessage("ERROR", "Failed to connect to database")
// 	logMessage("DEBUG", "Request received from 192.168.1.10")

// 	fmt.Println("Logging system initialized. Check the logs directory.")
// }
