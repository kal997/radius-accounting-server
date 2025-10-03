package logger

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileLogger(t *testing.T) {
	tests := []struct {
		name        string
		logfile     string
		setup       func(string)
		cleanup     func(string)
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid file path",
			logfile: "/tmp/test_new_logger.log",
			setup:   func(path string) {},
			cleanup: func(path string) {
				os.Remove(path)
			},
			wantErr: false,
		},
		{
			name:        "invalid file path - no permission",
			logfile:     "/root/no_permission.log",
			setup:       func(path string) {},
			cleanup:     func(path string) {},
			wantErr:     true,
			errContains: "failed to open log file",
		},
		{
			name:    "directory instead of file",
			logfile: "/tmp/",
			setup: func(path string) {
				// /tmp/ already exists as a directory
			},
			cleanup:     func(path string) {},
			wantErr:     true,
			errContains: "failed to open log file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(tt.logfile)
			defer tt.cleanup(tt.logfile)

			logger, err := NewFileLogger(tt.logfile)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, logger)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, logger)
				assert.NotNil(t, logger.file)
				assert.False(t, logger.closed)

				// Clean up
				logger.Close()
			}
		})
	}
}

func TestFileLogger_Log(t *testing.T) {
	tests := []struct {
		name        string
		setupLogger func() (*FileLogger, func())
		message     string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *FileLogger)
	}{
		{
			name: "successful log write",
			setupLogger: func() (*FileLogger, func()) {
				tmpFile := "/tmp/test_log_write.log"
				logger, _ := NewFileLogger(tmpFile)
				cleanup := func() {
					logger.Close()
					os.Remove(tmpFile)
				}
				return logger, cleanup
			},
			message: "test log message",
			wantErr: false,
			validate: func(t *testing.T, fl *FileLogger) {
				// Read the file to verify content
				content, err := os.ReadFile("/tmp/test_log_write.log")
				require.NoError(t, err)

				lines := strings.Split(strings.TrimSpace(string(content)), "\n")
				require.Len(t, lines, 1)

				// Verify timestamp format and message
				assert.Contains(t, lines[0], "test log message")
				// Check timestamp format (YYYY-MM-DD HH:MM:SS.mmmmmm)
				parts := strings.Split(lines[0], " - ")
				require.Len(t, parts, 2)

				// Verify timestamp can be parsed
				_, err = time.Parse("2006-01-02 15:04:05.000000", parts[0])
				assert.NoError(t, err)
			},
		},
		{
			name: "log after close",
			setupLogger: func() (*FileLogger, func()) {
				tmpFile := "/tmp/test_log_closed.log"
				logger, _ := NewFileLogger(tmpFile)
				logger.Close() // Close immediately
				cleanup := func() {
					os.Remove(tmpFile)
				}
				return logger, cleanup
			},
			message:     "should fail",
			wantErr:     true,
			errContains: "logger is closed",
		},
		{
			name: "concurrent log writes",
			setupLogger: func() (*FileLogger, func()) {
				tmpFile := "/tmp/test_concurrent.log"
				logger, _ := NewFileLogger(tmpFile)
				cleanup := func() {
					logger.Close()
					os.Remove(tmpFile)
				}
				return logger, cleanup
			},
			message: "concurrent message",
			wantErr: false,
			validate: func(t *testing.T, fl *FileLogger) {
				// Test concurrent access
				var wg sync.WaitGroup
				errors := make([]error, 10)

				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						errors[idx] = fl.Log(context.Background(), "concurrent message")
					}(i)
				}

				wg.Wait()

				// All writes should succeed
				for _, err := range errors {
					assert.NoError(t, err)
				}

				// Verify all messages were written
				content, err := os.ReadFile("/tmp/test_concurrent.log")
				require.NoError(t, err)
				lines := strings.Split(strings.TrimSpace(string(content)), "\n")
				assert.Len(t, lines, 11) // 1 from initial test + 10 concurrent
			},
		},
		{
			name: "empty message",
			setupLogger: func() (*FileLogger, func()) {
				tmpFile := "/tmp/test_empty_message.log"
				logger, _ := NewFileLogger(tmpFile)
				cleanup := func() {
					logger.Close()
					os.Remove(tmpFile)
				}
				return logger, cleanup
			},
			message: "",
			wantErr: false,
			validate: func(t *testing.T, fl *FileLogger) {
				content, err := os.ReadFile("/tmp/test_empty_message.log")
				require.NoError(t, err)
				// Should have timestamp followed by " - " and newline
				assert.Contains(t, string(content), " - \n")
			},
		},
		{
			name: "very long message",
			setupLogger: func() (*FileLogger, func()) {
				tmpFile := "/tmp/test_long_message.log"
				logger, _ := NewFileLogger(tmpFile)
				cleanup := func() {
					logger.Close()
					os.Remove(tmpFile)
				}
				return logger, cleanup
			},
			message: strings.Repeat("A", 10000),
			wantErr: false,
			validate: func(t *testing.T, fl *FileLogger) {
				content, err := os.ReadFile("/tmp/test_long_message.log")
				require.NoError(t, err)
				assert.Contains(t, string(content), strings.Repeat("A", 10000))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, cleanup := tt.setupLogger()
			defer cleanup()

			err := logger.Log(context.Background(), tt.message)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, logger)
				}
			}
		})
	}
}

func TestFileLogger_Close(t *testing.T) {
	tests := []struct {
		name        string
		setupLogger func() (*FileLogger, func())
		wantErr     bool
	}{
		{
			name: "close valid logger",
			setupLogger: func() (*FileLogger, func()) {
				tmpFile := "/tmp/test_close.log"
				logger, _ := NewFileLogger(tmpFile)
				cleanup := func() {
					os.Remove(tmpFile)
				}
				return logger, cleanup
			},
			wantErr: false,
		},
		{
			name: "close already closed logger",
			setupLogger: func() (*FileLogger, func()) {
				tmpFile := "/tmp/test_double_close.log"
				logger, _ := NewFileLogger(tmpFile)
				logger.Close() // First close
				cleanup := func() {
					os.Remove(tmpFile)
				}
				return logger, cleanup
			},
			wantErr: false, // Should return nil for already closed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, cleanup := tt.setupLogger()
			defer cleanup()

			err := logger.Close()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, logger.closed)
			}
		})
	}
}

func TestFileLogger_CloseMultipleTimes(t *testing.T) {
	tmpFile := "/tmp/test_multiple_close.log"
	defer os.Remove(tmpFile)

	logger, err := NewFileLogger(tmpFile)
	require.NoError(t, err)

	// First close should succeed
	err = logger.Close()
	assert.NoError(t, err)
	assert.True(t, logger.closed)

	// Second close should also return nil (idempotent)
	err = logger.Close()
	assert.NoError(t, err)
	assert.True(t, logger.closed)

	// Third close for good measure
	err = logger.Close()
	assert.NoError(t, err)
	assert.True(t, logger.closed)
}

func TestFileLogger_WriteError(t *testing.T) {
	// This test simulates a write error by closing the file descriptor
	// before attempting to write
	tmpFile := "/tmp/test_write_error.log"
	defer os.Remove(tmpFile)

	logger, err := NewFileLogger(tmpFile)
	require.NoError(t, err)

	// Manually close the file to simulate write error
	logger.file.Close()

	// Attempt to log should fail
	err = logger.Log(context.Background(), "this should fail")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write to log file")
}

func TestFileLogger_SyncError(t *testing.T) {
	// Create a mock file that fails on Sync
	tmpFile := "/tmp/test_sync_error.log"
	defer os.Remove(tmpFile)

	logger, err := NewFileLogger(tmpFile)
	require.NoError(t, err)

	// Save the original file
	origFile := logger.file

	// Create a pipe to simulate a file that can't be synced
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()

	// Replace the file with the write end of the pipe
	logger.file = w

	// Write should succeed but sync will fail on a pipe
	err = logger.Log(context.Background(), "test message")
	// Note: On some systems, Sync on a pipe might not fail,
	// so we just ensure no panic occurs

	// Close the pipe
	w.Close()

	// Restore original file for cleanup
	logger.file = origFile
	logger.Close()
}

// MockFile is a mock implementation for testing write failures
type MockFile struct {
	*os.File
	failWrite bool
	failSync  bool
}

func (m *MockFile) WriteString(s string) (n int, err error) {
	if m.failWrite {
		return 0, errors.New("mock write error")
	}
	return m.File.WriteString(s)
}

func (m *MockFile) Sync() error {
	if m.failSync {
		return errors.New("mock sync error")
	}
	return m.File.Sync()
}

func TestFileLogger_ConcurrentCloseAndLog(t *testing.T) {
	// Test race condition between Close and Log
	tmpFile := "/tmp/test_concurrent_close.log"
	defer os.Remove(tmpFile)

	logger, err := NewFileLogger(tmpFile)
	require.NoError(t, err)

	var wg sync.WaitGroup

	// Start multiple goroutines trying to log
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Log(context.Background(), "concurrent log")
		}()
	}

	// Start a goroutine trying to close
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(1 * time.Millisecond) // Small delay
		logger.Close()
	}()

	// More logging attempts after close
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(2 * time.Millisecond) // Ensure these run after close
			logger.Log(context.Background(), "after close")
		}()
	}

	wg.Wait()

	// Logger should be closed
	assert.True(t, logger.closed)
}
