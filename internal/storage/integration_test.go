package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kal997/radius-accounting-server/internal/config"
	"github.com/kal997/radius-accounting-server/internal/models"
)

// internal/storage/integration_test.go
func TestRedisStorage_Integration(t *testing.T) {
	// Load real config from environment
	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Skipf("Config not available: %v", err)
	}

	// Create storage
	storage, err := NewRedisStorage(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer storage.Close()

	// Create test record
	record := &models.AccountingRecord{
		Username:       "testuser",
		AcctSessionID:  "session123",
		NASIPAddress:   "192.168.1.1",
		AcctStatusType: models.Start,
		Timestamp:      time.Now().Format(time.RFC3339Nano),
		ClientIP:       "192.168.1.100",
		PacketType:     "Accounting-Request",
	}

	// Test store
	ctx := context.Background()
	err = storage.Store(ctx, record)
	assert.NoError(t, err)
}
