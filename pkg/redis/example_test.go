package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// ExampleUsage demonstrates various ways to use the Redis package
func ExampleUsage() {
	ctx := context.Background()

	// 1. Create Redis client with config
	config := &RedisConfig{
		Addr:            "localhost:6379",
		Password:        "",
		DB:              0,
		PoolSize:        10,
		MinIdleConns:    5,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	}

	// Using NewRedis (recommended for production)
	client, err := NewRedis(ctx, config)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Or using NewRedisFromURL
	client2, err := NewRedisFromURL(ctx, "redis://localhost:6379")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client2.Close()
}

// ExampleCache demonstrates caching functionality
func ExampleCache() {
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	cache := NewCache(redisClient, nil)

	// Simple string caching
	cache.Set(ctx, "user:1:name", "John", 30*time.Minute)

	// Struct caching
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	user := User{Name: "John", Email: "john@example.com"}
	cache.Set(ctx, "user:1", user, 30*time.Minute)

	// Get from cache
	name, _ := cache.Get(ctx, "user:1:name")
	log.Printf("Name: %s", name)

	// Get struct from cache
	var cachedUser User
	cache.GetStruct(ctx, "user:1", &cachedUser)
	log.Printf("User: %+v", cachedUser)

	// Get or set pattern
	var result User
	cache.GetOrSet(ctx, "user:2", 30*time.Minute, func() (interface{}, error) {
		// Simulate fetching from database
		return User{Name: "Jane", Email: "jane@example.com"}, nil
	}, &result)
	log.Printf("Result: %+v", result)
}

// ExampleDistributedLock demonstrates distributed locking
func ExampleDistributedLock() {
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	lock := NewDistributedLock(redisClient, nil)

	// Acquire lock
	acquired, err := lock.Acquire(ctx, "my-resource-lock")
	if err != nil {
		log.Printf("Failed to acquire lock: %v", err)
		return
	}
	defer acquired.Release(ctx)

	// Do work while holding the lock
	log.Println("Lock acquired, doing work...")
	time.Sleep(5 * time.Second)
	log.Println("Work complete")
}

// ExampleDistributedLockWithRetry demonstrates lock with retry
func ExampleDistributedLockWithRetry() {
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	lock := NewDistributedLock(redisClient, &LockConfig{
		RetryAttempts: 5,
		RetryDelay:    100 * time.Millisecond,
		LockTTL:       30 * time.Second,
	})

	// Try to acquire lock with retry
	acquired, err := lock.Acquire(ctx, "payment-processing")
	if err != nil {
		log.Printf("Could not acquire lock: %v", err)
		return
	}
	defer acquired.Release(ctx)

	log.Println("Lock acquired with retry")
}

// ExamplePubSub demonstrates publish/subscribe
func ExamplePubSub() {
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	pubsub := NewPubSub(redisClient)

	// Subscribe to channels
	err := pubsub.Subscribe(ctx, func(msg *PubSubMessage) {
		log.Printf("Received on %s: %v", msg.Channel, msg.Payload)
	}, "notifications", "alerts")

	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Publish messages
	pubsub.Publish(ctx, "notifications", "Hello from Redis!")
	pubsub.Publish(ctx, "alerts", map[string]string{"level": "warning", "message": "Test alert"})
}

// ExamplePubSubManager demonstrates managing multiple subscriptions
func ExamplePubSubManager() {
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	manager := NewPubSubManager(redisClient)
	defer manager.Close()

	// Create subscription for notifications
	manager.Subscribe(ctx, "notifications", func(msg *PubSubMessage) {
		log.Printf("Notification: %v", msg.Payload)
	}, "notifications")

	// Create subscription for alerts
	manager.SubscribePattern(ctx, "alerts", func(msg *PubSubMessage) {
		log.Printf("Alert: %v", msg.Payload)
	}, "alerts:*")

	// Publish messages
	manager.Publish(ctx, "notifications", "Hello!")
	manager.Publish(ctx, "alerts:warning", "This is a warning")
	manager.Publish(ctx, "alerts:error", "This is an error")
}

// ExampleStream demonstrates Redis Streams
func ExampleStream() {
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	stream := NewStream(redisClient)

	// Create consumer group (if not exists)
	stream.CreateGroup(ctx, "mystream", "mygroup")

	// Add messages to stream
	id, err := stream.Add(ctx, "mystream", map[string]interface{}{
		"event": "user.created",
		"data":  `{"user_id": "123", "email": "john@example.com"}`,
	})
	if err != nil {
		log.Printf("Failed to add to stream: %v", err)
	} else {
		log.Printf("Added message with ID: %s", id)
	}

	// Read messages (blocking)
	messages, err := stream.Read(ctx, "mystream", "mygroup", "consumer1", 10)
	if err != nil {
		log.Printf("Failed to read from stream: %v", err)
	}

	for _, msg := range messages {
		log.Printf("Received message: %+v", msg)
		// Process message...

		// Acknowledge message
		stream.Ack(ctx, "mystream", "mygroup", msg.ID)
	}
}

// ExampleIdempotency demonstrates idempotency using SETNX
func ExampleIdempotency() {
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	cache := NewCache(redisClient, nil)

	// Idempotency key for payment processing
	idempotencyKey := "payment:order-123:retry"

	// Try to acquire (simulate idempotency check)
	acquired, err := cache.SetNX(ctx, idempotencyKey, "processing", 24*time.Hour)
	if err != nil {
		log.Printf("Error checking idempotency: %v", err)
		return
	}

	if !acquired {
		log.Printf("Request already processed (duplicate)")
		return
	}

	// Process the payment...
	log.Println("Processing payment...")

	// On failure, you might want to release the key to allow retry
	// cache.Delete(ctx, idempotencyKey)
}

// ExampleRateLimiting demonstrates simple rate limiting
func ExampleRateLimiting() {
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	cache := NewCache(redisClient, nil)

	// Simple rate limiting: max 100 requests per minute per user
	key := "ratelimit:user-123:minute"
	limit := 100

	// Increment counter
	count, err := cache.Increment(ctx, key)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// Set expiry on first request
	if count == 1 {
		cache.Set(ctx, key+":expire", "1", time.Minute)
	}

	if count > int64(limit) {
		log.Printf("Rate limit exceeded: %d/%d", count, limit)
		return
	}

	log.Printf("Request allowed: %d/%d", count, limit)
}

// ExampleUserProfileCache demonstrates caching user profiles
func ExampleUserProfileCache() {
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	type Profile struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		PhotoURL string `json:"photo_url"`
	}

	cache := NewCache(redisClient, nil)

	userID := "user-123"

	// Try to get from cache
	var profile Profile
	err := cache.GetStruct(ctx, "profile:"+userID, &profile)

	if err == nil && profile.ID != "" {
		log.Printf("Cache hit: %+v", profile)
		return
	}

	// Cache miss - fetch from database
	profile = Profile{
		ID:       userID,
		Name:     "John Doe",
		Email:    "john@example.com",
		PhotoURL: "https://example.com/photos/123.jpg",
	}

	// Store in cache
	cache.Set(ctx, "profile:"+userID, profile, 15*time.Minute)
	log.Printf("Cache miss, fetched and stored: %+v", profile)
}

// ExampleHashCache demonstrates using Redis hashes for caching
func ExampleHashCache() {
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	cache := NewCache(redisClient, nil)

	// Store user attributes as hash
	cache.HashSet(ctx, "user:123", "name", "John")
	cache.HashSet(ctx, "user:123", "email", "john@example.com")
	cache.HashSet(ctx, "user:123", "verified", "true")

	// Get specific field
	email, _ := cache.HashGet(ctx, "user:123", "email")
	log.Printf("Email: %s", email)

	// Get all fields
	all, _ := cache.HashGetAll(ctx, "user:123")
	log.Printf("All fields: %+v", all)

	// Increment counter
	count, _ := cache.HashIncrBy(ctx, "user:123", "login_count", 1)
	log.Printf("Login count: %d", count)
}

// ExampleJSONMarshal demonstrates JSON marshaling for complex types
func ExampleJSONMarshal() {
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	type Order struct {
		ID        string    `json:"id"`
		Items     []string  `json:"items"`
		Total     float64   `json:"total"`
		CreatedAt time.Time `json:"created_at"`
	}

	order := Order{
		ID:        "order-123",
		Items:     []string{"item1", "item2", "item3"},
		Total:     99.99,
		CreatedAt: time.Now(),
	}

	// Marshal to JSON
	data, err := json.Marshal(order)
	if err != nil {
		log.Printf("Marshal error: %v", err)
		return
	}

	// Store directly as string
	redisClient.Set(ctx, "order:123", data, time.Hour)

	// Retrieve and unmarshal
	val, err := redisClient.Get(ctx, "order:123").Result()
	if err != nil {
		log.Printf("Get error: %v", err)
		return
	}

	var retrieved Order
	json.Unmarshal([]byte(val), &retrieved)
	fmt.Printf("Retrieved order: %+v", retrieved)
}
