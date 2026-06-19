package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/queue"
	"github.com/gorilla/websocket"
)

type WebSocketManager interface {
	Start()
	Stop()
	HandleConnection(conn *websocket.Conn, userID string) *Client
	BatchJoinRoom(roomID []string, c *Client)
	GetRoomStats(roomID string) (int, bool)
	GetManagerStats() ManagerStats
	collectStats()
	updateStats(updateFunc func(*ManagerStats))
	SendNotificationToUser(userID string, notification []byte) error
	SendNotificationToAll(notification []byte)
	GetConnectedUsers() []string
	BroadcastToRoom(roomID string, message []byte)
	BroadcastToRoomExcept(roomID string, message []byte, exceptUserID string)
	MessageByID(messageID string) (*response.MessageResponse, error)
}

// WebSocketManagerImpl implements WebSocketManager interface
type WebSocketManagerImpl struct {
	hub        WebSocketHub
	processor  WebsocketProcessor
	shutdown   chan struct{}
	wg         sync.WaitGroup
	stats      *ManagerStats
	statsMutex sync.RWMutex
}

type ManagerStats struct {
	TotalConnections  int
	ActiveRooms       int
	ActiveClients     int
	MessagesProcessed int
	StartTime         time.Time
}

type BroadcastMessage struct {
	RoomID   string          `json:"room_id"`
	Data     json.RawMessage `json:"data"`
	SenderID string          `json:"sender_id,omitempty"`
	Type     string          `json:"type,omitempty"`
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(messageService MessageSender, publisher queue.Publisher) WebSocketManager {
	hub := NewHub()
	processor := NewMessageProcessor(10, hub, messageService, publisher)

	manager := &WebSocketManagerImpl{
		hub:       hub,
		processor: processor,
		shutdown:  make(chan struct{}),
		stats: &ManagerStats{
			StartTime: time.Now(),
		},
	}

	return manager
}

// Start begins all WebSocket components
func (m *WebSocketManagerImpl) Start() {
	log.Println("Starting WebSocket manager...")

	// Start hub (1 goroutine)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.hub.Run(m.shutdown)
		log.Println("Hub stopped")
	}()

	// Start message processor (N goroutines)
	m.processor.Start()

	// Start stats collector (1 goroutine)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.collectStats()
	}()

	log.Println("WebSocket manager started successfully")
}

// Stop gracefully shuts down all components
func (m *WebSocketManagerImpl) Stop() {
	log.Println("Stopping WebSocket manager...")
	close(m.shutdown)
	m.processor.Stop()
	m.wg.Wait()
	log.Println("WebSocket manager stopped")
}

// HandleConnection creates a new client and starts communication
func (m *WebSocketManagerImpl) HandleConnection(conn *websocket.Conn, userID string) *Client {
	m.updateStats(func(stats *ManagerStats) {
		stats.TotalConnections++
		stats.ActiveClients++
	})

	// 3. Create & Register client
	client := NewClient(conn, m.hub, userID)
	c, ok := client.(*Client)
	if !ok {
		log.Printf("Invalid client type: %T", client)
		conn.Close()
		return nil
	}

	m.hub.Register(c)

	// 4. Start client goroutines
	m.wg.Add(2)
	go func() {
		defer m.wg.Done()
		client.ReadPump(m.processor)

		// Update stats on client disconnect (ReadPump returns)
		m.updateStats(func(stats *ManagerStats) {
			stats.ActiveClients--
		})
	}()

	go func() {
		defer m.wg.Done()
		client.WritePump()
	}()

	return c
}

// GetRoomStats returns statistics for a room
func (m *WebSocketManagerImpl) GetRoomStats(roomID string) (int, bool) {
	count := m.hub.GetRoomClients(roomID)
	return count, count > 0
}

// GetManagerStats returns overall manager statistics
func (m *WebSocketManagerImpl) GetManagerStats() ManagerStats {
	m.statsMutex.RLock()
	defer m.statsMutex.RUnlock()
	return *m.stats
}

// collectStats periodically collects and updates statistics
func (m *WebSocketManagerImpl) collectStats() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.updateStats(func(stats *ManagerStats) {
				// Update active rooms count
				stats.ActiveRooms = m.hub.GetRoomCount()
			})

			log.Printf("WebSocket Stats: Clients=%d, Rooms=%d, TotalMsgs=%d",
				m.stats.ActiveClients, m.stats.ActiveRooms, m.stats.MessagesProcessed)

		case <-m.shutdown:
			return
		}
	}
}

// updateStats safely updates statistics
func (m *WebSocketManagerImpl) updateStats(updateFunc func(*ManagerStats)) {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()
	updateFunc(m.stats)
}

func (m *WebSocketManagerImpl) SendNotificationToUser(userID string, notification []byte) error {
	return m.hub.BroadcastToUser(userID, notification)
}

func (m *WebSocketManagerImpl) SendNotificationToAll(notification []byte) {
	m.hub.BroadcastToAllUsers(notification)
}

func (m *WebSocketManagerImpl) GetConnectedUsers() []string {
	return m.hub.GetAllConnectedUsers()
}

func (m *WebSocketManagerImpl) BroadcastToRoom(roomID string, message []byte) {
	m.hub.BroadcastToRoom(roomID, message)
}

func (m *WebSocketManagerImpl) BroadcastToRoomExcept(roomID string, message []byte, exceptUserID string) {
	m.hub.BroadcastToRoomExcept(roomID, message, exceptUserID)
}

func (m *WebSocketManagerImpl) MessageByID(messageID string) (*response.MessageResponse, error) {
	return m.processor.GetMessageID(context.Background(), messageID)
}

func (m *WebSocketManagerImpl) BatchJoinRoom(roomID []string, c *Client) {
	ctx := context.Background()

	m.hub.BatchSubscribeToRoom(ctx, roomID, c)
}
