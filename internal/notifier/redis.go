package notifier

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisNotifier implements Notifier interface using Redis pub/sub
type RedisNotifier struct {
	client   *redis.Client
	pubsub   *redis.PubSub
	patterns []string
}

// NewRedisNotifier creates a new Redis notifier
func NewRedisNotifier(addr string) (*RedisNotifier, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisNotifier{
		client: client,
	}, nil
}

// Subscribe to Redis keyspace notifications
func (rn *RedisNotifier) Subscribe(ctx context.Context, patterns []string) (<-chan StorageEvent, error) {
	if len(patterns) == 0 {
		return nil, fmt.Errorf("no patterns provided")
	}

	// Convert patterns to keyspace notification patterns
	keyspacePatterns := make([]string, len(patterns))
	for i, pattern := range patterns {
		keyspacePatterns[i] = fmt.Sprintf("__keyspace@0__:%s", pattern)
	}

	// Subscribe to patterns
	rn.pubsub = rn.client.PSubscribe(ctx, keyspacePatterns...)
	rn.patterns = keyspacePatterns

	// Create event buffered channel
	eventChan := make(chan StorageEvent, 100)

	// Start goroutine to process messages
	go func() {
		defer close(eventChan)

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-rn.pubsub.Channel():
				if !ok {
					return
				}

				event := rn.parseMessage(msg)
				if event != nil {
					select {
					case eventChan <- *event:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return eventChan, nil
}

// parseMessage converts Redis message to StorageEvent
func (rn *RedisNotifier) parseMessage(msg *redis.Message) *StorageEvent {
	if msg == nil {
		return nil
	}

	// Extract key from keyspace notification
	// Channel format: __keyspace@0__:radius:acct:user:session:timestamp
	// Payload: operation (set, expire, del, etc.)

	if !strings.HasPrefix(msg.Channel, "__keyspace@0__:") {
		return nil
	}

	key := strings.TrimPrefix(msg.Channel, "__keyspace@0__:")
	operation := msg.Payload

	return &StorageEvent{
		Key:       key,
		Operation: operation,
		Timestamp: time.Now(),
	}
}

// Unsubscribe from patterns
func (rn *RedisNotifier) Unsubscribe(patterns []string) error {
	if rn.pubsub == nil {
		return fmt.Errorf("not subscribed")
	}

	keyspacePatterns := make([]string, len(patterns))
	for i, pattern := range patterns {
		keyspacePatterns[i] = fmt.Sprintf("__keyspace@0__:%s", pattern)
	}

	return rn.pubsub.PUnsubscribe(context.Background(), keyspacePatterns...)
}

// HealthCheck verifies Redis connectivity
func (rn *RedisNotifier) HealthCheck(ctx context.Context) error {
	return rn.client.Ping(ctx).Err()
}

// closes the notifier and cleans up resources
func (rn *RedisNotifier) Close() error {
	var err error

	if rn.pubsub != nil {
		err = rn.pubsub.Close()
	}

	if rn.client != nil {
		if closeErr := rn.client.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}

	return err
}
