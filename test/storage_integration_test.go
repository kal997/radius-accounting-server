//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/kal997/radius-accounting-server/internal/config"
	"github.com/kal997/radius-accounting-server/internal/models"
	"github.com/kal997/radius-accounting-server/internal/storage"
	"github.com/stretchr/testify/assert"
)

// internal/storage/integration_test.go
func TestRedisStorage_Integration(t *testing.T) {
	// Load real config from environment
	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Skipf("Config not available: %v", err)
	}

	// Create storage
	storage, err := storage.NewRedisStorage(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer storage.Close()

	// Create start test record
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

	// Test store
	ctx := context.Background()
	err = storage.Store(ctx, record)
	assert.NoError(t, err)
}
