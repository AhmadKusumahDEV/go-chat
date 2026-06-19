package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/services"
	wsManager "github.com/AhmadKusumahDEV/go-chat/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebsocketHandler interface {
	HandleConnection(c *gin.Context)
	GetConnectedUsers(c *gin.Context)
	BroadcastToAll(c *gin.Context)
	SendToUser(c *gin.Context)
}

type WebsocketHandlerImpl struct {
	manager wsManager.WebSocketManager
	room    services.RoomService
}

func (w *WebsocketHandlerImpl) HandleConnection(c *gin.Context) {
	userInfo, exists := c.Get("user_id")
	if !exists {
		userID := c.Query("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: user info not found"})
			return
		}
		userInfo = userID // string
	}

	var tempRooms []string

	rooms, err := w.room.GetRoomByUserID(c.Request.Context(), userInfo.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	for _, room := range rooms {
		tempRooms = append(tempRooms, room.ID)
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Gagal upgrade websocket: %v", err)
		return
	}

	client := w.manager.HandleConnection(conn, userInfo.(string))

	// join room
	w.manager.BatchJoinRoom(tempRooms, client)
}

func (h *WebsocketHandlerImpl) SendToUser(c *gin.Context) {
	userID := c.Param("userId")

	var req struct {
		Type    string                 `json:"type" binding:"required"`
		Title   string                 `json:"title" binding:"required"`
		Message string                 `json:"message" binding:"required"`
		Data    map[string]interface{} `json:"data,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notification, _ := json.Marshal(map[string]interface{}{
		"type":    req.Type,
		"title":   req.Title,
		"message": req.Message,
		"data":    req.Data,
	})

	err := h.manager.SendNotificationToUser(userID, notification)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification sent successfully"})
}

func (h *WebsocketHandlerImpl) BroadcastToAll(c *gin.Context) {
	headNotif := response.NotificationResponse{
		Type:       "notification",
		Title:      "From Websocket",
		Body:       "soliiddddddd",
		SenderID:   "1234878743",
		SenderName: "pria solo sejati",
		MessageID:  "1234848u834798373987948543",
	}

	notification, _ := json.Marshal(headNotif)

	h.manager.SendNotificationToAll(notification)

	c.JSON(http.StatusOK, gin.H{"message": "Notification broadcasted successfully"})
}

func (h *WebsocketHandlerImpl) GetConnectedUsers(c *gin.Context) {
	users := h.manager.GetConnectedUsers()
	c.JSON(http.StatusOK, gin.H{
		"count": len(users),
		"users": users,
	})
}

func NewWebsocketHandler(manager wsManager.WebSocketManager, room services.RoomService) WebsocketHandler {
	return &WebsocketHandlerImpl{
		manager: manager,
		room:    room,
	}
}
