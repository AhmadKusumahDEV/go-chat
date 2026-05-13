// internal/worker/notification_worker.go
package worker

import (
	"context"
	"encoding/json"
	"log"

	"firebase.google.com/go/v4/messaging"
	"github.com/AhmadKusumahDEV/go-chat/internal/queue"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/rabbitmq/amqp091-go"
)

type NotificationWorker struct {
	rabbitmq    *amqp091.Channel
	fcmClient   *messaging.Client
	userRepo    repository.RepositoryUser
	messageRepo repository.MessageRepository
}

func NewNotificationWorker(
	rabbitmq *amqp091.Channel,
	fcmClient *messaging.Client,
	userRepo repository.RepositoryUser,
	messageRepo repository.MessageRepository,
) *NotificationWorker {
	return &NotificationWorker{
		rabbitmq:    rabbitmq,
		fcmClient:   fcmClient,
		userRepo:    userRepo,
		messageRepo: messageRepo,
	}
}

// Start - Running terus menerus
func (w *NotificationWorker) Start(ctx context.Context) error {
	// 1. Declare queue
	q, err := w.rabbitmq.QueueDeclare(
		"push-notifications",
		true,
		false,
		false,
		false,
		amqp091.Table{
			"x-dead-letter-exchange": "chat.dlx",
		},
	)
	if err != nil {
		return err
	}

	// 2. Bind to exchange
	err = w.rabbitmq.QueueBind(
		q.Name,
		"notification.*", // routing key pattern
		"chat.notifications",
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// 3. Set QoS
	err = w.rabbitmq.Qos(5, 0, false)
	if err != nil {
		return err
	}

	// 4. Start consuming
	msgs, err := w.rabbitmq.Consume(
		q.Name,
		"notification-worker",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Println("✅ NotificationWorker started, waiting for messages...")

	// 5. Process forever
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("NotificationWorker shutting down...")
				return

			case msg, ok := <-msgs:
				if !ok {
					return
				}
				w.processNotification(&msg)
			}
		}
	}()

	return nil
}

func (w *NotificationWorker) processNotification(msg *amqp091.Delivery) {
	log.Printf("📥 Processing notification: %s", msg.MessageId)

	// 1. Deserialize
	var event queue.NotificationEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("❌ Failed to unmarshal: %v", err)
		msg.Nack(false, false)
		return
	}

	// 4. ACK
	msg.Ack(false)
}
