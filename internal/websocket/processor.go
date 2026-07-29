package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/queue"
	"github.com/gofrs/uuid"
	"github.com/rabbitmq/amqp091-go"
)

// MessageSender is a local interface to avoid importing services (which would cause a cycle).
// services.MessageService satisfies this interface.
type MessageSender interface {
	SendMessage(ctx context.Context, req *request.CreateMessageRequest, senderID string) (*response.MessageResponse, error)
	GetMessageByID(ctx context.Context, messageID string) (*response.MessageResponse, error)
}

type GetRoomSpecifice interface {
	GetSpecificRoomByUserID(ctx context.Context, userID string, roomID string) (*response.RoomResponse, error)
}

type WebsocketProcessor interface {
	QueueMessage(msg *ProcessMessage)
	Start()
	Stop()
	processMessage(msg *ProcessMessage)
	validateMessage(msg *BroadcastMessage) error
	worker(id int, data <-chan amqp091.Delivery)
	GetMessageID(ctx context.Context, messageID string) (*response.MessageResponse, error)
	NewRoomEvent(ctx context.Context, client *Client, targetUserId string, roomID string)
}

// MessageProcessorImpl implements MessageProcessor interface
type MessageProcessorImpl struct {
	workQueue      chan *ProcessMessage
	workers        int
	hub            WebSocketHub
	messageService MessageSender
	userServices   UserSpecifiedRequirements
	roomService    GetRoomSpecifice
	publisher      queue.Publisher
	channel        *amqp091.Channel
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
func NewMessageProcessor(workers int, hub WebSocketHub, messageService MessageSender, room GetRoomSpecifice, publisher queue.Publisher, channel *amqp091.Channel, userServices UserSpecifiedRequirements) WebsocketProcessor {
	return &MessageProcessorImpl{
		workQueue:      make(chan *ProcessMessage, 1000),
		workers:        workers,
		hub:            hub,
		messageService: messageService,
		userServices:   userServices,
		roomService:    room,
		publisher:      publisher,
		channel:        channel,
		shutdown:       make(chan struct{}),
		wg:             sync.WaitGroup{},
	}
}

// Start begins the message processor workers
func (mp *MessageProcessorImpl) Start() {
	events, err := mp.channel.Consume(
		"socket-notifications",
		"socket-events",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Println("error on consume socket event")

		return
	}

	for i := 0; i < mp.workers; i++ {
		mp.wg.Add(1)
		go mp.worker(i, events)
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
func (mp *MessageProcessorImpl) worker(id int, data <-chan amqp091.Delivery) {
	defer mp.wg.Done()

	for {
		select {
		case msg := <-mp.workQueue:
			mp.processMessage(msg)
		case msg := <-data:
			log.Println(msg)
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

	switch {
	case broadcastMsg.Type == "message_group":
		ctx := context.Background()
		var createMsgReq request.CreateMessageRequest
		if err := json.Unmarshal(broadcastMsg.Data, &createMsgReq); err != nil {
			log.Println("Invalid message payload", err)
			return
		}
		log.Println("log create msg: ", createMsgReq)

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

	case broadcastMsg.Type == "join_room_event":
		mp.hub.SubscribeToRoom(broadcastMsg.RoomID, msg.Client)

	case broadcastMsg.Type == "create_room_event":
		var userID request.EventNewDirectRoom
		err := json.Unmarshal(broadcastMsg.Data, &userID)
		if err != nil {
			log.Println("failed get value uesr target id")
			return
		}
		mp.NewRoomEvent(context.Background(), msg.Client, userID.TargetUesrID, broadcastMsg.RoomID)

	case strings.HasPrefix(broadcastMsg.Type, "call"):
		ctx := context.Background()

		var targetUserID string
		var forwardData any

		switch {
		case strings.HasSuffix(broadcastMsg.Type, ".offer"):
			var payload CallerSendOffer
			if err := json.Unmarshal(broadcastMsg.Data, &payload); err != nil {
				log.Println("error marshal offer payload:", err)
				return
			}

			senderId, err := uuid.FromString(msg.UserID)
			if err != nil {
				log.Println("invalid sender user id:", err)
				return
			}

			userInfo, err := mp.userServices.GetDetailUser(ctx, senderId)
			if err != nil {
				log.Println("failed get user detail:", err)
				return
			}

			offerData := CallerforwardOffer{
				CallId:       payload.CallId,
				TargetUserId: payload.TargetUserId,
				CallerName:   userInfo.Username,
				CallerId:     msg.UserID,
				Avatar:       userInfo.AvatarUrl,
				Sdp:          payload.Sdp,
				Mode:         payload.Mode,
			}

			if err := mp.publisher.PublishEventCall(ctx, offerData); err != nil {
				log.Println("failed publish event call to rabbitmq:", err)
			}

			targetUserID = payload.TargetUserId
			forwardData = offerData

		case strings.HasSuffix(broadcastMsg.Type, ".answer"):
			var payload CallSendAnswer
			if err := json.Unmarshal(broadcastMsg.Data, &payload); err != nil {
				log.Println("error marshal answer payload:", err)
				return
			}
			targetUserID = payload.TargetUserId
			forwardData = CallForwardAnswer{
				CallId: payload.CallId,
				Sdp:    payload.Sdp,
			}

		case strings.HasSuffix(broadcastMsg.Type, ".ice"):
			var payload CallSendIce
			if err := json.Unmarshal(broadcastMsg.Data, &payload); err != nil {
				log.Println("error marshal ice payload:", err)
				return
			}
			targetUserID = payload.TargetUserId
			forwardData = CallForwardIce{
				CallId:    payload.CallId,
				Candidate: payload.Candidate,
			}

		case strings.HasSuffix(broadcastMsg.Type, ".mute"):
			var payload CallSendMute
			if err := json.Unmarshal(broadcastMsg.Data, &payload); err != nil {
				log.Println("error marshal mute payload:", err)
				return
			}
			targetUserID = payload.TargetUserId
			forwardData = CallForwardMute{
				CallId: payload.CallId,
				Muted:  payload.Muted,
			}

		case strings.HasSuffix(broadcastMsg.Type, ".hangup"):
			var payload CallSendHangup
			if err := json.Unmarshal(broadcastMsg.Data, &payload); err != nil {
				log.Println("error marshal hangup payload:", err)
				return
			}
			targetUserID = payload.TargetUserId
			forwardData = CallForwardHangup{
				CallId: payload.CallId,
			}

		default:
			log.Println("Unknown call event type:", broadcastMsg.Type)
			return
		}

		if targetUserID != "" && forwardData != nil {
			forwardHeaderBytes, err := json.Marshal(FormatFowardEvent{
				Type: broadcastMsg.Type,
				Data: forwardData,
			})
			if err != nil {
				log.Println("error marshal forward header:", err)
				return
			}

			if err := mp.hub.BroadcastToUser(targetUserID, forwardHeaderBytes); err != nil {
				log.Println("error broadcast to user:", err)
				return
			}
		}

	default:
		fmt.Println("Tipe pesan tidak dikenali")
	}
}

func (mp *MessageProcessorImpl) NewRoomEvent(ctx context.Context, client *Client, targetUserId string, roomID string) {
	lastMessage, err := mp.roomService.GetSpecificRoomByUserID(ctx, targetUserId, roomID)
	if err != nil {
		log.Println(err)
		return
	}

	dataRoom := response.EventNewRoomDirectResponse{
		RoomID: roomID,
		Type:   "new_room",
		Data:   lastMessage,
	}

	message, err := json.Marshal(dataRoom)
	if err != nil {
		log.Println(err)
		return
	}

	notifEvent := queue.NotificationEvent{
		Type:       "message_Direct",
		MessageID:  lastMessage.LastMessage.ID,
		RoomID:     lastMessage.ID,
		SenderID:   client.UserID,
		SenderName: *lastMessage.TargetUsername,
		Title:      "New Message",
		Body:       lastMessage.LastMessage.Content,
	}

	exists := mp.hub.handleRegisterNewDirectRoom(client, targetUserId, roomID)
	if exists == nil {
		log.Println("publish to user")
		if err := mp.publisher.PublishNotification(ctx, notifEvent); err != nil {
			log.Println("Failed to publish notification to RabbitMQ", err)
		}
		return
	}

	log.Println("publish to user")
	if err := mp.publisher.PublishNotification(ctx, notifEvent); err != nil {
		log.Println("Failed to publish notification to RabbitMQ", err)
	}

	mp.hub.BroadcastToRoomExcept(roomID, message, client.UserID)
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
