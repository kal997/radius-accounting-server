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
			name: "invalid TTL",
			envVars: map[string]string{
				"RADIUS_SHARED_SECRET": "secret123",
				"REDIS_HOST":           "localhost",
				"RECORD_TTL_HOURS":     "invalid",
			},
			wantErr: "invalid RECORD_TTL_HOURS",
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

// Helper function to clear environment variables
func clearEnv() {
	envVars := []string{
		"RADIUS_SHARED_SECRET", "REDIS_HOST", "RECORD_TTL_HOURS",
		"LOG_LEVEL", "LOG_FILE",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}
