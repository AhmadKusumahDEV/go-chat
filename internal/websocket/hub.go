package websocket

import (
	"errors"
	"log"
	"sync"
	"time"
)

type WebSocketHub interface {
	Run(signal chan struct{})
	Register(client *Client)
	Unregister(client *Client)
	BroadcastToRoom(roomID string, message []byte)
	BroadcastToRoomExcept(roomID string, message []byte, exceptUserID string)
	GetRoomClients(roomID string) int
	GetRoom(roomID string) WebSocketRoom
	HandleBroadcast(message *BroadcastMessage)
	GetRoomCount() int
	SubscribeToRoom(roomID string, client *Client)
	handleRegister(client *Client)
	handleUnregister(client *Client)
	UnsubscribeFromRoom(roomID string, client *Client)
	BroadcastToUser(userID string, message []byte) error
	BroadcastToAllUsers(message []byte)
	GetAllConnectedUsers() []string
}

// Hub implements WebSocketHub interface
type Hub struct {
	client     map[*Client]bool
	rooms      map[string]WebSocketRoom
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mutex      sync.RWMutex
}

// subscribeToRoom implements WebSocketHub.

func NewHub() WebSocketHub {
	return &Hub{
		client:     make(map[*Client]bool),
		rooms:      make(map[string]WebSocketRoom),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 1000),
		mutex:      sync.RWMutex{},
	}
}

func (h *Hub) Run(signal chan struct{}) {
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case message := <-h.broadcast:
			h.HandleBroadcast(message)

		case <-signal:
			return
		}
	}
}

// unsubscribeFromRoom implements WebSocketHub.
func (h *Hub) SubscribeToRoom(roomID string, client *Client) {
	defer h.mutex.Unlock()
	h.mutex.Lock()
	room, exists := h.rooms[roomID]
	if !exists {
		room = NewRoom(roomID)
		h.rooms[roomID] = room
		go room.Run()
		log.Printf("Created new room: %s", roomID)
	}

	room.Register(client)
	log.Printf("Client %s joined room %s", client.UserID, roomID)
}

func (h *Hub) UnsubscribeFromRoom(roomID string, client *Client) {
	h.mutex.Lock()
	room, exists := h.rooms[roomID]
	h.mutex.Unlock()

	if exists {
		room.Unregister(client)

		// Clean up empty rooms
		if room.IsEmpty() {
			h.mutex.Lock()
			delete(h.rooms, roomID)
			h.mutex.Unlock()
			room.Stop()
			log.Printf("Room %s cleaned up (no clients)", roomID)
		}
	}

	log.Printf("Client %s left room %s", client.UserID, roomID)
}

// handleRegister processes client registration
func (h *Hub) handleRegister(Client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for c := range h.client {
		if c.UserID == Client.UserID {
			log.Printf("Client %s already registered", Client.UserID)
			return
		}
	}

	h.client[Client] = true
	log.Printf("register Client %s", Client.UserID)
}

// handleUnregister processes client removal
func (h *Hub) handleUnregister(client *Client) {
	h.mutex.Lock()
	delete(h.client, client)

	// Copy active rooms to remove client
	var activeRooms []WebSocketRoom
	for _, room := range h.rooms {
		activeRooms = append(activeRooms, room)
	}
	h.mutex.Unlock()

	for _, room := range activeRooms {
		room.Unregister(client)
	}

	client.Close() // Ensure the client resources are properly closed
	log.Printf("Client %s unregistered from Hub", client.UserID)
}

// HandleBroadcast processes message broadcasting
func (h *Hub) HandleBroadcast(message *BroadcastMessage) {
	h.mutex.RLock()
	room, exists := h.rooms[message.RoomID]
	h.mutex.RUnlock()

	if exists {
		if message.SenderID != "" {
			// Broadcast to all except sender
			room.BroadcastExcept(message.Data, message.SenderID)
		} else {
			// Broadcast to all
			room.Broadcast(message.Data)
		}
	}
}

// Implement WebSocketHub interface methods
func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) BroadcastToRoom(roomID string, message []byte) {
	h.broadcast <- &BroadcastMessage{
		RoomID: roomID,
		Data:   message,
	}
}

func (h *Hub) BroadcastToRoomExcept(roomID string, message []byte, exceptUserID string) {
	h.broadcast <- &BroadcastMessage{
		RoomID:   roomID,
		Data:     message,
		SenderID: exceptUserID,
	}
}

func (h *Hub) GetRoomClients(roomID string) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if room, exists := h.rooms[roomID]; exists {
		return room.ClientCount()
	}
	return 0
}

func (h *Hub) GetRoom(roomID string) WebSocketRoom {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.rooms[roomID]
}

func (h *Hub) GetRoomCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.rooms)
}

func (h *Hub) BroadcastToUser(userID string, message []byte) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.client {
		if client.UserID == userID {
			select {
			case client.Send <- message:
				return nil
			case <-time.After(5 * time.Second):
				return errors.New("timeout sending notification to user")
			}
		}
	}

	return errors.New("user not connected")
}

func (h *Hub) BroadcastToAllUsers(message []byte) {
	h.mutex.RLock()
	clients := make([]*Client, 0, len(h.client))
	for client := range h.client {
		clients = append(clients, client)
	}
	h.mutex.RUnlock()
	log.Println(clients)
	log.Println(message)
	for _, client := range clients {
		select {
		case client.Send <- message:
			// Success
		case <-time.After(2 * time.Second):
			log.Printf("Timeout sending notification to user %s", client.UserID)
		}
	}
}

func (h *Hub) GetAllConnectedUsers() []string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	users := make([]string, 0, len(h.client))
	for client := range h.client {
		users = append(users, client.UserID)
	}
	return users
}
