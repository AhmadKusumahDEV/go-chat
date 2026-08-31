package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// EmailHandler implements kafka.EmailEventHandler
type EmailHandler struct{}

func (h *EmailHandler) HandleOTPEmail(ctx context.Context, to string, otp string) error {
	log.Printf("Sending OTP email to %s: %s", to, otp)
	// Your email sending logic here
	return nil
}

func (h *EmailHandler) HandlePaymentSuccessEmail(ctx context.Context, to string, username string, orderID string, amount int64) error {
	log.Printf("Sending payment success email to %s for order %s (amount: %d)", to, orderID, amount)
	// Your email sending logic here
	return nil
}

// ExampleUsage demonstrates how to use the Kafka client
func ExampleUsage() {
	// 1. Create Kafka client
	config := &KafkaConfig{
		Brokers:           []string{"localhost:9092"},
		ConnectionTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		CompressionType:   "snappy",
		RequiredAcks:      -1,
		Topics: []TopicConfig{
			{
				Name:              "email-notifications",
				NumPartitions:     3,
				ReplicationFactor: 1,
				RetentionMs:       604800000, // 7 days
			},
		},
	}

	client, err := NewKafka(config)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	// 2. Publish a message
	ctx := context.Background()

	emailEvent := map[string]interface{}{
		"type":     "payment_success",
		"to":       "user@example.com",
		"subject":  "Payment Successful",
		"username": "John Doe",
		"order_id": "INV-2024-001",
		"amount":   99000,
	}

	eventJSON, _ := json.Marshal(emailEvent)

	err = client.Publish(ctx, "email-notifications", []byte("user@example.com"), eventJSON)
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
	} else {
		log.Println("Message published successfully")
	}

	// 3. Start consuming
	worker := NewEmailEventWorker(
		config.Brokers,
		"email-notifications",
		"email-worker-group",
		&EmailHandler{},
	)

	if err := worker.Start(ctx); err != nil {
		log.Printf("Failed to start worker: %v", err)
	}

	// Keep running
	select {}
}

// Example: Publishing with batch
func ExamplePublishBatch() {
	config := &KafkaConfig{
		Brokers:           []string{"localhost:9092"},
		ConnectionTimeout: 10 * time.Second,
	}

	client, _ := NewKafka(config)
	defer client.Close()

	ctx := context.Background()

	messages := []kafka.Message{
		{
			Key:   []byte("user1@example.com"),
			Value: []byte(`{"type":"otp","to":"user1@example.com","otp":"123456"}`),
		},
		{
			Key:   []byte("user2@example.com"),
			Value: []byte(`{"type":"otp","to":"user2@example.com","otp":"654321"}`),
		},
	}

	err := client.PublishBatch(ctx, "email-notifications", messages)
	if err != nil {
		log.Printf("Failed to publish batch: %v", err)
	} else {
		log.Println("Batch published successfully")
	}
}

// Example: Direct consume without worker pattern
func ExampleConsume() {
	config := &KafkaConfig{
		Brokers:           []string{"localhost:9092"},
		ConnectionTimeout: 10 * time.Second,
		MaxWaitTime:       500 * time.Millisecond,
	}

	client, _ := NewKafka(config)
	defer client.Close()

	ctx := context.Background()

	// Handler function
	handler := func(msg kafka.Message) error {
		log.Printf("Received: key=%s, value=%s", string(msg.Key), string(msg.Value))

		var event EmailEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return err
		}

		log.Printf("Processing email type: %s to: %s", event.Type, event.To)
		return nil
	}

	// Start consuming
	err := client.Consume(ctx, "email-notifications", "my-consumer-group", handler)
	if err != nil {
		log.Printf("Consumer error: %v", err)
	}
}

// Example: Batch consume for high throughput
func ExampleConsumeBatch() {
	config := &KafkaConfig{
		Brokers:           []string{"localhost:9092"},
		ConnectionTimeout: 10 * time.Second,
		BatchSize:         100,
	}

	client, _ := NewKafka(config)
	defer client.Close()

	ctx := context.Background()

	// Batch handler - process up to 100 messages at once
	batchHandler := func(messages []kafka.Message) error {
		log.Printf("Processing batch of %d messages", len(messages))

		for _, msg := range messages {
			var event EmailEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Failed to unmarshal: %v", err)
				continue
			}
			log.Printf("Processing: %s to %s", event.Type, event.To)
		}

		return nil
	}

	err := client.ConsumeWithBatch(ctx, "email-notifications", "batch-consumer-group", 100, 5*time.Second, batchHandler)
	if err != nil {
		log.Printf("Batch consumer error: %v", err)
	}
}

// Example: Health check
func ExampleHealthCheck() {
	config := &KafkaConfig{
		Brokers: []string{"localhost:9092"},
	}

	client, _ := NewKafka(config)
	defer client.Close()

	if err := client.HealthCheck(); err != nil {
		log.Printf("Kafka unhealthy: %v", err)
	} else {
		log.Println("Kafka is healthy")
	}
}

// Example: Create topic dynamically
func ExampleCreateTopic() {
	config := &KafkaConfig{
		Brokers: []string{"localhost:9092"},
	}

	client, _ := NewKafka(config)
	defer client.Close()

	err := client.CreateTopic("new-topic", 6, 1)
	if err != nil {
		log.Printf("Failed to create topic: %v", err)
	} else {
		fmt.Println("Topic created successfully")
	}
}
