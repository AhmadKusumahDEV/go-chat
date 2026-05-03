package websocket

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/gorilla/websocket"
)

type WebSocketManager interface {
	Start()
	Stop()
	HandleConnection(conn *websocket.Conn, userID string)
	GetRoomStats(roomID string) (int, bool)
	GetManagerStats() ManagerStats
	collectStats()
	updateStats(updateFunc func(*ManagerStats))
}

// WebSocketManagerImpl implements WebSocketManager interface
type WebSocketManagerImpl struct {
	hub            WebSocketHub
	processor      WebsocketProcessor
	shutdown       chan struct{}
	wg             sync.WaitGroup
	stats          *ManagerStats
	statsMutex     sync.RWMutex
	roomRepository repository.RepositoryRoom
}

type ManagerStats struct {
	TotalConnections  int
	ActiveRooms       int
	ActiveClients     int
	MessagesProcessed int
	StartTime         time.Time
}

type BroadcastMessage struct {
	RoomID string `json:"room_id"`
	Data   []byte `json:"data"`
	Sender string `json:"sender,omitempty"` // UserID of sender (for excluding)
	Type   string `json:"type,omitempty"`   // Message type
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(roomRepository repository.RepositoryRoom) WebSocketManager {
	hub := NewHub()
	processor := NewMessageProcessor(10, hub)

	manager := &WebSocketManagerImpl{
		hub:       hub,
		processor: processor,
		shutdown:  make(chan struct{}),
		stats: &ManagerStats{
			StartTime: time.Now(),
		},
		roomRepository: roomRepository,
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
func (m *WebSocketManagerImpl) HandleConnection(conn *websocket.Conn, userID string) {
	rooms, err := m.roomRepository.FindAllRoomByUserID(context.Background(), userID)
	if err != nil {
		log.Printf("Error finding rooms for user %s: %v", userID, err)
		conn.Close()
		return
	}

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
		return
	}

	m.hub.Register(c)

	for _, room := range rooms {
		m.hub.subscribeToRoom(room.ID.String(), c)
	}

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

	log.Printf("New client connected: user=%s", userID)
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
