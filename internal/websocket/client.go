package websocket

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketClient interface {
	ReadPump(processor WebsocketProcessor)
	WritePump()
	GetUserID() string
	SendMessage(message []byte)
	Close()
}

type Client struct {
	Hub       WebSocketHub
	Conn      *websocket.Conn
	Send      chan []byte
	UserID    string
	closeOnce sync.Once
}

func NewClient(conn *websocket.Conn, hub WebSocketHub, userID string) WebSocketClient {
	return &Client{
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
	}
}

// Close implements WebSocketClient.
func (c *Client) ReadPump(processor WebsocketProcessor) {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512000) // 512KB
	err := c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	if err != nil {
		log.Println(err)
		return
	}
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// Queue message for processing
		processor.QueueMessage(&ProcessMessage{
			Client:  c,
			Message: message,
			UserID:  c.UserID,
		})
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second) // Ping interval
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			log.Println(ok)
			err := c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				log.Println(err)
				return
			}
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Println("Failed to get next writer:", err)
				log.Println(err)
				return
			}
			res, err := w.Write(message)
			log.Println(res)
			log.Println(err)
			if err != nil {
				log.Println("Failed to write message:", err)
				log.Println(err)
				return
			}

			// Add queued messages to current WebSocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(message []byte) {
	select {
	case c.Send <- message:
	default:
		c.Close()
		c.Hub.Unregister(c)
	}
}

// GetUserID returns the client's user ID
func (c *Client) GetUserID() string {
	return c.UserID
}

// Close closes the client connection
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.Send)
		c.Conn.Close()
	})
}
