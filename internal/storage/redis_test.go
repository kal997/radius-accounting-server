package storage

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kal997/radius-accounting-server/internal/models"
)

// helper to create test storage with miniredis
func newTestStorage(t *testing.T, ttl time.Duration) (*RedisStorage, *miniredis.Miniredis, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	storage := &RedisStorage{
		client: newTestClient(mr.Addr()),
		ttl:    ttl,
	}

	cleanup := func() {
		storage.Close()
		mr.Close()
	}
	return storage, mr, cleanup
}

func newTestClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: addr, DB: 0})
}

func TestRedisStorage_StoreAndRetrieve(t *testing.T) {
	storage, mr, cleanup := newTestStorage(t, 5*time.Minute)
	defer cleanup()

	record := &models.AccountingRecord{
		Username:       "testuser",
		AcctSessionID:  "session123",
		NASIPAddress:   "127.0.0.1",
		AcctStatusType: models.Start,
		Timestamp:      time.Now().Format(time.RFC3339Nano),
		ClientIP:       "192.168.1.10",
		PacketType:     "Accounting-Request",
	}

	ctx := context.Background()
	err := storage.Store(ctx, record)
	require.NoError(t, err)

	// Check directly in miniredis
	val, err := mr.Get(record.GenerateRedisKey())
	require.NoError(t, err)

	var got models.AccountingRecord
	require.NoError(t, json.Unmarshal([]byte(val), &got))
	assert.Equal(t, record.Username, got.Username)
	assert.Equal(t, record.AcctSessionID, got.AcctSessionID)
}

func TestRedisStorage_HealthCheck(t *testing.T) {
	storage, mr, cleanup := newTestStorage(t, 1*time.Minute)
	defer cleanup()

	ctx := context.Background()
	assert.NoError(t, storage.HealthCheck(ctx))

	// stop server -> should fail
	mr.Close()
	assert.Error(t, storage.HealthCheck(ctx))
}

func TestRedisStorage_Close(t *testing.T) {
	storage, _, cleanup := newTestStorage(t, 1*time.Minute)
	defer cleanup()

	assert.NoError(t, storage.Close())
	// calling twice should raise an error
	assert.Error(t, storage.Close())
}
