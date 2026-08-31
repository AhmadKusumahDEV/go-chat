package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrLockNotAcquired is returned when the lock cannot be acquired
	ErrLockNotAcquired = errors.New("lock not acquired")
	// ErrLockNotHeld is returned when trying to release a lock that is not held
	ErrLockNotHeld = errors.New("lock not held")
)

// Lock represents a distributed lock
type Lock struct {
	client *redis.Client
	key    string
	value  string
	ttl    time.Duration
}

// LockConfig holds configuration for distributed locking
type LockConfig struct {
	// RetryAttempts is the number of times to retry acquiring the lock
	RetryAttempts int
	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration
	// LockTTL is how long the lock should be held before auto-expiring
	LockTTL time.Duration
}

// DefaultLockConfig returns a default lock configuration
func DefaultLockConfig() *LockConfig {
	return &LockConfig{
		RetryAttempts: 3,
		RetryDelay:    100 * time.Millisecond,
		LockTTL:       30 * time.Second,
	}
}

// DistributedLock provides distributed locking functionality using Redis
type DistributedLock struct {
	client *redis.Client
	config *LockConfig
}

// NewDistributedLock creates a new distributed lock manager
func NewDistributedLock(client *redis.Client, config *LockConfig) *DistributedLock {
	if config == nil {
		config = DefaultLockConfig()
	}
	return &DistributedLock{
		client: client,
		config: config,
	}
}

// Acquire attempts to acquire a lock with the given key
// Returns a Lock if successful, error if lock cannot be acquired
func (d *DistributedLock) Acquire(ctx context.Context, key string) (*Lock, error) {
	return d.AcquireWithTTL(ctx, key, d.config.LockTTL)
}

// AcquireWithTTL acquires a lock with a custom TTL
func (d *DistributedLock) AcquireWithTTL(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
	value := generateLockValue()
	lock := &Lock{
		client: d.client,
		key:    key,
		value:  value,
		ttl:    ttl,
	}

	for attempt := 0; attempt <= d.config.RetryAttempts; attempt++ {
		// Try to acquire lock using SETNX
		acquired, err := d.client.SetNX(ctx, key, value, ttl).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to acquire lock: %w", err)
		}

		if acquired {
			return lock, nil
		}

		// Lock not acquired, retry if attempts remaining
		if attempt < d.config.RetryAttempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(d.config.RetryDelay):
				// Continue to next attempt
			}
		}
	}

	return nil, ErrLockNotAcquired
}

// AcquireBlocking acquires a lock with blocking behavior
// Blocks until lock is acquired or context is cancelled
func (d *DistributedLock) AcquireBlocking(ctx context.Context, key string) (*Lock, error) {
	return d.AcquireBlockingWithTTL(ctx, key, d.config.LockTTL)
}

// AcquireBlockingWithTTL acquires a lock with blocking behavior and custom TTL
func (d *DistributedLock) AcquireBlockingWithTTL(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
	value := generateLockValue()
	lock := &Lock{
		client: d.client,
		key:    key,
		value:  value,
		ttl:    ttl,
	}

	for {
		acquired, err := d.client.SetNX(ctx, key, value, ttl).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to acquire lock: %w", err)
		}

		if acquired {
			return lock, nil
		}

		// Wait for lock to be released
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Continue trying
		}
	}
}

// TryAcquire attempts to acquire a lock once without retrying
func (d *DistributedLock) TryAcquire(ctx context.Context, key string) (*Lock, error) {
	return d.TryAcquireWithTTL(ctx, key, d.config.LockTTL)
}

// TryAcquireWithTTL attempts to acquire a lock once with custom TTL
func (d *DistributedLock) TryAcquireWithTTL(ctx context.Context, key string, ttl time.Duration) (*Lock, error) {
	value := generateLockValue()

	acquired, err := d.client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		return nil, ErrLockNotAcquired
	}

	return &Lock{
		client: d.client,
		key:    key,
		value:  value,
		ttl:    ttl,
	}, nil
}

// Release releases the lock
// Only releases if the lock is still held by this instance (using the stored value)
func (l *Lock) Release(ctx context.Context) error {
	// Use Lua script to atomically check and delete
	// This ensures we only delete if the value matches (we still hold the lock)
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, l.client, []string{l.key}, l.value).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result.(int64) == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// Extend extends the lock TTL
func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	// Use Lua script to atomically check and extend
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, l.client, []string{l.key}, l.value, int64(ttl/time.Millisecond)).Result()
	if err != nil {
		return fmt.Errorf("failed to extend lock: %w", err)
	}

	if result.(int64) == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// Key returns the lock key
func (l *Lock) Key() string {
	return l.key
}

// generateLockValue generates a unique value for the lock
func generateLockValue() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(16))
}

// randomString generates a random string of given length
func randomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
