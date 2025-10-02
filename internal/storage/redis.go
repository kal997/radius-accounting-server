package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kal997/radius-accounting-server/internal/config"
	"github.com/kal997/radius-accounting-server/internal/models"

	"github.com/redis/go-redis/v9"
)

// RedisStorage implements the Storage interface using Redis
type RedisStorage struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisStorage creates a new Redis storage instance
func NewRedisStorage(cfg *config.Config) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr: cfg.GetRedisAddr(),
		DB:   0,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStorage{
		client: client,
		ttl:    cfg.GetRecordTTL(),
	}, nil
}

// Store saves an accounting record
func (rs *RedisStorage) Store(ctx context.Context, record *models.AccountingRecord) error {
	key := record.GenerateRedisKey()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	if err := rs.client.Set(ctx, key, data, rs.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store record in Redis: %w", err)
	}

	return nil
}

// HealthCheck verifies Redis connectivity
func (rs *RedisStorage) HealthCheck(ctx context.Context) error {
	return rs.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (rs *RedisStorage) Close() error {
	return rs.client.Close()
}
