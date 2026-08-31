package kafka

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// TopicConfig represents Kafka topic configuration
type TopicConfig struct {
	Name              string `mapstructure:"name"`
	NumPartitions     int    `mapstructure:"num_partitions"`
	ReplicationFactor int    `mapstructure:"replication_factor"`
	MinInsyncReplicas int    `mapstructure:"min_insync_replicas"`
	RetentionMs       int64  `mapstructure:"retention_ms"`
	RetentionBytes    int64  `mapstructure:"retention_bytes"`
	CleanupPolicy     string `mapstructure:"cleanup_policy"` // "delete" or "compact"
	SegmentMs         int64  `mapstructure:"segment_ms"`
	SegmentBytes      int    `mapstructure:"segment_bytes"`
	CompressionType   string `mapstructure:"compression_type"` // "snappy", "gzip", "lz4", "zstd"
}

// ConsumerGroupConfig represents a consumer group configuration
type ConsumerGroupConfig struct {
	GroupID     string `mapstructure:"group_id"`
	Topic       string `mapstructure:"topic"`
	MinBytes    int    `mapstructure:"min_bytes"`
	MaxBytes    int    `mapstructure:"max_bytes"`
	MaxWaitMs   int    `mapstructure:"max_wait_ms"`
	StartOffset int    `mapstructure:"start_offset"` // 0 = earliest, 1 = latest
}

// KafkaConfig represents Kafka client configuration
type KafkaConfig struct {
	Brokers            []string              `mapstructure:"brokers"` // Multiple brokers for HA
	ConnectionTimeout  time.Duration         `mapstructure:"connection_timeout"`
	ReadTimeout        time.Duration         `mapstructure:"read_timeout"`
	WriteTimeout       time.Duration         `mapstructure:"write_timeout"`
	HeartbeatInterval  time.Duration         `mapstructure:"heartbeat_interval"`
	SessionTimeout     time.Duration         `mapstructure:"session_timeout"`
	RebalanceTimeout   time.Duration         `mapstructure:"rebalance_timeout"`
	MaxWaitTime        time.Duration         `mapstructure:"max_wait_time"`
	MinBytes           int                   `mapstructure:"min_bytes"` // Minimum bytes to fetch
	MaxBytes           int                   `mapstructure:"max_bytes"` // Maximum bytes to fetch per request
	QueueBufferingTime time.Duration         `mapstructure:"queue_buffering_time"`
	QueueBufferingMsgs int                   `mapstructure:"queue_buffering_msgs"`
	BatchSize          int                   `mapstructure:"batch_size"`
	LingerMs           int                   `mapstructure:"linger_ms"`
	CompressionType    string                `mapstructure:"compression_type"`
	RequiredAcks       int                   `mapstructure:"required_acks"` // -1 = all, 0 = none, 1 = leader
	Async              bool                  `mapstructure:"async"`         // Async produce
	Topics             []TopicConfig         `mapstructure:"topics"`
	ConsumerGroups     []ConsumerGroupConfig `mapstructure:"consumer_groups"`
}

// Kafka represents a Kafka client with connection management
type Kafka struct {
	readers map[string]*kafka.Reader
	writers map[string]*kafka.Writer
	conn    *kafka.Conn
	config  *KafkaConfig
	done    chan bool
}

// NewKafka creates a new Kafka client with retry on startup + auto-reconnect
func NewKafka(config *KafkaConfig) (*Kafka, error) {
	// Apply defaults
	applyDefaults(config)

	k := &Kafka{
		readers: make(map[string]*kafka.Reader),
		writers: make(map[string]*kafka.Writer),
		config:  config,
		done:    make(chan bool),
	}

	// Initial connection with retry
	maxRetries := 10
	for i := 1; i <= maxRetries; i++ {
		log.Printf("Connecting to Kafka brokers: %v (attempt %d/%d)", config.Brokers, i, maxRetries)
		if err := k.connect(); err != nil {
			if i == maxRetries {
				return nil, fmt.Errorf("failed to connect to Kafka after %d attempts: %w", maxRetries, err)
			}
			log.Printf("Connection attempt %d failed: %v, retrying in %v...", i, err, config.ConnectionTimeout)
			time.Sleep(config.ConnectionTimeout)
			continue
		}
		break
	}

	// Setup topics
	if err := k.setupTopics(); err != nil {
		return nil, err
	}

	go k.handleReconnect()

	log.Printf("Kafka client initialized successfully with brokers: %v", config.Brokers)
	return k, nil
}

// applyDefaults sets default values for Kafka config
func applyDefaults(config *KafkaConfig) {
	if config.ConnectionTimeout == 0 {
		config.ConnectionTimeout = 10 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 30 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30 * time.Second
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 3 * time.Second
	}
	if config.SessionTimeout == 0 {
		config.SessionTimeout = 45 * time.Second
	}
	if config.RebalanceTimeout == 0 {
		config.RebalanceTimeout = 60 * time.Second
	}
	if config.MaxWaitTime == 0 {
		config.MaxWaitTime = 500 * time.Millisecond
	}
	if config.MinBytes == 0 {
		config.MinBytes = 1
	}
	if config.MaxBytes == 0 {
		config.MaxBytes = 10 * 1024 * 1024 // 10MB
	}
	if config.RequiredAcks == 0 {
		config.RequiredAcks = -1 // Wait for all replicas
	}
	if config.CompressionType == "" {
		config.CompressionType = "snappy"
	}
}

// connect establishes connection to the first broker
func (k *Kafka) connect() error {
	log.Printf("Connecting to Kafka at %v...", k.config.Brokers)

	// Connect to the first broker
	conn, err := kafka.Dial("tcp", k.config.Brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka at %s: %w", k.config.Brokers[0], err)
	}

	k.conn = conn
	log.Printf("Connected to Kafka successfully")
	return nil
}

// setupTopics creates topics if they don't exist
func (k *Kafka) setupTopics() error {
	if len(k.config.Topics) == 0 {
		log.Println("No topics configured, skipping topic setup")
		return nil
	}

	log.Println("Setting up Kafka topics...")

	for _, topic := range k.config.Topics {
		err := k.createTopic(&topic)
		if err != nil {
			log.Printf("Warning: failed to create topic %s: %v", topic.Name, err)
		} else {
			log.Printf("Topic declared/verified: %s (partitions: %d, replication: %d)",
				topic.Name, topic.NumPartitions, topic.ReplicationFactor)
		}
	}

	log.Println("Kafka topic setup complete")
	return nil
}

// createTopic creates a topic with the given configuration
func (k *Kafka) createTopic(config *TopicConfig) error {
	if k.conn == nil {
		return fmt.Errorf("not connected to Kafka")
	}

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             config.Name,
			NumPartitions:     config.NumPartitions,
			ReplicationFactor: config.ReplicationFactor,
		},
	}

	err := k.conn.CreateTopics(topicConfigs...)
	if err != nil {
		// Topic might already exist, which is okay
		log.Printf("CreateTopics result for %s: %v", config.Name, err)
	}

	return nil
}

// handleReconnect handles automatic reconnection
func (k *Kafka) handleReconnect() {
	for {
		select {
		case <-k.done:
			return
		case <-time.After(10 * time.Second):
			// Check connection health periodically
			if k.conn == nil {
				log.Println("Kafka connection lost, reconnecting...")

				for {
					time.Sleep(k.config.ConnectionTimeout)

					if err := k.connect(); err != nil {
						log.Printf("Kafka reconnect failed: %v, retrying...", err)
						continue
					}

					// Recreate writers
					k.recreateWriters()

					log.Println("Reconnected to Kafka")
					break
				}
			}
		}
	}
}

// recreateWriters recreates all writers after reconnection
func (k *Kafka) recreateWriters() {
	for topic, writer := range k.writers {
		// Close old writer
		writer.Close()

		// Create new writer
		k.writers[topic] = k.newWriter(topic)
		log.Printf("Writer recreated for topic: %s", topic)
	}
}

// newWriter creates a new Kafka writer for a topic
func (k *Kafka) newWriter(topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(k.config.Brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    k.config.BatchSize,
		BatchTimeout: time.Duration(k.config.LingerMs) * time.Millisecond,
		WriteTimeout: k.config.WriteTimeout,
		RequiredAcks: kafka.RequiredAcks(k.config.RequiredAcks),
		Async:        k.config.Async,
		Compression:  getCompression(k.config.CompressionType),
	}
}

// GetWriter returns a writer for the specified topic
func (k *Kafka) GetWriter(topic string) *kafka.Writer {
	if writer, exists := k.writers[topic]; exists {
		return writer
	}

	// Create new writer for topic
	writer := k.newWriter(topic)
	k.writers[topic] = writer
	return writer
}

// GetReader returns a reader for the specified topic and consumer group
func (k *Kafka) GetReader(topic string, groupID string) *kafka.Reader {
	readerKey := fmt.Sprintf("%s-%s", topic, groupID)
	if reader, exists := k.readers[readerKey]; exists {
		return reader
	}

	// Determine start offset
	startOffset := kafka.LastOffset
	for _, cg := range k.config.ConsumerGroups {
		if cg.Topic == topic && cg.GroupID == groupID {
			if cg.StartOffset == 0 {
				startOffset = kafka.FirstOffset
			}
			break
		}
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        k.config.Brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       k.config.MinBytes,
		MaxBytes:       k.config.MaxBytes,
		MaxWait:        k.config.MaxWaitTime,
		StartOffset:    startOffset,
		CommitInterval: time.Second,
	})

	k.readers[readerKey] = reader
	return reader
}

// Publish sends a message to a topic
func (k *Kafka) Publish(ctx context.Context, topic string, key []byte, value []byte) error {
	writer := k.GetWriter(topic)

	msg := kafka.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	}

	err := writer.WriteMessages(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to publish message to %s: %w", topic, err)
	}

	return nil
}

// PublishBatch sends multiple messages to a topic
func (k *Kafka) PublishBatch(ctx context.Context, topic string, messages []kafka.Message) error {
	writer := k.GetWriter(topic)

	err := writer.WriteMessages(ctx, messages...)
	if err != nil {
		return fmt.Errorf("failed to publish batch to %s: %w", topic, err)
	}

	return nil
}

// Consume reads messages from a topic using consumer group
func (k *Kafka) Consume(ctx context.Context, topic string, groupID string, handler func(msg kafka.Message) error) error {
	reader := k.GetReader(topic, groupID)
	defer reader.Close()

	log.Printf("Starting consumer for topic: %s, group: %s", topic, groupID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Consumer stopped for topic: %s", topic)
			return ctx.Err()
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				log.Printf("Error reading message from %s: %v", topic, err)
				continue
			}

			if err := handler(msg); err != nil {
				log.Printf("Error handling message from %s: %v", topic, err)
				// Decide: should we continue or stop?
				// For now, continue processing
			}
		}
	}
}

// ConsumeWithBatch reads messages in batches for higher throughput
func (k *Kafka) ConsumeWithBatch(ctx context.Context, topic string, groupID string, batchSize int, batchTimeout time.Duration, handler func(messages []kafka.Message) error) error {
	reader := k.GetReader(topic, groupID)
	defer reader.Close()

	log.Printf("Starting batch consumer for topic: %s, group: %s, batchSize: %d", topic, groupID, batchSize)

	batch := make([]kafka.Message, 0, batchSize)
	ticker := time.NewTicker(batchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Process remaining batch before exiting
			if len(batch) > 0 {
				handler(batch)
			}
			log.Printf("Batch consumer stopped for topic: %s", topic)
			return ctx.Err()
		case <-ticker.C:
			// Timeout - process current batch
			if len(batch) > 0 {
				if err := handler(batch); err != nil {
					log.Printf("Error handling batch from %s: %v", topic, err)
				}
				batch = batch[:0]
			}
		default:
			// Try to read a message with a short timeout
			readCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			msg, err := reader.ReadMessage(readCtx)
			cancel()

			if err != nil {
				if err != context.DeadlineExceeded {
					log.Printf("Error reading message from %s: %v", topic, err)
				}
				continue
			}

			batch = append(batch, msg)

			if len(batch) >= batchSize {
				if err := handler(batch); err != nil {
					log.Printf("Error handling batch from %s: %v", topic, err)
				}
				batch = batch[:0]
			}
		}
	}
}

// Close closes all connections and cleans up
func (k *Kafka) Close() error {
	log.Println("Closing Kafka connections...")

	// Stop reconnect goroutine
	close(k.done)

	// Close all readers
	for key, reader := range k.readers {
		if err := reader.Close(); err != nil {
			log.Printf("Error closing reader %s: %v", key, err)
		}
	}

	// Close all writers
	for topic, writer := range k.writers {
		if err := writer.Close(); err != nil {
			log.Printf("Error closing writer for %s: %v", topic, err)
		}
	}

	// Close main connection
	if k.conn != nil {
		if err := k.conn.Close(); err != nil {
			log.Printf("Error closing Kafka connection: %v", err)
		}
	}

	log.Println("Kafka connections closed")
	return nil
}

// HealthCheck checks if the connection is alive
func (k *Kafka) HealthCheck() error {
	if k.conn == nil {
		return fmt.Errorf("Kafka connection is nil")
	}

	// Try to get metadata to verify connection
	_, err := k.conn.Brokers()
	if err != nil {
		return fmt.Errorf("Kafka connection health check failed: %w", err)
	}

	return nil
}

// GetConnection returns the raw Kafka connection
func (k *Kafka) GetConnection() *kafka.Conn {
	return k.conn
}

// CreateTopic creates a new topic (can be called dynamically)
func (k *Kafka) CreateTopic(name string, partitions int, replicationFactor int) error {
	config := &TopicConfig{
		Name:              name,
		NumPartitions:     partitions,
		ReplicationFactor: replicationFactor,
	}
	return k.createTopic(config)
}

// ListBrokers returns the list of broker addresses
func (k *Kafka) ListBrokers() []string {
	return k.config.Brokers
}

// getCompression converts compression type string to kafka.Compression
func getCompression(compressionType string) kafka.Compression {
	switch compressionType {
	case "snappy":
		return kafka.Snappy
	case "gzip":
		return kafka.Gzip
	case "lz4":
		return kafka.Lz4
	case "zstd":
		return kafka.Zstd
	default:
		return kafka.Snappy
	}
}
