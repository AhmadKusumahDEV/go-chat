package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

// EmailEventWorker processes email events from Kafka
type EmailEventWorker struct {
	reader  *kafka.Reader
	handler EmailEventHandler
}

// EmailEventHandler interface for processing email events
type EmailEventHandler interface {
	HandleOTPEmail(ctx context.Context, to string, otp string) error
	HandlePaymentSuccessEmail(ctx context.Context, to string, username string, orderID string, amount int64) error
}

// EmailEvent represents an email event (same structure as queue.EmailEvent)
type EmailEvent struct {
	Type    string `json:"type"` // "otp", "payment_success"
	To      string `json:"to"`
	Subject string `json:"subject"`
	OTP     string `json:"otp,omitempty"`
	// Payment fields
	Username string `json:"username,omitempty"`
	PlanName string `json:"plan_name,omitempty"`
	OrderID  string `json:"order_id,omitempty"`
	Amount   int64  `json:"amount,omitempty"`
}

// NewEmailEventWorker creates a new email event worker
func NewEmailEventWorker(brokers []string, topic string, groupID string, handler EmailEventHandler) *EmailEventWorker {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10 * 1024 * 1024, // 10MB
	})

	return &EmailEventWorker{
		reader:  reader,
		handler: handler,
	}
}

// Start begins consuming messages from Kafka
func (w *EmailEventWorker) Start(ctx context.Context) error {
	log.Printf("Starting Kafka EmailEventWorker for topic: %s, group: %s", w.reader.Config().Topic, w.reader.Config().GroupID)

	go w.consume(ctx)
	return nil
}

func (w *EmailEventWorker) consume(ctx context.Context) {
	defer w.reader.Close()

	for {
		select {
		case <-ctx.Done():
			log.Println("EmailEventWorker shutting down...")
			return
		default:
			msg, err := w.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("Error reading message: %v", err)
				continue
			}

			w.processMessage(ctx, msg)
		}
	}
}

func (w *EmailEventWorker) processMessage(ctx context.Context, msg kafka.Message) {
	log.Printf("Processing email event: %s", msg.Key)

	var event EmailEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.Printf("Failed to unmarshal email event: %v", err)
		return
	}

	log.Printf("Email type: %s, To: %s", event.Type, event.To)

	var err error
	switch event.Type {
	case "otp":
		err = w.handler.HandleOTPEmail(ctx, event.To, event.OTP)
	case "payment_success":
		err = w.handler.HandlePaymentSuccessEmail(ctx, event.To, event.Username, event.OrderID, event.Amount)
	default:
		log.Printf("Unknown email event type: %s", event.Type)
		return
	}

	if err != nil {
		log.Printf("Failed to process email event %s: %v", event.Type, err)
		// Could implement retry logic here
		return
	}

	log.Printf("Successfully processed email event: %s", event.Type)
}

// Stop gracefully stops the worker
func (w *EmailEventWorker) Stop() error {
	return w.reader.Close()
}
