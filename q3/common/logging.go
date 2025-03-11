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

// Initialize logging (Creates a new file per session)
type Logger struct {
	logFile  *os.File
	logMutex sync.Mutex
}

// NewLogger initializes a new logger instance
func NewLogger(folderPath string) *Logger {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s/session_%s.log", folderPath, timestamp)

	// Ensure the logs directory exists
	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	// Create the log file
	logFile, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	return &Logger{logFile: logFile}
}

// PrintLog safely writes log messages with timestamp
func (l *Logger) PrintLog(format string, a ...any) {
	l.logMutex.Lock() // Ensure thread safety
	defer l.logMutex.Unlock()
	timestamp := time.Now().Format("2006-01-02 15:04:05") // YYYY-MM-DD HH:MM:SS
	writer := bufio.NewWriter(l.logFile)
	fmt.Fprintf(writer, "[%s] %s\n", timestamp, fmt.Sprintf(format, a...))
	writer.Flush()
	// l.logMutex.Unlock()
}

// Close properly closes the log file
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}
