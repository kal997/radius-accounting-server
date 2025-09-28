package notifier

import (
	"context"
	"time"
)

type Notifier interface {
	Subscribe(ctx context.Context, patterns []string) (<-chan StorageEvent, error)
	Unsubscribe(patterns []string) error
	HealthCheck(ctx context.Context) error
	Close() error
}

type StorageEvent struct {
	Key       string
	Operation string
	Timestamp time.Time
}
