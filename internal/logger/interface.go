package logger

import "context"

// Logger defines the interface for logging messages
type Logger interface {
	// Log writes a message to the logger
	Log(ctx context.Context, message string) error

	// Close closes the logger and any resources
	Close() error
}
