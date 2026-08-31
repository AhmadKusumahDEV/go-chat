package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	config  *RabbitMQConfig
	done    chan bool
}

type WorkerFunc func(ctx context.Context, event *amqp091.Delivery)

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
		log.Printf("Connecting to RabbitMQ... (attempt %d/%d)", i, maxRetries)
		if err := rmq.connect(); err != nil {
			if i == maxRetries {
				return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
			}
			log.Printf("Connection attempt %d failed: %v, retrying in %v...", i, err, config.ReconnectDelay)
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

	go rmq.handleReconnect()

	return rmq, nil
}

func (r *RabbitMQ) connect() error {
	log.Printf("Connecting to RabbitMQ at %s...", r.config.URL)

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

	log.Printf("Connected to RabbitMQ successfully")
	return nil
}

func (r *RabbitMQ) setupTopology() error {
	log.Println("Setting up RabbitMQ topology...")

	for j := range r.config.Exchange {
		var args amqp091.Table

		if r.config.Exchange[j].Arguments != nil {
			args = *r.config.Exchange[j].Arguments
		}

		err := r.channel.ExchangeDeclare(
			r.config.Exchange[j].Name,
			r.config.Exchange[j].Kind,
			r.config.Exchange[j].Durable,
			r.config.Exchange[j].AutoDelete,
			r.config.Exchange[j].Internal,
			r.config.Exchange[j].NoWait,
			args,
		)
		if err != nil {
			return fmt.Errorf("failed to declare exchange %s: %w", r.config.Exchange[j].Name, err)
		}
		log.Printf("Exchange declared: %s (%s)", r.config.Exchange[j].Name, r.config.Exchange[j].Kind)
	}

	for q := range r.config.Queue {
		var args amqp091.Table

		if r.config.Queue[q].Arguments != nil {
			args = *r.config.Queue[q].Arguments
		}

		_, err := r.channel.QueueDeclare(
			r.config.Queue[q].Name,
			r.config.Queue[q].Durable,
			r.config.Queue[q].AutoDelete,
			r.config.Queue[q].Exclusive,
			r.config.Queue[q].NoWait,
			args,
		)

		if err != nil {
			return fmt.Errorf("failed to declare DLQ: %w", err)
		}

		for b := range r.config.Queue[q].Bindings {
			var bindArgs amqp091.Table

			if r.config.Queue[q].Bindings[b].Arguments != nil {
				bindArgs = *r.config.Queue[q].Bindings[b].Arguments
			}

			err = r.channel.QueueBind(
				r.config.Queue[q].Name,
				r.config.Queue[q].Bindings[b].Key,
				r.config.Queue[q].Bindings[b].Exchange,
				r.config.Queue[q].Bindings[b].NoWait,
				bindArgs,
			)
			if err != nil {
				return fmt.Errorf("failed to bind queue %s: %w", r.config.Queue[q].Name, err)
			}
		}
	}

	log.Println("RabbitMQ topology setup complete")
	return nil
}

// handleReconnect - Auto-reconnect jika connection lost
func (r *RabbitMQ) handleReconnect() {
	for {
		select {
		case <-r.done:
			return
		case <-r.conn.NotifyClose(make(chan *amqp091.Error)):
			log.Println("RabbitMQ connection lost, reconnecting...")

			for {
				time.Sleep(r.config.ReconnectDelay)

				if err := r.connect(); err != nil {
					log.Printf("Reconnect failed: %v, retrying...", err)
					continue
				}

				if err := r.setupTopology(); err != nil {
					log.Printf("Topology setup failed: %v, retrying...", err)
					continue
				}

				log.Println("Reconnected to RabbitMQ")
				break
			}
		}
	}
}

func (r *RabbitMQ) GetChannel() *amqp091.Channel {
	return r.channel
}

// GetConnection - Get connection
func (r *RabbitMQ) GetConnection() *amqp091.Connection {
	return r.conn
}

func (r *RabbitMQ) Close() error {
	log.Println("🔌 Closing RabbitMQ connection...")

	close(r.done)

	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			log.Printf("Error closing channel: %v", err)
		}
	}

	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}

	log.Println("RabbitMQ connection closed")
	return nil
}

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

func (r *RabbitMQ) DedicatedWorkerPool()
