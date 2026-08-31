package redis

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis represents a Redis client with connection management and reconnection support
type Redis struct {
	client   *redis.Client
	config   *RedisConfig
	done     chan bool
	mu       sync.RWMutex
	isClosed bool
}

// NewRedis creates a new Redis client with connection management
func NewRedis(ctx context.Context, config *RedisConfig) (*Redis, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Apply defaults
	applyDefaults(config)

	r := &Redis{
		config: config,
		done:   make(chan bool),
	}

	// Create client
	r.client = r.createClient(config)

	// Initial connection with ping
	if err := r.client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("Connected to Redis at %s", config.Addr)

	// Start connection monitor
	go r.monitor()

	return r, nil
}

// NewRedisFromURL creates a new Redis client from a URL string
func NewRedisFromURL(ctx context.Context, url string) (*Redis, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	config := &RedisConfig{
		Addr:     opt.Addr,
		Password: opt.Password,
		DB:       opt.DB,
	}

	return NewRedis(ctx, config)
}

// createClient creates a new redis.Client from config
func (r *Redis) createClient(config *RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:            config.Addr,
		Password:        config.Password,
		DB:              config.DB,
		PoolSize:        config.PoolSize,
		MinIdleConns:    config.MinIdleConns,
		MaxIdleConns:    config.MaxIdleConns,
		DialTimeout:     config.DialTimeout,
		ReadTimeout:     config.ReadTimeout,
		WriteTimeout:    config.WriteTimeout,
		PoolTimeout:     config.PoolTimeout,
		MaxRetries:      config.MaxRetries,
		MinRetryBackoff: config.MinRetryBackoff,
		MaxRetryBackoff: config.MaxRetryBackoff,
		ConnMaxIdleTime: config.ConnMaxIdleTime,
		ConnMaxLifetime: config.ConnMaxLifetime,
	})
}

// monitor monitors the connection and reconnects if needed
func (r *Redis) monitor() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			r.mu.RLock()
			if r.isClosed {
				r.mu.RUnlock()
				return
			}
			client := r.client
			r.mu.RUnlock()

			// Check connection
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := client.Ping(ctx).Err()
			cancel()

			if err != nil {
				log.Printf("Redis connection lost, attempting to reconnect...")
				r.reconnect()
			}
		}
	}
}

// reconnect attempts to reconnect to Redis
func (r *Redis) reconnect() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isClosed {
		return
	}

	// Close existing client
	if r.client != nil {
		r.client.Close()
	}

	// Create new client
	r.client = r.createClient(r.config)

	// Wait for connection
	maxRetries := 5
	for i := 1; i <= maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := r.client.Ping(ctx).Err()
		cancel()

		if err == nil {
			log.Printf("Redis reconnected successfully")
			return
		}

		log.Printf("Redis reconnect attempt %d/%d failed: %v", i, maxRetries, err)
		time.Sleep(time.Duration(i) * time.Second)
	}

	log.Printf("Failed to reconnect to Redis after %d attempts", maxRetries)
}

// GetClient returns the underlying redis.Client
func (r *Redis) GetClient() *redis.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

// GetContext returns the client with context support
func (r *Redis) GetContext() *redis.Client {
	return r.GetClient()
}

// Close closes the Redis connection
func (r *Redis) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isClosed {
		return nil
	}

	r.isClosed = true
	close(r.done)

	if r.client != nil {
		return r.client.Close()
	}

	return nil
}

// HealthCheck checks if the connection is alive
func (r *Redis) HealthCheck() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.isClosed {
		return fmt.Errorf("Redis connection is closed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}

	return nil
}

// applyDefaults sets default values for Redis config
func applyDefaults(config *RedisConfig) {
	if config.PoolSize == 0 {
		config.PoolSize = 10
	}
	if config.MinIdleConns == 0 {
		config.MinIdleConns = 5
	}
	if config.MaxIdleConns == 0 {
		config.MaxIdleConns = 10
	}
	if config.DialTimeout == 0 {
		config.DialTimeout = 5 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 3 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 3 * time.Second
	}
	if config.PoolTimeout == 0 {
		config.PoolTimeout = 4 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.MinRetryBackoff == 0 {
		config.MinRetryBackoff = 8 * time.Millisecond
	}
	if config.MaxRetryBackoff == 0 {
		config.MaxRetryBackoff = 512 * time.Millisecond
	}
	if config.ConnMaxIdleTime == 0 {
		config.ConnMaxIdleTime = 30 * time.Minute
	}
}

// DefaultConfig returns a default Redis configuration
func DefaultConfig() *RedisConfig {
	return &RedisConfig{
		PoolSize:        10,
		MinIdleConns:    5,
		MaxIdleConns:    10,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		ConnMaxIdleTime: 30 * time.Minute,
	}
}
