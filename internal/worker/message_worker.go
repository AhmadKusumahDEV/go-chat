// internal/worker/message_worker.go
package worker

import (
	"context"
	"encoding/json"
	"log"

	"github.com/AhmadKusumahDEV/go-chat/internal/queue"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/rabbitmq/amqp091-go"
)

type MessageWorker struct {
	rabbitmq    *amqp091.Channel
	messageRepo repository.MessageRepository
}

func NewMessageWorker(
	rabbitmq *amqp091.Channel,
	messageRepo repository.MessageRepository,
) *MessageWorker {
	return &MessageWorker{
		rabbitmq:    rabbitmq,
		messageRepo: messageRepo,
	}
}

func (w *MessageWorker) Start(ctx context.Context) error {
	// 1. Declare queue
	// q, err := w.rabbitmq.QueueDeclare(
	// 	"message-persistence",
	// 	true,
	// 	false,
	// 	false,
	// 	false,
	// 	amqp091.Table{
	// 		"x-dead-letter-exchange": "chat.dlx",
	// 	},
	// )
	// if err != nil {
	// 	return err
	// }

	// err = w.rabbitmq.QueueBind(
	// 	q.Name,
	// 	"",
	// 	"chat.messages",
	// 	false,
	// 	nil,
	// )
	// if err != nil {
	// 	return err
	// }

	// err = w.rabbitmq.Qos(
	// 	10,
	// 	0,
	// 	false,
	// )
	// if err != nil {
	// 	return err
	// }

	// 4. Start consuming
	msgs, err := w.rabbitmq.Consume(
		"message-persistence",
		"message-persistence-worker",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Println("✅ MessageWorker started, waiting for messages...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("MessageWorker shutting down...")
				return

			case msg, ok := <-msgs:
				if !ok {
					log.Println("Channel closed")
					return
				}

				// Process message
				w.processMessage(&msg)
			}
		}
	}()

	return nil
}

func (w *MessageWorker) processMessage(msg *amqp091.Delivery) {
	log.Printf("📥 Processing message: %s", msg.MessageId)

	// 1. Deserialize event
	var event queue.MessageEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("❌ Failed to unmarshal: %v", err)
		msg.Nack(false, false) // Send to DLQ
		return
	}

	ctx := context.Background()
	err := w.messageRepo.Create(ctx, event.Message)

	if err != nil {
		log.Printf("❌ Failed to save to DB: %v", err)

		// Check retry count
		retryCount := 0
		if msg.Headers != nil {
			if count, ok := msg.Headers["x-retry-count"].(int32); ok {
				retryCount = int(count)
			}
		}

		if retryCount >= 3 {
			log.Printf("⚠️ Max retry reached, sending to DLQ")
			msg.Nack(false, false) // Send to DLQ
		} else {
			log.Printf("🔄 Retry attempt %d", retryCount+1)

			err := w.rabbitmq.Publish(
				"",
				msg.RoutingKey,
				false,
				false,
				amqp091.Publishing{
					Body:         msg.Body,
					ContentType:  "application/json",
					DeliveryMode: amqp091.Persistent,
					Headers: amqp091.Table{
						"x-retry-count": int32(retryCount + 1),
					},
				},
			)
			if err != nil {
				log.Printf("❌ Failed to publish retry: %v", err)
				msg.Nack(false, true) // Requeue
				return
			}
			msg.Ack(false)
		}
		return
	}

	// 3. Success - ACK message
	log.Printf("✅ Message saved to DB: %s", event.Message.ID)
	msg.Ack(false)
}
