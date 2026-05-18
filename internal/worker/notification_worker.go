// internal/worker/notification_worker.go
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/AhmadKusumahDEV/go-chat/internal/queue"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/rabbitmq/amqp091-go"
)

var (
	invalidTokenErrorCodes = []string{
		"UNREGISTERED",       // App instance was unregistered
		"INVALID_ARGUMENT",   // Token is malformed
		"SENDER_ID_MISMATCH", // Token doesn't belong to this sender
	}

	retryableErrorCodes = []string{
		"INTERNAL",       // Internal server error
		"UNAVAILABLE",    // Server unavailable
		"QUOTA_EXCEEDED", // Quota exceeded - should wait
	}
)

type NotificationWorker struct {
	rabbitmq     *amqp091.Channel
	fcmClient    *messaging.Client
	userRepo     repository.RepositoryUser
	messageRepo  repository.MessageRepository
	firebaseRepo repository.RepositoryFirebase
	memberRepo  repository.RepositoryMembers
	roomRepo    repository.RepositoryRoom
}

func NewNotificationWorker(
	rabbitmq *amqp091.Channel,
	fcmClient *messaging.Client,
	userRepo repository.RepositoryUser,
	messageRepo repository.MessageRepository,
	firebaseRepo repository.RepositoryFirebase,
	memberRepo repository.RepositoryMembers,
	roomRepo repository.RepositoryRoom,
) *NotificationWorker {
	return &NotificationWorker{
		rabbitmq:     rabbitmq,
		fcmClient:    fcmClient,
		userRepo:     userRepo,
		messageRepo:  messageRepo,
		firebaseRepo: firebaseRepo,
		memberRepo:  memberRepo,
		roomRepo:    roomRepo,
	}
}

// Start - Running terus menerus
func (w *NotificationWorker) Start(ctx context.Context) error {
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

	err = w.rabbitmq.QueueBind(
		q.Name,
		"notification.#", // routing key pattern (# matches 0 or more words)
		"chat.notifications",
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Printf("✅ Queue %q bound to exchange %q with routing key %q", q.Name, "chat.notifications", "notification.#")

	err = w.rabbitmq.Qos(5, 0, false)
	if err != nil {
		return err
	}

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
	startTime := time.Now()

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📥 [START] Processing notification | MsgID=%s | RoutingKey=%s",
		msg.MessageId, msg.RoutingKey)

	// 1. Deserialize event
	log.Printf("🔍 [STEP 1/6] Deserializing message body...")
	var event queue.NotificationEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("❌ [STEP 1/6] FAILED to unmarshal: %v", err)
		log.Printf("   📄 Raw body: %s", string(msg.Body))
		msg.Nack(false, false)
		return
	}

	log.Printf("✅ [STEP 1/6] SUCCESS | Type=%s | RoomID=%s | MessageID=%s | SenderID=%s",
		event.Type, event.RoomID, event.MessageID, event.SenderID)
	log.Printf("   📝 Title: %q | Body: %q", event.Title, event.Body)

	ctx := context.Background()

	// 2. Fetch room name for better notification
	log.Printf("🔍 [STEP 2/6] Fetching room name...")
	roomName := "Chat" // default
	if w.roomRepo != nil {
		if name, err := w.roomRepo.FindRoomName(ctx, event.RoomID); err == nil {
			roomName = name
			log.Printf("✅ [STEP 2/6] SUCCESS | RoomName=%q", roomName)
		} else {
			log.Printf("⚠️ [STEP 2/6] Using default room name: %v", err)
		}
	} else {
		log.Printf("⚠️ [STEP 2/6] Room repo not configured, using default name")
	}

	// 3. Fetch room members
	log.Printf("🔍 [STEP 3/6] Fetching room members from database...")
	log.Printf("   📋 RoomID: %s", event.RoomID)

	memberIDs, err := w.memberRepo.GetRoomMemberIDs(ctx, event.RoomID)
	if err != nil {
		log.Printf("❌ [STEP 3/6] FAILED to fetch room members: %v", err)
		log.Printf("   🔄 Action: Will requeue message for retry")
		msg.Nack(false, true)
		return
	}

	log.Printf("✅ [STEP 3/6] SUCCESS | Total members in room: %d", len(memberIDs))
	log.Printf("   👥 Members: %v", memberIDs)

	// Remove sender from target members
	var targetIDs []string
	for _, id := range memberIDs {
		if id != event.SenderID {
			targetIDs = append(targetIDs, id)
		}
	}

	log.Printf("👤 Excluded sender (%s) from targets | Remaining targets: %d", event.SenderID, len(targetIDs))

	if len(targetIDs) == 0 {
		log.Printf("ℹ️  [STEP 3/6] COMPLETED (no action needed)")
		log.Printf("   📝 Reason: No other members in room besides sender")
		log.Printf("   ✅ ACK sent - discarding message")
		msg.Ack(false)
		return
	}

	log.Printf("📨 Will notify %d member(s): %v", len(targetIDs), targetIDs)

	// 4. Fetch FCM tokens
	log.Printf("🔍 [STEP 4/6] Fetching FCM tokens from database...")
	log.Printf("   📋 Target user IDs: %v", targetIDs)

	tokens, err := w.firebaseRepo.GetTokensByUserIDs(ctx, targetIDs)
	if err != nil {
		log.Printf("❌ [STEP 4/6] FAILED to fetch FCM tokens: %v", err)
		log.Printf("   🔄 Action: Will requeue message for retry")
		msg.Nack(false, true)
		return
	}

	if len(tokens) == 0 {
		log.Printf("⚠️  [STEP 4/6] COMPLETED (no action needed)")
		log.Printf("   📝 Reason: No active FCM tokens found for any of the %d target users", len(targetIDs))
		log.Printf("   🔍 This means: Users have NOT registered their device for push notifications")
		log.Printf("   💡 To fix: Users need to call the FCM registration endpoint first")
		log.Printf("   ✅ ACK sent - discarding message")
		msg.Ack(false)
		return
	}

	log.Printf("✅ [STEP 4/6] SUCCESS | Found %d active FCM token(s)", len(tokens))
	if len(tokens) <= 5 {
		log.Printf("   📱 Tokens: %v", maskTokens(tokens))
	} else {
		log.Printf("   📱 Tokens: %d tokens (first 3: %v...)", len(tokens), maskTokens(tokens[:3]))
	}

	// 5. Build notification title with room name and sender name
	notificationTitle := fmt.Sprintf("[%s] %s", roomName, event.SenderName)
	notificationBody := event.Body

	// Truncate body if too long
	if len(notificationBody) > 100 {
		notificationBody = notificationBody[:97] + "..."
	}

	log.Printf("🔍 [STEP 5/6] Sending FCM multicast notification...")
	log.Printf("   📱 Sending to %d device(s)", len(tokens))
	log.Printf("   🔔 Notification Title: %q", notificationTitle)
	log.Printf("   🔔 Notification Body: %q", notificationBody)
	log.Printf("   📦 Data payload: room_id=%s, message_id=%s, type=%s",
		event.RoomID, event.MessageID, event.Type)

	response, err := w.fcmClient.SendEachForMulticast(ctx, &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: notificationTitle,
			Body:  notificationBody,
		},
		Data: map[string]string{
			"room_id":    event.RoomID,
			"message_id": event.MessageID,
			"type":       event.Type,
		},
		Tokens: tokens,
	})

	if err != nil {
		log.Printf("❌ [STEP 5/6] FAILED to send FCM multicast: %v", err)
		log.Printf("   🔄 Action: Will requeue message for retry")
		log.Printf("   💡 Possible causes: Firebase quota exceeded, invalid tokens, network issues")
		msg.Nack(false, true)
		return
	}

	// 6. Process results and handle invalid tokens
	log.Printf("✅ [STEP 5/6] FCM Send Complete")
	log.Printf("   📊 Results: Success=%d | Failure=%d | Total=%d",
		response.SuccessCount, response.FailureCount, len(tokens))

	var invalidTokens []string

	if response.FailureCount > 0 {
		log.Printf("⚠️  [STEP 6/6] Processing failed token deliveries...")
		for i, resp := range response.Responses {
			if !resp.Success {
				token := tokens[i]
				errCode := getErrorCode(resp.Error)

				log.Printf("   ❌ Token[%d]: %s | Error: %v", i, maskSingleToken(token), resp.Error)

				if isInvalidTokenError(errCode) {
					invalidTokens = append(invalidTokens, token)
					log.Printf("   🔴 PERMANENT ERROR (%s) - Token will be deactivated in DB", errCode)
				} else {
					log.Printf("   🟡 TRANSIENT ERROR (%s) - Will keep token active for retry", errCode)
				}
			}
		}
	} else {
		log.Printf("✅ [STEP 6/6] All %d notifications delivered successfully!", response.SuccessCount)
	}

	// Deactivate invalid tokens in database
	if len(invalidTokens) > 0 {
		log.Printf("🔍 Deactivating %d invalid FCM token(s) in database...", len(invalidTokens))

		deactivatedCount, err := w.firebaseRepo.DeactivateTokensByUserIDs(ctx, targetIDs, tokens)
		if err != nil {
			log.Printf("❌ FAILED to deactivate invalid tokens: %v", err)
			log.Printf("   ⚠️  Warning: Invalid tokens remain active in database")
		} else {
			log.Printf("✅ SUCCESS | Deactivated %d invalid token(s)", deactivatedCount)
			log.Printf("   🔴 Tokens removed: %v", maskTokens(invalidTokens))
		}
	}

	// Summary
	duration := time.Since(startTime)
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("✅ [COMPLETE] Notification processing finished")
	log.Printf("   ⏱️  Duration: %v", duration)
	log.Printf("   📋 MessageID: %s | RoomID: %s | RoomName: %s", event.MessageID, event.RoomID, roomName)
	log.Printf("   📱 Devices notified: %d/%d", response.SuccessCount, len(tokens))
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	msg.Ack(false)
}

// maskTokens masks FCM tokens for logging (show first and last 4 chars)
func maskTokens(tokens []string) []string {
	masked := make([]string, len(tokens))
	for i, t := range tokens {
		masked[i] = maskSingleToken(t)
	}
	return masked
}

// maskSingleToken masks a single FCM token
func maskSingleToken(token string) string {
	if len(token) <= 12 {
		return "****"
	}
	return fmt.Sprintf("%s...%s", token[:6], token[len(token)-6:])
}

// getErrorCode extracts the error code from Firebase Messaging error
func getErrorCode(err error) string {
	if err == nil {
		return ""
	}
	errStr := err.Error()
	parts := strings.SplitN(errStr, ":", 2)
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return errStr
}

// isInvalidTokenError checks if error code indicates token is permanently invalid
func isInvalidTokenError(errCode string) bool {
	for _, code := range invalidTokenErrorCodes {
		if code == errCode {
			return true
		}
	}
	return false
}

// isRetryableError checks if error code indicates transient failure (should retry)
func isRetryableError(errCode string) bool {
	for _, code := range retryableErrorCodes {
		if code == errCode {
			return true
		}
	}
	return false
}
