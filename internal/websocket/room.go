package websocket

import (
	"log"
	"sync"
	"time"
)

type WebSocketRoom interface {
	Run()
	Register(client *Client)
	Unregister(client *Client)
	Broadcast(message []byte)
	BroadcastExcept(message []byte, exceptUserID string)
	ClientCount() int
	IsEmpty() bool
	Stop()
	GetClients() map[string]*Client // userID -> Client
}

// Room represents a chat room
type Room struct {
	ID         string
	clients    map[string]*Client // userID -> Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	stop       chan bool
	mutex      sync.RWMutex
}

// NewRoom creates a new chat room
func NewRoom(id string) WebSocketRoom {
	return &Room{
		ID:         id,
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
		stop:       make(chan bool),
	}
}

// Run starts the room's message handling loop
func (r *Room) Run() {
	log.Printf("Room %s started", r.ID)

	for {
		select {
		case client := <-r.register:
			r.handleRegister(client)

		case client := <-r.unregister:
			r.handleUnregister(client)

		case message := <-r.broadcast:
			r.handleBroadcast(message)

		case <-r.stop:
			log.Printf("Room %s stopped", r.ID)
			return
		}
	}
}

// handleRegister adds a client to the room
func (r *Room) handleRegister(client *Client) {
	r.mutex.Lock()
	r.clients[client.UserID] = client
	r.mutex.Unlock()

	log.Printf("User %s joined room %s", client.UserID, r.ID)
}

// handleUnregister removes a client from the room
func (r *Room) handleUnregister(client *Client) {
	r.mutex.Lock()
	delete(r.clients, client.UserID)
	r.mutex.Unlock()

	log.Printf("User %s left room %s", client.UserID, r.ID)
}

// handleBroadcast sends a message to all clients in the room
func (r *Room) handleBroadcast(message *BroadcastMessage) {
	log.Printf("Broadcasting message in room %s", r.ID)

	for userID, client := range r.clients {
		// Skip sender
		if message.Sender != "" && userID == message.Sender {
			continue
		}

		select {
		case client.Send <- message.Data:
			// sent successfully

		case <-time.After(5 * time.Second):
			log.Printf("Client %s timeout in room %s", userID, r.ID)

			go func(c *Client) {
				r.unregister <- c
			}(client)
		}
	}
}

// Register adds a client to the room
func (r *Room) Register(client *Client) {
	r.register <- client
}

// Unregister removes a client from the room
func (r *Room) Unregister(client *Client) {
	r.unregister <- client
}

// Broadcast sends a message to all clients in the room
func (r *Room) Broadcast(message []byte) {
	r.broadcast <- &BroadcastMessage{
		RoomID: r.ID,
		Data:   message,
	}
}

func (r *Room) BroadcastExcept(message []byte, exceptUserID string) {
	r.broadcast <- &BroadcastMessage{
		RoomID: r.ID,
		Data:   message,
		Sender: exceptUserID,
	}
}

func (r *Room) ClientCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.clients)
}

// IsEmpty checks if the room has no clients
func (r *Room) IsEmpty() bool {
	return r.ClientCount() == 0
}

// Stop gracefully stops the room
func (r *Room) Stop() {
	close(r.stop)
}

// GetClients returns all clients in the room
func (r *Room) GetClients() map[string]*Client {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	clients := make(map[string]*Client)
	for userID, client := range r.clients {
		clients[userID] = client
	}
	return clients
}
