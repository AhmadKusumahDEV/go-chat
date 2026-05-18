package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher interface {
	PublishMessage(ctx context.Context, event *MessageEvent) error
	PublishNotification(ctx context.Context, event interface{}) error
}

type MessageEvent struct {
	Type    string          `json:"type"`
	Message *models.Message `json:"message"`
}

type NotificationEvent struct {
	Type       string   `json:"type"`
	MessageID  string   `json:"messageId"`
	RoomID     string   `json:"roomId"`
	RoomName   string   `json:"room_name"`
	SenderID   string   `json:"senderId"`
	SenderName string   `json:"sender_name"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	UserIDs    []string `json:"userIds"`
}

// Implementation
type rabbitMQPublisher struct {
	channel *amqp.Channel
}

func NewRabbitMQPublisher(channel *amqp.Channel) Publisher {
	return &rabbitMQPublisher{channel: channel}
}

func (p *rabbitMQPublisher) PublishMessage(ctx context.Context, event *MessageEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		ctx,
		"chat.messages", // exchange
		"",              // routing key
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
}

func (p *rabbitMQPublisher) PublishNotification(ctx context.Context, event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		ctx,
		"chat.notifications",
		"notification.message.new",
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
}
