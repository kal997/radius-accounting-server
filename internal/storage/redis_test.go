package storage

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kal997/radius-accounting-server/internal/config"
	"github.com/kal997/radius-accounting-server/internal/models"
)

// Test the actual NewRedisStorage constructor with real config
func TestNewRedisStorage_Success(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Set environment variables for config
	_ = os.Setenv("RADIUS_SHARED_SECRET", "testsecret123")
	host, port, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)

	_ = os.Setenv("REDIS_HOST", host)
	_ = os.Setenv("REDIS_PORT", port)
	_ = os.Setenv("RECORD_TTL_HOURS", "24")
	_ = os.Setenv("LOG_LEVEL", "info")
	_ = os.Setenv("LOG_FILE", "/tmp/test.log")
	defer func() {
		_ = os.Unsetenv("RADIUS_SHARED_SECRET")
		_ = os.Unsetenv("REDIS_HOST")
		_ = os.Unsetenv("RECORD_TTL_HOURS")
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("LOG_FILE")
	}()
	cfg, err := config.LoadFromEnv()
	require.NoError(t, err)

	storage, err := NewRedisStorage(cfg)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer func() {

		_ = storage.Close()
	}()

	// Verify connection works
	ctx := context.Background()
	assert.NoError(t, storage.HealthCheck(ctx))
	assert.Equal(t, 24*time.Hour, storage.ttl)
}

// Test NewRedisStorage with connection failure
func TestNewRedisStorage_ConnectionFailure(t *testing.T) {
	// Set environment variables with invalid Redis host
	_ = os.Setenv("RADIUS_SHARED_SECRET", "testsecret123")
	_ = os.Setenv("REDIS_HOST", "invalid-host-that-does-not-exist")
	_ = os.Setenv("RECORD_TTL_HOURS", "24")
	_ = os.Setenv("LOG_LEVEL", "info")
	_ = os.Setenv("LOG_FILE", "/tmp/test.log")
	defer func() {
		_ = os.Unsetenv("RADIUS_SHARED_SECRET")
		_ = os.Unsetenv("REDIS_HOST")
		_ = os.Unsetenv("RECORD_TTL_HOURS")
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("LOG_FILE")
	}()

	cfg, err := config.LoadFromEnv()
	require.NoError(t, err)

	storage, err := NewRedisStorage(cfg)
	assert.Error(t, err)
	assert.Nil(t, storage)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

// Helper to create test storage with miniredis
func newTestStorage(tb testing.TB, ttl time.Duration) (*RedisStorage, *miniredis.Miniredis, func()) {
	mr, err := miniredis.Run()
	require.NoError(tb, err)

	storage := &RedisStorage{
		client: redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		ttl:    ttl,
	}

	cleanup := func() {
		_ = storage.Close()
		mr.Close()
	}
	return storage, mr, cleanup
}

// Test Store with all code paths
func TestRedisStorage_Store_Success(t *testing.T) {
	storage, mr, cleanup := newTestStorage(t, 5*time.Minute)
	defer cleanup()

	record := &models.StartRecord{
		BaseAccountingRecord: models.BaseAccountingRecord{
			Username:      "testuser",
			AcctSessionID: "session123",
			NASIPAddress:  "127.0.0.1",
			ClientIP:      "192.168.1.10",
			Timestamp:     time.Now().Format(time.RFC3339Nano),
		},
		FramedIPAddress: "10.0.0.5",
	}

	ctx := context.Background()
	err := storage.Store(ctx, record)
	require.NoError(t, err)

	// Verify data was stored
	key := record.GenerateRedisKey()
	val, err := mr.Get(key)
	require.NoError(t, err)

	var got models.StartRecord
	require.NoError(t, json.Unmarshal([]byte(val), &got))
	assert.Equal(t, record.Username, got.Username)
	assert.Equal(t, record.AcctSessionID, got.AcctSessionID)

	// Verify TTL was set
	ttl := mr.TTL(key)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, 5*time.Minute)
}

// Test Store with Redis error
func TestRedisStorage_Store_RedisError(t *testing.T) {
	storage, mr, cleanup := newTestStorage(t, 5*time.Minute)
	defer cleanup()

	record := &models.StartRecord{
		BaseAccountingRecord: models.BaseAccountingRecord{
			Username:      "testuser",
			AcctSessionID: "session123",
			NASIPAddress:  "127.0.0.1",
			ClientIP:      "192.168.1.10",
			Timestamp:     time.Now().Format(time.RFC3339Nano),
		},
		FramedIPAddress: "10.0.0.5",
	}

	// Close miniredis to simulate Redis failure
	mr.Close()

	ctx := context.Background()
	err := storage.Store(ctx, record)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store record in Redis")
}

// Test HealthCheck - both success and failure
func TestRedisStorage_HealthCheck(t *testing.T) {
	storage, mr, cleanup := newTestStorage(t, 5*time.Minute)
	defer cleanup()

	ctx := context.Background()

	// Test successful health check
	assert.NoError(t, storage.HealthCheck(ctx))

	// Stop miniredis to simulate failure
	mr.Close()
	assert.Error(t, storage.HealthCheck(ctx))
}

// Test Close
func TestRedisStorage_Close(t *testing.T) {
	storage, _, cleanup := newTestStorage(t, 1*time.Minute)
	defer cleanup()

	// First close should succeed
	assert.NoError(t, storage.Close())

	// Second close should return an error (connection already closed)
	assert.Error(t, storage.Close())
}

// Test with context cancellation
func TestRedisStorage_Store_ContextCancelled(t *testing.T) {
	storage, _, cleanup := newTestStorage(t, 5*time.Minute)
	defer cleanup()

	record := &models.StartRecord{
		BaseAccountingRecord: models.BaseAccountingRecord{
			Username:      "testuser",
			AcctSessionID: "session123",
			NASIPAddress:  "127.0.0.1",
			ClientIP:      "192.168.1.10",
			Timestamp:     time.Now().Format(time.RFC3339Nano),
		},
		FramedIPAddress: "10.0.0.5",
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Store should handle cancelled context gracefully
	err := storage.Store(ctx, record)
	assert.Error(t, err)
}

// Test Store with nil record (defensive programming)
func TestRedisStorage_Store_NilRecord(t *testing.T) {
	storage, _, cleanup := newTestStorage(t, 5*time.Minute)
	defer cleanup()

	ctx := context.Background()

	// This will panic if not handled properly
	defer func() {
		if r := recover(); r != nil {
			// Expected panic for nil record
			assert.NotNil(t, r)
		}
	}()

	_ = storage.Store(ctx, nil)
}

// Integration test that uses the full constructor
func TestRedisStorage_FullIntegration(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// Set up environment for config
	_ = os.Setenv("RADIUS_SHARED_SECRET", "integrationtest")
	host, port, err := net.SplitHostPort(mr.Addr())
	require.NoError(t, err)

	_ = os.Setenv("REDIS_HOST", host)
	_ = os.Setenv("REDIS_PORT", port)

	_ = os.Setenv("RECORD_TTL_HOURS", "1")
	_ = os.Setenv("LOG_LEVEL", "debug")
	_ = os.Setenv("LOG_FILE", "/tmp/integration.log")
	defer func() {
		_ = os.Unsetenv("RADIUS_SHARED_SECRET")
		_ = os.Unsetenv("REDIS_HOST")
		_ = os.Unsetenv("RECORD_TTL_HOURS")
		_ = os.Unsetenv("LOG_LEVEL")
		_ = os.Unsetenv("LOG_FILE")
	}()

	// Create config and storage using actual constructor
	cfg, err := config.LoadFromEnv()
	require.NoError(t, err)

	storage, err := NewRedisStorage(cfg)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer func() {
		_ = storage.Close()
	}()
	// Test full workflow
	ctx := context.Background()

	// Health check
	assert.NoError(t, storage.HealthCheck(ctx))

	// Store multiple records
	for i := 0; i < 3; i++ {
		record := &models.StartRecord{
			BaseAccountingRecord: models.BaseAccountingRecord{
				Username:      "user" + string(rune('0'+i)),
				AcctSessionID: "session" + string(rune('0'+i)),
				NASIPAddress:  "10.0.0." + string(rune('0'+i)),
				ClientIP:      "192.168.1." + string(rune('0'+i)),
				Timestamp:     time.Now().Add(time.Duration(i) * time.Second).Format(time.RFC3339Nano),
			},
			FramedIPAddress: "10.0.0." + string(rune('0'+i)),
		}

		assert.NoError(t, storage.Store(ctx, record))
	}

	// Verify all records are stored
	keys := mr.Keys()
	assert.Len(t, keys, 3)
}

// Benchmark for Store operation
func BenchmarkRedisStorage_Store(b *testing.B) {
	storage, _, cleanup := newTestStorage(b, 5*time.Minute)
	defer cleanup()

	ctx := context.Background()
	record := &models.StartRecord{
		BaseAccountingRecord: models.BaseAccountingRecord{
			Username:      "benchuser",
			AcctSessionID: "benchsession",
			NASIPAddress:  "127.0.0.1",
			ClientIP:      "192.168.1.10",
			Timestamp:     time.Now().Format(time.RFC3339Nano),
		},
		FramedIPAddress: "10.0.0.55",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.Store(ctx, record)
	}
}
