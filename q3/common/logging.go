package common

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	"bufio"
	// "context"
	// "google.golang.org/grpc"
)

type Logger struct {
	logFile  *os.File
	logMutex sync.Mutex
}

func NewLogger(folderPath string) *Logger {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s/session_%s.log", folderPath, timestamp)

	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	logFile, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	return &Logger{logFile: logFile}
}

func (l *Logger) PrintLog(format string, a ...any) {
	l.logMutex.Lock() 
	defer l.logMutex.Unlock()
	timestamp := time.Now().Format("2006-01-02 15:04:05") 
	writer := bufio.NewWriter(l.logFile)
	fmt.Fprintf(writer, "[%s] %s\n", timestamp, fmt.Sprintf(format, a...))
	writer.Flush()
}

func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}
