package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/queue"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/rabbitmq/amqp091-go"
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

type UserSpecifiedRequirements interface {
	GetDetailUser(ctx context.Context, userId uuid.UUID) (*response.UserResponse, error)
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

type CallerSendOffer struct {
	CallId       string `json:"call_id"`
	TargetUserId string `json:"target_user_id"`
	Sdp          string `json:"sdp"`
	Mode         string `json:"mode"`
}

type CallerforwardOffer struct {
	CallId       string  `json:"call_id"`
	TargetUserId string  `json:"target_user_id,omitempty"`
	CallerName   string  `json:"caller_name"`
	CallerId     string  `json:"caller_id"`
	Avatar       *string `json:"avatar,omitempty"`
	Sdp          string  `json:"sdp"`
	Mode         string  `json:"mode"`
}

type CallSendAnswer struct {
	CallId       string `json:"call_id"`
	TargetUserId string `json:"target_user_id"` // send back to caller init
	Sdp          string `json:"sdp"`
}

type CallForwardAnswer struct {
	CallId string `json:"call_id"`
	Sdp    string `json:"sdp"`
}

type CallSendIce struct {
	CallId       string    `json:"call_id"`
	TargetUserId string    `json:"target_user_id"`
	Candidate    Candidate `json:"candidate"`
}

type CallForwardIce struct {
	CallId    string    `json:"call_id"`
	Candidate Candidate `json:"candidate"`
}

type CallSendHangup struct {
	CallId       string `json:"call_id"`
	TargetUserId string `json:"target_user_id"`
}

type CallForwardHangup struct {
	CallId string `json:"call_id"`
}

type CallSendMute struct {
	CallId       string `json:"call_id"`
	TargetUserId string `json:"target_user_id"`
	Muted        bool   `json:"muted"`
}

type CallForwardMute struct {
	CallId string `json:"call_id"`
	Muted  bool   `json:"muted"`
}

type FormatFowardEvent struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Candidate struct {
	Candidate     string  `json:"candidate"`
	SdpMLineIndex *int    `json:"sdpMLineIndex"`
	SdpMid        *string `json:"sdpMid"`
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(messageService MessageSender, room GetRoomSpecifice, publisher queue.Publisher, channel *amqp091.Channel, userServices UserSpecifiedRequirements) WebSocketManager {
	hub := NewHub()
	processor := NewMessageProcessor(10, hub, messageService, room, publisher, channel, userServices)

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
