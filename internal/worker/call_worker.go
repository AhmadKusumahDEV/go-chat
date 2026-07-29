package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/AhmadKusumahDEV/go-chat/internal/websocket"
	"github.com/rabbitmq/amqp091-go"
)

type CallNotificationWorker struct {
	rabbitmq     *amqp091.Channel
	fcmClient    *messaging.Client
	userRepo     repository.RepositoryUser
	messageRepo  repository.MessageRepository
	firebaseRepo repository.RepositoryFirebase
	memberRepo   repository.RepositoryMembers
	roomRepo     repository.RepositoryRoom
}

func NewCallNotificationWorker(
	rabbitmq *amqp091.Channel,
	fcmClient *messaging.Client,
	userRepo repository.RepositoryUser,
	messageRepo repository.MessageRepository,
	firebaseRepo repository.RepositoryFirebase,
	memberRepo repository.RepositoryMembers,
	roomRepo repository.RepositoryRoom,
) *CallNotificationWorker {
	return &CallNotificationWorker{
		rabbitmq:     rabbitmq,
		fcmClient:    fcmClient,
		userRepo:     userRepo,
		messageRepo:  messageRepo,
		firebaseRepo: firebaseRepo,
		memberRepo:   memberRepo,
		roomRepo:     roomRepo,
	}
}

func (w *CallNotificationWorker) Start(ctx context.Context) error {
	data, err := w.rabbitmq.Consume(
		"call-notifications",
		"call.event",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Println(err)
		return nil
	}

	log.Println("✅ CallWorker started, waiting for messages...")

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("shutting down worker")
				return

			case payload, ok := <-data:
				if !ok {
					return
				}
				w.TriggerNotificationCall(&payload)
			}
		}
	}()

	return nil
}

func (w *CallNotificationWorker) TriggerNotificationCall(event *amqp091.Delivery) {
	ctx := context.Background()
	var data websocket.CallerforwardOffer

	if err := json.Unmarshal(event.Body, &data); err != nil {
		log.Printf("❌ [STEP 1/6] FAILED to unmarshal: %v", err)
		log.Printf("   📄 Raw body: %s", string(event.Body))
		event.Nack(false, false)
		return
	}

	token, err := w.firebaseRepo.GetTokensByUserID(ctx, data.TargetUserId)
	if err != nil {
		log.Printf("❌ user not have token active for trigger notifiaction: %v", err)
		event.Nack(false, false)
		return
	}

	avatarStr := ""
	if data.Avatar != nil {
		avatarStr = *data.Avatar
	}

	duration := time.Duration(45 * time.Second)

	callMessage := &messaging.Message{
		Data: map[string]string{
			"type":        "incoming_call",
			"call_id":     data.CallId,
			"caller_id":   data.CallerId,
			"caller_name": data.CallerName,
			"avatar":      avatarStr,
			"sdp":         data.Sdp,
			"mode":        data.Mode,
		},

		Android: &messaging.AndroidConfig{
			Priority: "high",
			TTL:      &duration,
		},

		Token: token,
	}

	response, err := w.fcmClient.Send(ctx, callMessage)
	if err != nil {
		log.Printf("❌ Failed to send P2P call trigger: %v", err)
		event.Nack(false, false)
		return
	}

	// Response akan mengembalikan string berupa Message ID unik jika berhasil
	log.Printf("✅ P2P Call triggered successfully. Message ID: %s", response)

	event.Ack(false)
}
