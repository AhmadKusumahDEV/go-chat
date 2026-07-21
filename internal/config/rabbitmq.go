// internal/config/rabbitmq.go
package config

import (
	"fmt"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type Exchange struct {
	Name       string `mapstructure:"name"`
	Kind       string `mapstructure:"kind"`
	Durable    bool   `mapstructure:"durable"`
	AutoDelete bool   `mapstructure:"auto_delete"`
	Internal   bool   `mapstructure:"internal"`
	NoWait     bool   `mapstructure:"no_wait"`
	Arguments  any    `mapstructure:"arguments"`
}

type Binding struct {
	Exchange string `mapstructure:"exchange"`
	Key      string `mapstructure:"key"`
	Routing  string `mapstructure:"routing"`
}

type Queue struct {
	Name       string `mapstructure:"name"`
	Durable    bool   `mapstructure:"durable"`
	AutoDelete bool   `mapstructure:"auto_delete"`
	Exclusive  bool   `mapstructure:"exclusive"`
	NoWait     bool   `mapstructure:"no_wait"`
	Arguments  any    `mapstructure:"arguments"`
}

type RabbitMQConfig struct {
	URL            string        `mapstructure:"url"`
	MaxChannels    int           `mapstructure:"max_channels"`
	ReconnectDelay time.Duration `mapstructure:"reconnect_delay"`
	PrefetchCount  int           `mapstructure:"prefetch_count"`
	PrefetchSize   int           `mapstructure:"prefetch_size"`
	Heartbeat      time.Duration `mapstructure:"heartbeat"`
	ConnectionName string        `mapstructure:"connection_name"`
	Exchange       []Exchange    `mapstructure:"exchange"`
}

type RabbitMQ struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	config  *RabbitMQConfig
	done    chan bool
}

// NewRabbitMQ - Create RabbitMQ connection dengan retry on startup + auto-reconnect
func NewRabbitMQ(config *RabbitMQConfig) (*RabbitMQ, error) {
	if config.MaxChannels == 0 {
		config.MaxChannels = 100
	}
	if config.ReconnectDelay == 0 {
		config.ReconnectDelay = 5 * time.Second
	}
	if config.Heartbeat == 0 {
		config.Heartbeat = 10 * time.Second
	}
	if config.ConnectionName == "" {
		config.ConnectionName = "chat-app"
	}

	rmq := &RabbitMQ{
		config: config,
		done:   make(chan bool),
	}

	// Initial connection with retry
	maxRetries := 10
	for i := 1; i <= maxRetries; i++ {
		log.Printf("🔄 Connecting to RabbitMQ... (attempt %d/%d)", i, maxRetries)
		if err := rmq.connect(); err != nil {
			if i == maxRetries {
				return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
			}
			log.Printf("⚠️ Connection attempt %d failed: %v, retrying in %v...", i, err, config.ReconnectDelay)
			time.Sleep(config.ReconnectDelay)
			continue
		}
		// Success - break out of retry loop
		break
	}

	// Setup exchanges dan queues
	if err := rmq.setupTopology(); err != nil {
		return nil, err
	}

	// Start auto-reconnect goroutine
	go rmq.handleReconnect()

	return rmq, nil
}

func (r *RabbitMQ) connect() error {
	log.Printf("🔄 Connecting to RabbitMQ at %s...", r.config.URL)

	// Create connection with custom config
	config := amqp091.Config{
		Heartbeat: r.config.Heartbeat,
		Locale:    "en_US",
		Properties: amqp091.Table{
			"connection_name": r.config.ConnectionName,
		},
	}

	conn, err := amqp091.DialConfig(r.config.URL, config)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ at %s: %w", r.config.URL, err)
	}

	// Create channel
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS globally (bisa di-override per consumer)
	err = ch.Qos(
		r.config.PrefetchCount, // prefetch count
		r.config.PrefetchSize,  // prefetch size
		false,                  // global
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	r.conn = conn
	r.channel = ch

	log.Printf("✅ Connected to RabbitMQ successfully")
	return nil
}

func (r *RabbitMQ) setupTopology() error {
	log.Println("🔧 Setting up RabbitMQ topology...")

	// 1. Declare exchanges
	exchanges := []struct {
		name       string
		kind       string
		durable    bool
		autoDelete bool
	}{
		{"chat.messages", "fanout", true, false},
		{"chat.notifications", "topic", true, false},
		{"chat.dlx", "fanout", true, false}, // Dead Letter Exchange
	}

	for _, ex := range exchanges {
		err := r.channel.ExchangeDeclare(
			ex.name,
			ex.kind,
			ex.durable,
			ex.autoDelete,
			false, // internal
			false, // no-wait
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to declare exchange %s: %w", ex.name, err)
		}
		log.Printf("✅ Exchange declared: %s (%s)", ex.name, ex.kind)
	}

	// 2. Declare Dead Letter Queue
	_, err := r.channel.QueueDeclare(
		"chat.dlq", // queue name
		true,       // durable
		false,      // delete when unused
		false,      // exclusive
		false,      // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Bind DLQ to DLX
	err = r.channel.QueueBind(
		"chat.dlq",
		"",
		"chat.dlx",
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind DLQ: %w", err)
	}

	log.Println("✅ RabbitMQ topology setup complete")
	return nil
}

// handleReconnect - Auto-reconnect jika connection lost
func (r *RabbitMQ) handleReconnect() {
	for {
		select {
		case <-r.done:
			return
		case <-r.conn.NotifyClose(make(chan *amqp091.Error)):
			log.Println("⚠️ RabbitMQ connection lost, reconnecting...")

			for {
				time.Sleep(r.config.ReconnectDelay)

				if err := r.connect(); err != nil {
					log.Printf("❌ Reconnect failed: %v, retrying...", err)
					continue
				}

				if err := r.setupTopology(); err != nil {
					log.Printf("❌ Topology setup failed: %v, retrying...", err)
					continue
				}

				log.Println("✅ Reconnected to RabbitMQ")
				break
			}
		}
	}
}

// GetChannel - Get channel for publishing/consuming
func (r *RabbitMQ) GetChannel() *amqp091.Channel {
	return r.channel
}

// GetConnection - Get connection
func (r *RabbitMQ) GetConnection() *amqp091.Connection {
	return r.conn
}

// Close - Close connection gracefully
func (r *RabbitMQ) Close() error {
	log.Println("🔌 Closing RabbitMQ connection...")

	close(r.done) // Stop reconnect goroutine

	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			log.Printf("⚠️ Error closing channel: %v", err)
		}
	}

	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			log.Printf("⚠️ Error closing connection: %v", err)
		}
	}

	log.Println("✅ RabbitMQ connection closed")
	return nil
}

// CreateChannel - Create new channel (untuk worker yang butuh dedicated channel)
func (r *RabbitMQ) CreateChannel() (*amqp091.Channel, error) {
	ch, err := r.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	// Set QoS
	err = ch.Qos(
		r.config.PrefetchCount,
		r.config.PrefetchSize,
		false,
	)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return ch, nil
}

// HealthCheck - Check if connection is alive
func (r *RabbitMQ) HealthCheck() error {
	if r.conn == nil || r.conn.IsClosed() {
		return fmt.Errorf("connection is closed")
	}

	if r.channel == nil {
		return fmt.Errorf("channel is nil")
	}

	return nil
}
