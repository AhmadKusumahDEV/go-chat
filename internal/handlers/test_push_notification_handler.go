package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/gin-gonic/gin"
	"firebase.google.com/go/v4/messaging"
)

// TestPushNotificationHandler handles testing push notifications
type TestPushNotificationHandler struct {
	fcmClient    *messaging.Client
	firebaseRepo repository.RepositoryFirebase
	roomRepo     repository.RepositoryRoom
}

type TestPushRequest struct {
	UserID   string `json:"userId" binding:"required"`   // Target user ID
	RoomID   string `json:"roomId"`                      // Optional room ID for context
	Title    string `json:"title" binding:"required"`     // Notification title
	Body     string `json:"body" binding:"required"`      // Notification body
	Message  string `json:"message"`                      // Optional message content
}

func NewTestPushNotificationHandler(
	fcmClient *messaging.Client,
	firebaseRepo repository.RepositoryFirebase,
	roomRepo repository.RepositoryRoom,
) *TestPushNotificationHandler {
	return &TestPushNotificationHandler{
		fcmClient:    fcmClient,
		firebaseRepo: firebaseRepo,
		roomRepo:     roomRepo,
	}
}

// SendTestPushNotification sends a test FCM push notification to a specific user's device
// POST /test/push-notification
func (h *TestPushNotificationHandler) SendTestPushNotification(c *gin.Context) {
	var req TestPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Step 1: Get FCM tokens for the user
	tokens, err := h.firebaseRepo.GetTokensByUserIDs(ctx, []string{req.UserID})
	if err != nil {
		log.Printf("❌ Failed to get tokens for user %s: %v", req.UserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get FCM tokens"})
		return
	}

	if len(tokens) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active FCM tokens found for this user. User may not have registered their device."})
		return
	}

	// Step 2: Get room name if roomID provided
	roomName := "Chat"
	if req.RoomID != "" && h.roomRepo != nil {
		if name, err := h.roomRepo.FindRoomName(ctx, req.RoomID); err == nil {
			roomName = name
		}
	}

	// Step 3: Build notification data
	notificationTitle := req.Title
	if req.RoomID != "" {
		notificationTitle = "[" + roomName + "] " + req.Title
	}

	notificationBody := req.Body
	if len(notificationBody) > 100 {
		notificationBody = notificationBody[:97] + "..."
	}

	// Step 4: Send FCM notification to first token (one device)
	token := tokens[0]
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: notificationTitle,
			Body:  notificationBody,
		},
		Data: map[string]string{
			"room_id": req.RoomID,
			"type":    "test_notification",
		},
		Token: token,
	}

	// Step 5: Send the notification
	response, err := h.fcmClient.Send(ctx, message)
	if err != nil {
		log.Printf("❌ Failed to send FCM notification: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send push notification"})
		return
	}

	log.Printf("✅ Test push notification sent successfully")
	log.Printf("   📱 Token: %s...", token[:min(6, len(token))])
	log.Printf("   🔔 Title: %q", notificationTitle)
	log.Printf("   📝 Body: %q", notificationBody)
	log.Printf("   🆔 MessageID: %s", response)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Push notification sent successfully",
		"messageId": response,
		"token":     maskToken(token),
		"title":     notificationTitle,
		"body":      notificationBody,
	})
}

// SendTestMulticast sends test notification to all user's devices
// POST /test/push-notification/multicast
func (h *TestPushNotificationHandler) SendTestMulticast(c *gin.Context) {
	var req TestPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Step 1: Get all FCM tokens for the user
	tokens, err := h.firebaseRepo.GetTokensByUserIDs(ctx, []string{req.UserID})
	if err != nil {
		log.Printf("❌ Failed to get tokens for user %s: %v", req.UserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get FCM tokens"})
		return
	}

	if len(tokens) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active FCM tokens found for this user"})
		return
	}

	// Step 2: Build notification
	notificationBody := req.Body
	if len(notificationBody) > 100 {
		notificationBody = notificationBody[:97] + "..."
	}

	// Step 3: Send multicast to all tokens
	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: req.Title,
			Body:  notificationBody,
		},
		Data: map[string]string{
			"room_id": req.RoomID,
			"type":    "test_notification",
		},
		Tokens: tokens,
	}

	response, err := h.fcmClient.SendEachForMulticast(ctx, message)
	if err != nil {
		log.Printf("❌ Failed to send FCM multicast: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send push notification"})
		return
	}

	log.Printf("✅ Test multicast sent: %d success, %d failure", response.SuccessCount, response.FailureCount)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Push notifications sent",
		"successCount": response.SuccessCount,
		"failureCount": response.FailureCount,
		"totalDevices": len(tokens),
	})
}

// GetUserTokens returns FCM tokens for a user (for debugging)
// GET /test/push-notification/tokens/:userId
func (h *TestPushNotificationHandler) GetUserTokens(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}

	ctx := context.Background()
	tokens, err := h.firebaseRepo.GetTokensByUserIDs(ctx, []string{userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tokens"})
		return
	}

	maskedTokens := make([]string, len(tokens))
	for i, t := range tokens {
		maskedTokens[i] = maskToken(t)
	}

	c.JSON(http.StatusOK, gin.H{
		"userId":       userID,
		"tokenCount":   len(tokens),
		"tokens":       maskedTokens,
		"hasRegistered": len(tokens) > 0,
	})
}

// maskToken masks a token for logging/display
func maskToken(token string) string {
	if len(token) <= 12 {
		return "****"
	}
	return token[:6] + "..." + token[len(token)-6:]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var _ config.RouteRegistrar = (*TestPushRouter)(nil)

type TestPushRouter struct {
	handler *TestPushNotificationHandler
}

func (r *TestPushRouter) RegisterRoutes(engine *gin.Engine, srv *config.Server) {
	test := engine.Group("/test")
	{
		test.POST("/push-notification", r.handler.SendTestPushNotification)
		test.POST("/push-notification/multicast", r.handler.SendTestMulticast)
		test.GET("/push-notification/tokens/:userId", r.handler.GetUserTokens)
	}
}

func NewTestPushRouter(handler *TestPushNotificationHandler) *TestPushRouter {
	return &TestPushRouter{handler: handler}
}