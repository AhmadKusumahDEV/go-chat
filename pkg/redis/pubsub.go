package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// PubSubMessage represents a message published via pub/sub
type PubSubMessage struct {
	Channel string
	Payload interface{}
}

// MessageHandler is a function that handles pub/sub messages
type MessageHandler func(msg *PubSubMessage)

// PubSub provides pub/sub functionality using Redis
type PubSub struct {
	client *redis.Client
}

// NewPubSub creates a new PubSub instance
func NewPubSub(client *redis.Client) *PubSub {
	return &PubSub{
		client: client,
	}
}

// Publish publishes a message to a channel
func (p *PubSub) Publish(ctx context.Context, channel string, payload interface{}) error {
	var data []byte
	var err error

	switch v := payload.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	if err := p.client.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	return nil
}

// Subscribe subscribes to one or more channels
func (p *PubSub) Subscribe(ctx context.Context, handler MessageHandler, channels ...string) error {
	if len(channels) == 0 {
		return fmt.Errorf("at least one channel is required")
	}

	pubsub := p.client.Subscribe(ctx, channels...)

	// Wait for confirmation of subscription
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	log.Printf("Subscribed to channels: %v", channels)

	// Start goroutine to handle messages
	go func() {
		defer pubsub.Close()

		ch := pubsub.Channel()

		for {
			select {
			case <-ctx.Done():
				log.Printf("PubSub subscription cancelled for channels: %v", channels)
				return
			case msg, ok := <-ch:
				if !ok {
					log.Printf("PubSub channel closed for channels: %v", channels)
					return
				}

				handler(&PubSubMessage{
					Channel: msg.Channel,
					Payload: msg.Payload,
				})
			}
		}
	}()

	return nil
}

// SubscribeWithPattern subscribes to channels matching a pattern
func (p *PubSub) SubscribeWithPattern(ctx context.Context, handler MessageHandler, pattern string) error {
	pubsub := p.client.PSubscribe(ctx, pattern)

	// Wait for confirmation
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to psubscribe: %w", err)
	}

	log.Printf("Subscribed to pattern: %s", pattern)

	// Start goroutine to handle messages
	go func() {
		defer pubsub.Close()

		ch := pubsub.Channel()

		for {
			select {
			case <-ctx.Done():
				log.Printf("PubSub pattern subscription cancelled: %s", pattern)
				return
			case msg, ok := <-ch:
				if !ok {
					log.Printf("PubSub channel closed for pattern: %s", pattern)
					return
				}

				handler(&PubSubMessage{
					Channel: msg.Channel,
					Payload: msg.Payload,
				})
			}
		}
	}()

	return nil
}

// PubSubManager manages multiple pub/sub subscriptions
type PubSubManager struct {
	client       *redis.Client
	subscriptions map[string]*redis.PubSub
	mu           sync.RWMutex
}

// NewPubSubManager creates a new PubSubManager
func NewPubSubManager(client *redis.Client) *PubSubManager {
	return &PubSubManager{
		client:       client,
		subscriptions: make(map[string]*redis.PubSub),
	}
}

// Subscribe adds a new subscription
func (m *PubSubManager) Subscribe(ctx context.Context, id string, handler MessageHandler, channels ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.subscriptions[id]; exists {
		return fmt.Errorf("subscription %s already exists", id)
	}

	pubsub := m.client.Subscribe(ctx, channels...)
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	m.subscriptions[id] = pubsub

	// Start handling in background
	go m.handleMessages(id, pubsub, handler)

	log.Printf("Created subscription %s for channels: %v", id, channels)
	return nil
}

// SubscribePattern adds a pattern subscription
func (m *PubSubManager) SubscribePattern(ctx context.Context, id string, handler MessageHandler, pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.subscriptions[id]; exists {
		return fmt.Errorf("subscription %s already exists", id)
	}

	pubsub := m.client.PSubscribe(ctx, pattern)
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to psubscribe: %w", err)
	}

	m.subscriptions[id] = pubsub

	// Start handling in background
	go m.handleMessages(id, pubsub, handler)

	log.Printf("Created pattern subscription %s for: %s", id, pattern)
	return nil
}

// Unsubscribe removes a subscription
func (m *PubSubManager) Unsubscribe(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pubsub, exists := m.subscriptions[id]
	if !exists {
		return fmt.Errorf("subscription %s not found", id)
	}

	if err := pubsub.Close(); err != nil {
		return fmt.Errorf("failed to close subscription: %w", err)
	}

	delete(m.subscriptions, id)
	log.Printf("Removed subscription: %s", id)
	return nil
}

// handleMessages handles incoming messages for a subscription
func (m *PubSubManager) handleMessages(id string, pubsub *redis.PubSub, handler MessageHandler) {
	ch := pubsub.Channel()

	for msg := range ch {
		handler(&PubSubMessage{
			Channel: msg.Channel,
			Payload: msg.Payload,
		})
	}
}

// Publish publishes a message to a channel
func (m *PubSubManager) Publish(ctx context.Context, channel string, payload interface{}) error {
	var data []byte
	var err error

	switch v := payload.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	if err := m.client.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	return nil
}

// Close closes all subscriptions
func (m *PubSubManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, pubsub := range m.subscriptions {
		if err := pubsub.Close(); err != nil {
			log.Printf("Failed to close subscription %s: %v", id, err)
		}
	}

	m.subscriptions = make(map[string]*redis.PubSub)
	return nil
}

// StreamMessage represents a stream message
type StreamMessage struct {
	ID     string
	Stream string
	Fields map[string]interface{}
}

// Stream provides Redis Stream functionality
type Stream struct {
	client *redis.Client
}

// NewStream creates a new Stream instance
func NewStream(client *redis.Client) *Stream {
	return &Stream{client: client}
}

// Add adds a message to a stream
func (s *Stream) Add(ctx context.Context, stream string, fields map[string]interface{}) (string, error) {
	values := make(map[string]interface{})
	for k, v := range fields {
		switch val := v.(type) {
		case string:
			values[k] = val
		case []byte:
			values[k] = string(val)
		default:
			data, err := json.Marshal(v)
			if err != nil {
				return "", fmt.Errorf("failed to marshal field %s: %w", k, err)
			}
			values[k] = string(data)
		}
	}

	id, err := s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()

	if err != nil {
		return "", fmt.Errorf("failed to add to stream: %w", err)
	}

	return id, nil
}

// Read reads messages from a stream
func (s *Stream) Read(ctx context.Context, stream string, group string, consumer string, count int64) ([]StreamMessage, error) {
	if count == 0 {
		count = 10
	}

	streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    count,
		Block:    time.Second,
	}).Result()

	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to read from stream: %w", err)
	}

	var messages []StreamMessage
	for _, streamResult := range streams {
		for _, msg := range streamResult.Messages {
			fields := make(map[string]interface{})
			for k, v := range msg.Values {
				if str, ok := v.(string); ok {
					fields[k] = str
				} else {
					fields[k] = v
				}
			}
			messages = append(messages, StreamMessage{
				ID:     msg.ID,
				Stream: streamResult.Stream,
				Fields: fields,
			})
		}
	}

	return messages, nil
}

// CreateGroup creates a consumer group for a stream
func (s *Stream) CreateGroup(ctx context.Context, stream string, group string) error {
	err := s.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create group: %w", err)
	}
	return nil
}

// Ack acknowledges processed messages
func (s *Stream) Ack(ctx context.Context, stream string, group string, ids ...string) (int64, error) {
	count, err := s.client.XAck(ctx, stream, group, ids...).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to ack: %w", err)
	}
	return count, nil
}

// Pending returns pending messages in a group
func (s *Stream) Pending(ctx context.Context, stream string, group string) ([]redis.XPendingExt, error) {
	pending, err := s.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: stream,
		Group:  group,
		Start:  "-",
		End:    "+",
		Count:  100,
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to get pending: %w", err)
	}

	return pending, nil
}

// Claim claims pending messages for a consumer
func (s *Stream) Claim(ctx context.Context, stream string, group string, consumer string, minIdle time.Duration, ids ...string) ([]StreamMessage, error) {
	messages, err := s.client.XClaim(ctx, &redis.XClaimArgs{
		Stream:   stream,
		Group:    group,
		Consumer: consumer,
		MinIdle:  minIdle,
		Messages: ids,
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to claim: %w", err)
	}

	var result []StreamMessage
	for _, msg := range messages {
		fields := make(map[string]interface{})
		for k, v := range msg.Values {
			if str, ok := v.(string); ok {
				fields[k] = str
			} else {
				fields[k] = v
			}
		}
		result = append(result, StreamMessage{
			ID:     msg.ID,
			Stream: stream,
			Fields: fields,
		})
	}

	return result, nil
}
