package logger

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// FileLogger implements Logger interface for file-based logging
type FileLogger struct {
	file   *os.File
	mutex  sync.Mutex
	closed bool
}

// NewFileLogger creates a new file logger
func NewFileLogger(logfile string) (*FileLogger, error) {
	file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileLogger{
		file: file,
	}, nil
}

// Log writes a timestamped message to the file
func (fl *FileLogger) Log(ctx context.Context, message string) error {
	fl.mutex.Lock()
	defer fl.mutex.Unlock()

	if fl.closed {
		return fmt.Errorf("logger is closed")
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000000")
	logLine := fmt.Sprintf("%s - %s\n", timestamp, message)

	if _, err := fl.file.WriteString(logLine); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	// Ensure data is written to disk
	return fl.file.Sync()
}

// Close closes the log file
func (fl *FileLogger) Close() error {
	fl.mutex.Lock()
	defer fl.mutex.Unlock()

	if fl.closed {
		return nil
	}

	fl.closed = true
	return fl.file.Close()
}
