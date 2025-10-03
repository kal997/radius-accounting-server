package notifier

import (
	"context"
	"net"
	"testing"
	"time"
	
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRedisNotifier(t *testing.T) {
	tests := []struct {
		name        string
		setupRedis  func() (string, func())
		wantErr     bool
		errContains string
	}{
		{
			name: "successful connection",
			setupRedis: func() (string, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				return mr.Addr(), func() { mr.Close() }
			},
			wantErr: false,
		},
		{
			name: "connection failure - invalid address",
			setupRedis: func() (string, func()) {
				return "invalid:address:format", func() {}
			},
			wantErr:     true,
			errContains: "failed to connect to Redis",
		},
		{
			name: "connection failure - unreachable host",
			setupRedis: func() (string, func()) {
				return "127.0.0.1:59999", func() {}
			},
			wantErr:     true,
			errContains: "failed to connect to Redis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, cleanup := tt.setupRedis()
			defer cleanup()

			notifier, err := NewRedisNotifier(addr)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, notifier)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, notifier)
				assert.NotNil(t, notifier.client)
				notifier.Close()
			}
		})
	}
}

func TestRedisNotifier_Subscribe(t *testing.T) {
	tests := []struct {
		name        string
		patterns    []string
		setupRedis  func() (*RedisNotifier, func())
		wantErr     bool
		errContains string
		validate    func(*testing.T, <-chan StorageEvent)
	}{
		{
			name:     "successful subscription with single pattern",
			patterns: []string{"radius:acct:*"},
			setupRedis: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)
				return notifier, func() {
					notifier.Close()
					mr.Close()
				}
			},
			wantErr: false,
			validate: func(t *testing.T, ch <-chan StorageEvent) {
				assert.NotNil(t, ch)
			},
		},
		{
			name:     "successful subscription with multiple patterns",
			patterns: []string{"radius:acct:*", "radius:auth:*", "radius:session:*"},
			setupRedis: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)
				return notifier, func() {
					notifier.Close()
					mr.Close()
				}
			},
			wantErr: false,
			validate: func(t *testing.T, ch <-chan StorageEvent) {
				assert.NotNil(t, ch)
			},
		},
		{
			name:     "empty patterns error",
			patterns: []string{},
			setupRedis: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)
				return notifier, func() {
					notifier.Close()
					mr.Close()
				}
			},
			wantErr:     true,
			errContains: "no patterns provided",
		},
		{
			name:     "nil patterns slice",
			patterns: nil,
			setupRedis: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)
				return notifier, func() {
					notifier.Close()
					mr.Close()
				}
			},
			wantErr:     true,
			errContains: "no patterns provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier, cleanup := tt.setupRedis()
			defer cleanup()

			ctx := context.Background()
			eventChan, err := notifier.Subscribe(ctx, tt.patterns)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, eventChan)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, eventChan)
				assert.NotNil(t, notifier.pubsub)
				assert.Equal(t, len(tt.patterns), len(notifier.patterns))

				// Verify pattern format
				for i, pattern := range tt.patterns {
					expectedPattern := "__keyspace@0__:" + pattern
					assert.Equal(t, expectedPattern, notifier.patterns[i])
				}

				if tt.validate != nil {
					tt.validate(t, eventChan)
				}
			}
		})
	}
}

func TestRedisNotifier_Subscribe_ContextCancellation(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	notifier, err := NewRedisNotifier(mr.Addr())
	require.NoError(t, err)
	defer notifier.Close()

	ctx, cancel := context.WithCancel(context.Background())
	eventChan, err := notifier.Subscribe(ctx, []string{"test:*"})
	require.NoError(t, err)

	// Cancel context
	cancel()

	// Channel should be closed shortly
	select {
	case _, ok := <-eventChan:
		assert.False(t, ok, "channel should be closed")
	case <-time.After(100 * time.Millisecond):
		// Give it a bit more time
		_, ok := <-eventChan
		assert.False(t, ok, "channel should be closed after context cancellation")
	}
}

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
			name: "del operation",
			msg: &redis.Message{
				Channel: "__keyspace@0__:some:key",
				Payload: "del",
			},
			expected: &StorageEvent{
				Key:       "some:key",
				Operation: "del",
			},
		},
		{
			name: "invalid channel format - missing prefix",
			msg: &redis.Message{
				Channel: "invalid:channel",
				Payload: "set",
			},
			expected: nil,
		},
		{
			name: "invalid channel format - wrong prefix",
			msg: &redis.Message{
				Channel: "__keyspace@1__:some:key",
				Payload: "set",
			},
			expected: nil,
		},
		{
			name: "empty channel",
			msg: &redis.Message{
				Channel: "",
				Payload: "set",
			},
			expected: nil,
		},
		{
			name: "empty payload",
			msg: &redis.Message{
				Channel: "__keyspace@0__:some:key",
				Payload: "",
			},
			expected: &StorageEvent{
				Key:       "some:key",
				Operation: "",
			},
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
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tt.expected.Key, result.Key)
			assert.Equal(t, tt.expected.Operation, result.Operation)
			assert.WithinDuration(t, time.Now(), result.Timestamp, 100*time.Millisecond)
		})
	}
}

func TestRedisNotifier_Unsubscribe(t *testing.T) {
	tests := []struct {
		name          string
		setupNotifier func() (*RedisNotifier, func())
		patterns      []string
		wantErr       bool
		errContains   string
	}{
		{
			name: "successful unsubscribe",
			setupNotifier: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)

				// Subscribe first
				ctx := context.Background()
				_, err = notifier.Subscribe(ctx, []string{"test:*"})
				require.NoError(t, err)

				return notifier, func() {
					notifier.Close()
					mr.Close()
				}
			},
			patterns: []string{"test:*"},
			wantErr:  false,
		},
		{
			name: "unsubscribe without subscription",
			setupNotifier: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)
				return notifier, func() {
					notifier.Close()
					mr.Close()
				}
			},
			patterns:    []string{"test:*"},
			wantErr:     true,
			errContains: "not subscribed",
		},
		{
			name: "unsubscribe multiple patterns",
			setupNotifier: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)

				// Subscribe to multiple patterns
				ctx := context.Background()
				_, err = notifier.Subscribe(ctx, []string{"test:*", "radius:*", "session:*"})
				require.NoError(t, err)

				return notifier, func() {
					notifier.Close()
					mr.Close()
				}
			},
			patterns: []string{"test:*", "radius:*"},
			wantErr:  false,
		},
		{
			name: "empty patterns",
			setupNotifier: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)

				ctx := context.Background()
				_, err = notifier.Subscribe(ctx, []string{"test:*"})
				require.NoError(t, err)

				return notifier, func() {
					notifier.Close()
					mr.Close()
				}
			},
			patterns: []string{},
			wantErr:  false, // PUnsubscribe with empty patterns is valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier, cleanup := tt.setupNotifier()
			defer cleanup()

			err := notifier.Unsubscribe(tt.patterns)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisNotifier_HealthCheck(t *testing.T) {
	tests := []struct {
		name          string
		setupNotifier func() (*RedisNotifier, func())
		wantErr       bool
	}{
		{
			name: "healthy connection",
			setupNotifier: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)
				return notifier, func() {
					notifier.Close()
					mr.Close()
				}
			},
			wantErr: false,
		},
		{
			name: "unhealthy connection - server stopped",
			setupNotifier: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)

				// Stop the server to make connection unhealthy
				mr.Close()

				return notifier, func() {
					notifier.Close()
				}
			},
			wantErr: true,
		},
		{
			name: "health check with context timeout",
			setupNotifier: func() (*RedisNotifier, func()) {
				// Create a listener that accepts but doesn't respond
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)

				go func() {
					for {
						conn, err := listener.Accept()
						if err != nil {
							return
						}
						// Accept but don't respond - simulate hanging connection
						defer conn.Close()
						time.Sleep(1 * time.Second)
					}
				}()

				// Create notifier with custom client
				notifier := &RedisNotifier{
					client: redis.NewClient(&redis.Options{
						Addr:        listener.Addr().String(),
						DialTimeout: 10 * time.Millisecond,
					}),
				}

				return notifier, func() {
					notifier.Close()
					listener.Close()
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier, cleanup := tt.setupNotifier()
			defer cleanup()

			ctx := context.Background()
			err := notifier.HealthCheck(ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisNotifier_Close(t *testing.T) {
	tests := []struct {
		name          string
		setupNotifier func() (*RedisNotifier, func())
		wantErr       bool
	}{
		{
			name: "close with only client",
			setupNotifier: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)
				return notifier, func() { mr.Close() }
			},
			wantErr: false,
		},
		{
			name: "close with client and pubsub",
			setupNotifier: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)

				// Subscribe to create pubsub
				ctx := context.Background()
				_, err = notifier.Subscribe(ctx, []string{"test:*"})
				require.NoError(t, err)

				return notifier, func() { mr.Close() }
			},
			wantErr: false,
		},
		{
			name: "close nil client and pubsub",
			setupNotifier: func() (*RedisNotifier, func()) {
				return &RedisNotifier{}, func() {}
			},
			wantErr: false,
		},
		{
			name: "close already closed",
			setupNotifier: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)

				// Close once
				notifier.Close()

				return notifier, func() { mr.Close() }
			},
			wantErr: true,
		},
		{
			name: "close with pubsub error",
			setupNotifier: func() (*RedisNotifier, func()) {
				mr, err := miniredis.Run()
				require.NoError(t, err)
				notifier, err := NewRedisNotifier(mr.Addr())
				require.NoError(t, err)

				ctx := context.Background()
				_, err = notifier.Subscribe(ctx, []string{"test:*"})
				require.NoError(t, err)

				// Close pubsub first to simulate error scenario
				notifier.pubsub.Close()

				return notifier, func() { mr.Close() }
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier, cleanup := tt.setupNotifier()
			defer cleanup()

			err := notifier.Close()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisNotifier_MessageProcessing(t *testing.T) {
	// This test simulates actual message processing through the goroutine
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	notifier, err := NewRedisNotifier(mr.Addr())
	require.NoError(t, err)
	defer notifier.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventChan, err := notifier.Subscribe(ctx, []string{"test:*"})
	require.NoError(t, err)

	// Simulate messages being sent to the pubsub channel
	// Note: miniredis doesn't support keyspace notifications, so we'd need
	// to mock this differently in a real implementation

	// Test context cancellation during message processing
	cancel()

	// Verify channel closes
	time.Sleep(50 * time.Millisecond)
	select {
	case _, ok := <-eventChan:
		assert.False(t, ok, "channel should be closed")
	default:
		t.Error("channel should be closed after context cancellation")
	}
}

func TestRedisNotifier_ConcurrentOperations(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	notifier, err := NewRedisNotifier(mr.Addr())
	require.NoError(t, err)
	defer notifier.Close()

	ctx := context.Background()

	// Concurrent subscriptions
	done := make(chan bool, 3)

	go func() {
		_, err := notifier.Subscribe(ctx, []string{"pattern1:*"})
		assert.NoError(t, err)
		done <- true
	}()

	go func() {
		err := notifier.HealthCheck(ctx)
		assert.NoError(t, err)
		done <- true
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)
		err := notifier.Unsubscribe([]string{"pattern1:*"})
		// May or may not error depending on timing
		_ = err
		done <- true
	}()

	// Wait for all operations
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Error("concurrent operation timeout")
		}
	}
}

// MockRedisClient for testing error scenarios
type MockRedisClient struct {
	*redis.Client
	pingErr error
}

func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	if m.pingErr != nil {
		cmd.SetErr(m.pingErr)
	}
	return cmd
}

func TestRedisNotifier_PingError(t *testing.T) {
	// Test specific ping error scenario
	notifier := &RedisNotifier{
		client: redis.NewClient(&redis.Options{
			Addr: "unreachable:6379",
		}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := notifier.HealthCheck(ctx)
	assert.Error(t, err)
}
