package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/queue"
)

// MessageSender is a local interface to avoid importing services (which would cause a cycle).
// services.MessageService satisfies this interface.
type MessageSender interface {
	SendMessage(ctx context.Context, req *request.CreateMessageRequest, senderID string) (*response.MessageResponse, error)
	GetMessageByID(ctx context.Context, messageID string) (*response.MessageResponse, error)
}

type WebsocketProcessor interface {
	QueueMessage(msg *ProcessMessage)
	Start()
	Stop()
	processMessage(msg *ProcessMessage)
	validateMessage(msg *BroadcastMessage) error
	worker(id int)
	GetMessageID(ctx context.Context, messageID string) (*response.MessageResponse, error)
}

// MessageProcessorImpl implements MessageProcessor interface
type MessageProcessorImpl struct {
	workQueue      chan *ProcessMessage
	workers        int
	hub            WebSocketHub
	messageService MessageSender
	publisher      queue.Publisher
	shutdown       chan struct{}
	wg             sync.WaitGroup
}

func (mp *MessageProcessorImpl) GetMessageID(ctx context.Context, messageID string) (*response.MessageResponse, error) {
	return mp.messageService.GetMessageByID(ctx, messageID)
}

type ProcessMessage struct {
	Client  *Client
	Message []byte
	UserID  string
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(workers int, hub WebSocketHub, messageService MessageSender, publisher queue.Publisher) WebsocketProcessor {
	return &MessageProcessorImpl{
		workQueue:      make(chan *ProcessMessage, 1000),
		workers:        workers,
		hub:            hub,
		messageService: messageService,
		publisher:      publisher,
		shutdown:       make(chan struct{}),
		wg:             sync.WaitGroup{},
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

func (mp *MessageProcessorImpl) processMessage(msg *ProcessMessage) {
	var broadcastMsg BroadcastMessage
	err := json.Unmarshal(msg.Message, &broadcastMsg)
	if err != nil {
		errorRes, _ := json.Marshal(map[string]string{"message_type": "error", "message": "Invalid JSON format payload"})
		msg.Client.SendMessage(errorRes)
		return
	}
	log.Println("log brodcast msg: ", broadcastMsg)

	err = mp.validateMessage(&broadcastMsg)
	if err != nil {
		log.Println(err)
		errorRes, _ := json.Marshal(map[string]string{"message_type": "error", "message": err.Error()})
		msg.Client.SendMessage(errorRes)
		return
	}

	switch broadcastMsg.Type {
	case "message_group":
		var createMsgReq request.CreateMessageRequest
		if err := json.Unmarshal(broadcastMsg.Data, &createMsgReq); err != nil {
			log.Println("Invalid message payload", err)
			return
		}
		log.Println("log create msg: ", createMsgReq)

		ctx := context.Background()
		savedMsg, err := mp.messageService.SendMessage(ctx, &createMsgReq, msg.UserID)
		if err != nil {
			log.Println("Failed to save message to DB", err)
			errorRes, _ := json.Marshal(map[string]string{"type": "error", "message": err.Error()})
			msg.Client.SendMessage(errorRes)
			return
		}

		mp.hub.BroadcastToRoomExcept(broadcastMsg.RoomID, broadcastMsg.Data, msg.UserID)

		// Publish notification to RabbitMQ with user IDs for FCM
		notifEvent := queue.NotificationEvent{
			Type:       "message_group",
			MessageID:  savedMsg.ID,
			RoomID:     savedMsg.RoomID,
			SenderID:   msg.UserID,
			SenderName: savedMsg.SenderName,
			Title:      "New Message",
			Body:       savedMsg.Content,
		}
		log.Println("publish to user")
		if err := mp.publisher.PublishNotification(ctx, notifEvent); err != nil {
			log.Println("Failed to publish notification to RabbitMQ", err)
		}

	case "join_room":
		log.Println("solidd join")
	}
}

// validateMessage checks if the message is valid
func (mp *MessageProcessorImpl) validateMessage(msg *BroadcastMessage) error {
	if len(msg.Data) > 30000 {
		return errors.New("message payload exceeds 30KB limit")
	}
	return nil
}

// QueueMessage adds a message to the processing queue
func (mp *MessageProcessorImpl) QueueMessage(msg *ProcessMessage) {
	select {
	case mp.workQueue <- msg:
		// Message queued successfully
	case <-time.After(3 * time.Second):
		log.Println("Message queue full, dropping message after 3 seconds")
	}
}
