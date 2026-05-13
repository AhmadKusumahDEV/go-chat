package websocket

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"
)

type WebsocketProcessor interface {
	QueueMessage(msg *ProcessMessage)
	Start()
	Stop()
	processMessage(msg *ProcessMessage)
	validateMessage(msg *BroadcastMessage) error
	worker(id int)
}

// MessageProcessorImpl implements MessageProcessor interface
type MessageProcessorImpl struct {
	workQueue chan *ProcessMessage
	workers   int
	hub       WebSocketHub
	shutdown  chan struct{}
	wg        sync.WaitGroup
}

type ProcessMessage struct {
	Client  *Client
	Message []byte
	UserID  string
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(workers int, hub WebSocketHub) WebsocketProcessor {
	return &MessageProcessorImpl{
		workQueue: make(chan *ProcessMessage, 1000),
		workers:   workers,
		hub:       hub,
		shutdown:  make(chan struct{}),
		wg:        sync.WaitGroup{},
	}
}

// Start begins the message processor workers
func (mp *MessageProcessorImpl) Start() {
	for i := 0; i < mp.workers; i++ {
		mp.wg.Add(1)
		go mp.worker(i)
	}
	log.Printf("Message processor started with %d workers", mp.workers)
}

// Stop gracefully stops the message processor
func (mp *MessageProcessorImpl) Stop() {
	close(mp.shutdown)
	mp.wg.Wait()
	log.Println("Message processor stopped")
}

// worker processes messages from the queue
func (mp *MessageProcessorImpl) worker(id int) {
	defer mp.wg.Done()

	for {
		select {
		case msg := <-mp.workQueue:
			mp.processMessage(msg)
		case <-mp.shutdown:
			return
		}
	}
}

// processMessage handles individual messages
func (mp *MessageProcessorImpl) processMessage(msg *ProcessMessage) {
	var broadcastMsg BroadcastMessage
	err := json.Unmarshal(msg.Message, &broadcastMsg)
	if err != nil {
		errorRes, _ := json.Marshal(map[string]string{"type": "error", "message": "Invalid JSON format payload"})
		msg.Client.SendMessage(errorRes)
		return
	}

	err = mp.validateMessage(&broadcastMsg)
	if err != nil {
		log.Println(err)
		errorRes, _ := json.Marshal(map[string]string{"type": "error", "message": err.Error()})
		msg.Client.SendMessage(errorRes)
		return
	}

	switch broadcastMsg.Type {
	case "message_group":
		mp.hub.BroadcastToRoom(broadcastMsg.RoomID, broadcastMsg.Data)
	case "join_room":
		mp.hub.SubscribeToRoom(broadcastMsg.RoomID, msg.Client)
	case "leave_room":
		mp.hub.UnsubscribeFromRoom(broadcastMsg.RoomID, msg.Client)
	}
}

// validateMessage checks if the message is valid
func (mp *MessageProcessorImpl) validateMessage(msg *BroadcastMessage) error {
	if len(msg.Data) > 30000 { // 30KB limit
		return errors.New("message payload exceeds 30KB limit")
	}
	return nil
}

// QueueMessage adds a message to the processing queue
func (mp *MessageProcessorImpl) QueueMessage(msg *ProcessMessage) {
	select {
	case mp.workQueue <- msg:

	case <-time.After(3 * time.Second):
		log.Println("Message queue full, dropping message after 3 second")
		// Message queued successfully
	default:
		log.Println("Message queue full, dropping message")
	}
}
