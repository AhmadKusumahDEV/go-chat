package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache provides caching functionality using Redis
type Cache struct {
	client *redis.Client
	prefix string
}

// CacheConfig holds configuration for caching
type CacheConfig struct {
	// Prefix is the key prefix for all cache entries
	Prefix string
	// DefaultTTL is the default TTL for cache entries
	DefaultTTL time.Duration
	// EnableSerialization enables JSON serialization for complex types
	EnableSerialization bool
}

// DefaultCacheConfig returns a default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		Prefix:              "cache:",
		DefaultTTL:          30 * time.Minute,
		EnableSerialization: true,
	}
}

// NewCache creates a new Cache instance
func NewCache(client *redis.Client, config *CacheConfig) *Cache {
	if config == nil {
		config = DefaultCacheConfig()
	}
	return &Cache{
		client: client,
		prefix: config.Prefix,
	}
}

// Set stores a value in the cache
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := c.prefix + key

	var data []byte
	var err error

	// Try to serialize if it's a complex type
	if v, ok := value.([]byte); ok {
		data = v
	} else if v, ok := value.(string); ok {
		data = []byte(v)
	} else {
		data, err = json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal cache value: %w", err)
		}
	}

	if err := c.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// SetDefault stores a value in the cache with the default TTL
func (c *Cache) SetDefault(ctx context.Context, key string, value interface{}) error {
	return c.Set(ctx, key, value, DefaultCacheConfig().DefaultTTL)
}

// Get retrieves a value from the cache
// Returns the value and whether it was found
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	fullKey := c.prefix + key

	val, err := c.client.Get(ctx, fullKey).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get cache: %w", err)
	}

	return val, nil
}

// GetStruct retrieves and unmarshals a struct from the cache
func (c *Cache) GetStruct(ctx context.Context, key string, dest interface{}) error {
	val, err := c.Get(ctx, key)
	if err != nil {
		return err
	}
	if val == "" {
		return nil // Not found, caller should handle
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache value: %w", err)
	}

	return nil
}

// GetTTL returns the remaining TTL for a key
func (c *Cache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := c.prefix + key

	ttl, err := c.client.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	return ttl, nil
}

// Delete removes a value from the cache
func (c *Cache) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key

	if err := c.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	return nil
}

// DeletePattern deletes all keys matching the pattern
func (c *Cache) DeletePattern(ctx context.Context, pattern string) (int64, error) {
	fullPattern := c.prefix + pattern

	var deleted int64
	iter := c.client.Scan(ctx, 0, fullPattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			log.Printf("Failed to delete key %s: %v", iter.Val(), err)
			continue
		}
		deleted++
	}

	if err := iter.Err(); err != nil {
		return deleted, fmt.Errorf("failed to scan keys: %w", err)
	}

	return deleted, nil
}

// Exists checks if a key exists in the cache
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.prefix + key

	result, err := c.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return result > 0, nil
}

// Increment increments a numeric value
func (c *Cache) Increment(ctx context.Context, key string) (int64, error) {
	fullKey := c.prefix + key

	val, err := c.client.Incr(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment: %w", err)
	}

	return val, nil
}

// IncrementBy increments a numeric value by delta
func (c *Cache) IncrementBy(ctx context.Context, key string, delta int64) (int64, error) {
	fullKey := c.prefix + key

	val, err := c.client.IncrBy(ctx, fullKey, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment: %w", err)
	}

	return val, nil
}

// Decrement decrements a numeric value
func (c *Cache) Decrement(ctx context.Context, key string) (int64, error) {
	fullKey := c.prefix + key

	val, err := c.client.Decr(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to decrement: %w", err)
	}

	return val, nil
}

// SetNX sets a value only if it doesn't exist (for distributed locking)
func (c *Cache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	fullKey := c.prefix + key

	var data string
	switch v := value.(type) {
	case string:
		data = v
	default:
		jsonData, err := json.Marshal(value)
		if err != nil {
			return false, fmt.Errorf("failed to marshal value: %w", err)
		}
		data = string(jsonData)
	}

	result, err := c.client.SetNX(ctx, fullKey, data, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to setNX: %w", err)
	}

	return result, nil
}

// GetOrSet gets a value from cache or sets it using the provided function
func (c *Cache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error), dest interface{}) error {
	// Try to get from cache
	err := c.GetStruct(ctx, key, dest)
	if err == nil {
		return nil // Found in cache
	}

	// Not found, call function to get value
	value, err := fn()
	if err != nil {
		return fmt.Errorf("failed to get value: %w", err)
	}

	// Store in cache
	if err := c.Set(ctx, key, value, ttl); err != nil {
		log.Printf("Failed to cache value: %v", err)
		// Don't return error, we have the value
	}

	// Unmarshal to dest
	if dest != nil {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		if err := json.Unmarshal(data, dest); err != nil {
			return fmt.Errorf("failed to unmarshal to dest: %w", err)
		}
	}

	return nil
}

// HashSet sets a hash field
func (c *Cache) HashSet(ctx context.Context, key string, field string, value interface{}) error {
	fullKey := c.prefix + key

	var data string
	switch v := value.(type) {
	case string:
		data = v
	default:
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		data = string(jsonData)
	}

	if err := c.client.HSet(ctx, fullKey, field, data).Err(); err != nil {
		return fmt.Errorf("failed to hash set: %w", err)
	}

	return nil
}

// HashGet retrieves a hash field
func (c *Cache) HashGet(ctx context.Context, key string, field string) (string, error) {
	fullKey := c.prefix + key

	val, err := c.client.HGet(ctx, fullKey, field).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to hash get: %w", err)
	}

	return val, nil
}

// HashGetAll retrieves all hash fields
func (c *Cache) HashGetAll(ctx context.Context, key string) (map[string]string, error) {
	fullKey := c.prefix + key

	val, err := c.client.HGetAll(ctx, fullKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to hash get all: %w", err)
	}

	return val, nil
}

// HashDelete deletes a hash field
func (c *Cache) HashDelete(ctx context.Context, key string, fields ...string) error {
	fullKey := c.prefix + key

	if err := c.client.HDel(ctx, fullKey, fields...).Err(); err != nil {
		return fmt.Errorf("failed to hash delete: %w", err)
	}

	return nil
}

// HashIncrBy increments a hash field by delta
func (c *Cache) HashIncrBy(ctx context.Context, key string, field string, delta int64) (int64, error) {
	fullKey := c.prefix + key

	val, err := c.client.HIncrBy(ctx, fullKey, field, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to hash increment: %w", err)
	}

	return val, nil
}

// ListPush pushes a value to a list
func (c *Cache) ListPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	fullKey := c.prefix + key

	val, err := c.client.RPush(ctx, fullKey, values...).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to list push: %w", err)
	}

	return val, nil
}

// ListRange returns a range of elements from a list
func (c *Cache) ListRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	fullKey := c.prefix + key

	val, err := c.client.LRange(ctx, fullKey, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list range: %w", err)
	}

	return val, nil
}

// ListLen returns the length of a list
func (c *Cache) ListLen(ctx context.Context, key string) (int64, error) {
	fullKey := c.prefix + key

	val, err := c.client.LLen(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to list len: %w", err)
	}

	return val, nil
}

// SetAdd adds a member to a set
func (c *Cache) SetAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	fullKey := c.prefix + key

	val, err := c.client.SAdd(ctx, fullKey, members...).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to set add: %w", err)
	}

	return val, nil
}

// SetMembers returns all members of a set
func (c *Cache) SetMembers(ctx context.Context, key string) ([]string, error) {
	fullKey := c.prefix + key

	val, err := c.client.SMembers(ctx, fullKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to set members: %w", err)
	}

	return val, nil
}

// SetIsMember checks if a value is a member of a set
func (c *Cache) SetIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	fullKey := c.prefix + key

	val, err := c.client.SIsMember(ctx, fullKey, member).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set is member: %w", err)
	}

	return val, nil
}
