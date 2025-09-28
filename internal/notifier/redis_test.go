package notifier

import (
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRedisNotifier_parseMessage(t *testing.T) {
	notifier := &RedisNotifier{}

	tests := []struct {
		name     string
		msg      *redis.Message
		expected *StorageEvent
	}{
		{
			name: "valid accounting key",
			msg: &redis.Message{
				Channel: "__keyspace@0__:radius:acct:testuser:session123:2024-01-15T10:30:45Z",
				Payload: "set",
			},
			expected: &StorageEvent{
				Key:       "radius:acct:testuser:session123:2024-01-15T10:30:45Z",
				Operation: "set",
			},
		},
		{
			name: "expire operation",
			msg: &redis.Message{
				Channel: "__keyspace@0__:radius:acct:user:session:timestamp",
				Payload: "expire",
			},
			expected: &StorageEvent{
				Key:       "radius:acct:user:session:timestamp",
				Operation: "expire",
			},
		},
		{
			name: "invalid channel format",
			msg: &redis.Message{
				Channel: "invalid:channel",
				Payload: "set",
			},
			expected: nil,
		},
		{
			name:     "nil message",
			msg:      nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := notifier.parseMessage(tt.msg)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("Expected event, got nil")
			}

			if result.Key != tt.expected.Key {
				t.Errorf("Expected key %s, got %s", tt.expected.Key, result.Key)
			}

			if result.Operation != tt.expected.Operation {
				t.Errorf("Expected operation %s, got %s", tt.expected.Operation, result.Operation)
			}
		})
	}
}
