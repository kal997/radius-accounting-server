package storage

import (
	"context"

	"github.com/kal997/radius-accounting-system/internal/models"
)


// Storage defines the database-agnostic storage interface
type Storage interface {
	// Store saves an accounting record
	Store(ctx context.Context, record *models.AccountingRecord) error

	// HealthCheck verifies storage connectivity
	HealthCheck(ctx context.Context) error

	// Close closes the storage connection
	Close() error
}
