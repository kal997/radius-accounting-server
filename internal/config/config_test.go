package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromEnv_ValidConfig(t *testing.T) {
	// Clean environment
	clearEnv()
	defer clearEnv()

	// Set valid environment variables
	os.Setenv("RADIUS_SHARED_SECRET", "secretkey123")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("RECORD_TTL_HOURS", "24")
	os.Setenv("LOG_LEVEL", "info")
	os.Setenv("LOG_FILE", "/var/log/test.log")

	cfg, err := LoadFromEnv()

	require.NoError(t, err)
	assert.Equal(t, ":1813", cfg.GetRADIUSAddr())
	assert.Equal(t, "secretkey123", cfg.GetSharedSecret())
	assert.Equal(t, "localhost:6379", cfg.GetRedisAddr())
	assert.Equal(t, 24*time.Hour, cfg.GetRecordTTL())
	assert.Equal(t, LogLevelInfo, cfg.GetLogLevel())
	assert.Equal(t, "/var/log/test.log", cfg.GetLogFile())
}

func TestLoadFromEnv_ValidConfig_test_setup(t *testing.T) {
	// Clean environment
	clearEnv()
	defer clearEnv()

	// Set valid environment variables
	os.Setenv("RADIUS_SHARED_SECRET", "secretkey123")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("RECORD_TTL_HOURS", "24")
	os.Setenv("LOG_LEVEL", "info")
	os.Setenv("REDIS_PORT", "1111")
	os.Setenv("LOG_FILE", "/var/log/test.log")

	cfg, err := LoadFromEnv()

	require.NoError(t, err)
	assert.Equal(t, ":1813", cfg.GetRADIUSAddr())
	assert.Equal(t, "secretkey123", cfg.GetSharedSecret())
	assert.Equal(t, "localhost:1111", cfg.GetRedisAddr())
	assert.Equal(t, 24*time.Hour, cfg.GetRecordTTL())
	assert.Equal(t, LogLevelInfo, cfg.GetLogLevel())
	assert.Equal(t, "/var/log/test.log", cfg.GetLogFile())
}
func TestLoadFromEnv_MissingRequired(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr string
	}{
		{
			name:    "missing shared secret",
			envVars: map[string]string{},
			wantErr: "RADIUS_SHARED_SECRET environment variable is required",
		},
		{
			name: "missing redis host",
			envVars: map[string]string{
				"RADIUS_SHARED_SECRET": "secret123",
			},
			wantErr: "REDIS_HOST environment variable is required",
		},
		{
			name: "missing TTL",
			envVars: map[string]string{
				"RADIUS_SHARED_SECRET": "secret123",
				"REDIS_HOST":           "localhost",
			},
			wantErr: "RECORD_TTL_HOURS environment variable is required",
		},
		{
			name: "invalid TTL",
			envVars: map[string]string{
				"RADIUS_SHARED_SECRET": "secret123",
				"REDIS_HOST":           "localhost",
				"RECORD_TTL_HOURS":     "invalid",
			},
			wantErr: "invalid RECORD_TTL_HOURS",
		},
		{
			name: "missing log level",
			envVars: map[string]string{
				"RADIUS_SHARED_SECRET": "secret123",
				"REDIS_HOST":           "localhost",
				"RECORD_TTL_HOURS":     "24",
			},
			wantErr: "LOG_LEVEL environment variable is required",
		},
		{
			name: "invalid log level",
			envVars: map[string]string{
				"RADIUS_SHARED_SECRET": "secret123",
				"REDIS_HOST":           "localhost",
				"RECORD_TTL_HOURS":     "24",
				"LOG_LEVEL":            "invalid",
			},
			wantErr: "invalid LOG_LEVEL: invalid (valid: debug, info, warn, error)",
		},
		{
			name: "missing log file",
			envVars: map[string]string{
				"RADIUS_SHARED_SECRET": "secret123",
				"REDIS_HOST":           "localhost",
				"RECORD_TTL_HOURS":     "24",
				"LOG_LEVEL":            "info",
			},
			wantErr: "LOG_FILE environment variable is required",
		},
		{
			name: "invalid REDIS_PORT",
			envVars: map[string]string{
				"RADIUS_SHARED_SECRET": "secret123",
				"REDIS_HOST":           "localhost",
				"RECORD_TTL_HOURS":     "24",
				"LOG_LEVEL":            "info",
				"LOG_FILE":             "/var/log/test.log",
				"REDIS_PORT":           "invalid",
			},
			wantErr: "invalid REDIS_ADDR: strconv.Atoi: parsing \"invalid\": invalid syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv()
			defer clearEnv()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := LoadFromEnv()

			assert.Error(t, err)
			assert.Nil(t, cfg)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr string
	}{
		{
			name: "valid config",
			config: &Config{
				radiusPort:   1813,
				sharedSecret: "verysecret123",
				redisHost:    "localhost",
				redisPort:    6379,
				recordTTL:    24 * time.Hour,
				logLevel:     LogLevelInfo,
				logFile:      "/var/log/test.log",
			},
			wantErr: "",
		},
		{
			name: "empty shared secret",
			config: &Config{
				radiusPort:   1813,
				sharedSecret: "",
				redisHost:    "localhost",
				redisPort:    6379,
				recordTTL:    24 * time.Hour,
				logLevel:     LogLevelInfo,
				logFile:      "/var/log/test.log",
			},
			wantErr: "shared secret cannot be empty",
		},
		{
			name: "short shared secret",
			config: &Config{
				radiusPort:   1813,
				sharedSecret: "short",
				redisHost:    "localhost",
				redisPort:    6379,
				recordTTL:    24 * time.Hour,
				logLevel:     LogLevelInfo,
				logFile:      "/var/log/test.log",
			},
			wantErr: "shared secret must be at least 8 characters long",
		},
		{
			name: "empty redis host",
			config: &Config{
				radiusPort:   1813,
				sharedSecret: "verysecret123",
				redisHost:    "",
				redisPort:    6379,
				recordTTL:    24 * time.Hour,
				logLevel:     LogLevelInfo,
				logFile:      "/var/log/test.log",
			},
			wantErr: "redis host cannot be empty",
		},
		{
			name: "zero TTL",
			config: &Config{
				radiusPort:   1813,
				sharedSecret: "verysecret123",
				redisHost:    "localhost",
				redisPort:    6379,
				recordTTL:    0,
				logLevel:     LogLevelInfo,
				logFile:      "/var/log/test.log",
			},
			wantErr: "record TTL must be greater than 0",
		},
		{
			name: "negative TTL",
			config: &Config{
				radiusPort:   1813,
				sharedSecret: "verysecret123",
				redisHost:    "localhost",
				redisPort:    6379,
				recordTTL:    -1 * time.Hour,
				logLevel:     LogLevelInfo,
				logFile:      "/var/log/test.log",
			},
			wantErr: "record TTL must be greater than 0",
		},
		{
			name: "invalid log level",
			config: &Config{
				radiusPort:   1813,
				sharedSecret: "verysecret123",
				redisHost:    "localhost",
				redisPort:    6379,
				recordTTL:    24 * time.Hour,
				logLevel:     LogLevel("invalid"),
				logFile:      "/var/log/test.log",
			},
			wantErr: "invalid log level: invalid (valid: debug, info, warn, error)",
		},
		{
			name: "empty log file",
			config: &Config{
				radiusPort:   1813,
				sharedSecret: "verysecret123",
				redisHost:    "localhost",
				redisPort:    6379,
				recordTTL:    24 * time.Hour,
				logLevel:     LogLevelInfo,
				logFile:      "",
			},
			wantErr: "log file path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestConfig_GetterMethods(t *testing.T) {
	clearEnv()
	defer clearEnv()

	// Set up valid config
	os.Setenv("RADIUS_SHARED_SECRET", "secretkey123")
	os.Setenv("REDIS_HOST", "redis-server")
	os.Setenv("RECORD_TTL_HOURS", "48")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FILE", "/var/log/radius.log")

	cfg, err := LoadFromEnv()
	require.NoError(t, err)

	// Test all getters
	assert.Equal(t, "redis-server:6379", cfg.GetRedisAddr())
	assert.Equal(t, ":1813", cfg.GetRADIUSAddr())
	assert.Equal(t, "secretkey123", cfg.GetSharedSecret())
	assert.Equal(t, 48*time.Hour, cfg.GetRecordTTL())
	assert.Equal(t, LogLevelDebug, cfg.GetLogLevel())
	assert.Equal(t, "/var/log/radius.log", cfg.GetLogFile())
	assert.True(t, cfg.IsDebugEnabled())
}

func TestIsDebugEnabled(t *testing.T) {
	tests := []struct {
		name     string
		logLevel LogLevel
		want     bool
	}{
		{
			name:     "debug level returns true",
			logLevel: LogLevelDebug,
			want:     true,
		},
		{
			name:     "info level returns false",
			logLevel: LogLevelInfo,
			want:     false,
		},
		{
			name:     "warn level returns false",
			logLevel: LogLevelWarn,
			want:     false,
		},
		{
			name:     "error level returns false",
			logLevel: LogLevelError,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				logLevel: tt.logLevel,
			}
			assert.Equal(t, tt.want, cfg.IsDebugEnabled())
		})
	}
}

func TestIsValidLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		want  bool
	}{
		{
			name:  "debug is valid",
			level: LogLevelDebug,
			want:  true,
		},
		{
			name:  "info is valid",
			level: LogLevelInfo,
			want:  true,
		},
		{
			name:  "warn is valid",
			level: LogLevelWarn,
			want:  true,
		},
		{
			name:  "error is valid",
			level: LogLevelError,
			want:  true,
		},
		{
			name:  "invalid level",
			level: LogLevel("invalid"),
			want:  false,
		},
		{
			name:  "empty level",
			level: LogLevel(""),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidLogLevel(tt.level))
		})
	}
}

// Helper function to clear environment variables
func clearEnv() {
	envVars := []string{
		"RADIUS_SHARED_SECRET", "REDIS_HOST", "RECORD_TTL_HOURS",
		"LOG_LEVEL", "LOG_FILE", "REDIS_PORT",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}
