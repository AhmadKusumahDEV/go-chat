package handlers

import (
	"fmt"
	"log"
	"net/http"

	wsManager "github.com/AhmadKusumahDEV/go-chat/internal/websocket"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// Dalam environment production, sesuaikan CheckOrigin untuk keamanan
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebsocketHandler interface {
	HandleConnection(c *gin.Context)
}

type WebsocketHandlerImpl struct {
	manager wsManager.WebSocketManager
}

// HandleConnection melakukan upgrade HTTP ke Websocket
func (w *WebsocketHandlerImpl) HandleConnection(c *gin.Context) {
	userInfo, exists := c.Get("user_info")
	if !exists {
		userID := c.Query("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: user info not found"})
			return
		}
		userInfo = userID // string
	}

	var userID string
	switch v := userInfo.(type) {
	case string:
		userID = v
	default:
		userID = fmt.Sprintf("%v", v)
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Gagal upgrade websocket: %v", err)
		return
	}

	w.manager.HandleConnection(conn, userID)
}

func NewWebsocketHandler(manager wsManager.WebSocketManager) WebsocketHandler {
	return &WebsocketHandlerImpl{
		manager: manager,
	}
}
