package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// Config holds all application configuration
// Fields are private to ensure immutability after creation
type Config struct {
	// RADIUS server configuration
	radiusPort   int
	sharedSecret string

	// Redis configuration
	redisHost string
	redisPort int
	recordTTL time.Duration

	// Logging configuration
	logLevel LogLevel
	logFile  string
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	config := &Config{
		radiusPort: 1813, // Standard RADIUS accounting port
		redisPort:  6379, // Standard Redis port
	}

	// RADIUS configuration
	secret := os.Getenv("RADIUS_SHARED_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("RADIUS_SHARED_SECRET environment variable is required")
	}
	config.sharedSecret = secret

	// Redis configuration
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		return nil, fmt.Errorf("REDIS_HOST environment variable is required")
	}
	config.redisHost = host

	// TTL configuration
	ttlStr := os.Getenv("RECORD_TTL_HOURS")
	if ttlStr == "" {
		return nil, fmt.Errorf("RECORD_TTL_HOURS environment variable is required")
	}
	hours, err := strconv.Atoi(ttlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid RECORD_TTL_HOURS: %w", err)
	}
	config.recordTTL = time.Duration(hours) * time.Hour

	// Logging configuration
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		return nil, fmt.Errorf("LOG_LEVEL environment variable is required")
	}
	logLevel := LogLevel(levelStr)
	if !isValidLogLevel(logLevel) {
		return nil, fmt.Errorf("invalid LOG_LEVEL: %s (valid: debug, info, warn, error)", levelStr)
	}
	config.logLevel = logLevel

	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		return nil, fmt.Errorf("LOG_FILE environment variable is required")
	}
	config.logFile = logFile

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {

	if c.sharedSecret == "" {
		return fmt.Errorf("shared secret cannot be empty")
	}

	if len(c.sharedSecret) < 8 {
		return fmt.Errorf("shared secret must be at least 8 characters long")
	}

	if c.redisHost == "" {
		return fmt.Errorf("redis host cannot be empty")
	}


	if c.recordTTL <= 0 {
		return fmt.Errorf("record TTL must be greater than 0")
	}

	if !isValidLogLevel(c.logLevel) {
		return fmt.Errorf("invalid log level: %s (valid: debug, info, warn, error)", c.logLevel)
	}

	if c.logFile == "" {
		return fmt.Errorf("log file path cannot be empty")
	}

	return nil
}

// GetRedisAddr returns the Redis address in host:port format
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.redisHost, c.redisPort)
}

// GetRADIUSAddr returns the RADIUS server address in :port format
func (c *Config) GetRADIUSAddr() string {
	return fmt.Sprintf(":%d", c.radiusPort)
}

// GetSharedSecret returns the RADIUS shared secret
func (c *Config) GetSharedSecret() string {
	return c.sharedSecret
}

// GetRecordTTL returns the record TTL duration
func (c *Config) GetRecordTTL() time.Duration {
	return c.recordTTL
}

// GetLogLevel returns the configured log level
func (c *Config) GetLogLevel() LogLevel {
	return c.logLevel
}

// GetLogFile returns the log file path
func (c *Config) GetLogFile() string {
	return c.logFile
}

// IsDebugEnabled returns true if debug logging is enabled
func (c *Config) IsDebugEnabled() bool {
	return c.logLevel == LogLevelDebug
}

// Helper function to validate log levels
func isValidLogLevel(level LogLevel) bool {
	switch level {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
		return true
	default:
		return false
	}
}