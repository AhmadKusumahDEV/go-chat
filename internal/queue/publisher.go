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
	PublishEventSocket(ctx context.Context, event interface{}) error
	PublishEventCall(ctx context.Context, event interface{}) error
	PublishEmailEvent(ctx context.Context, event *EmailEvent) error
}

type EmailEvent struct {
	Type     string `json:"type"`
	To       string `json:"to"`
	Subject  string `json:"subject"`
	OTP      string `json:"otp,omitempty"`
	Username string `json:"username,omitempty"`
	PlanName string `json:"plan_name,omitempty"`
	OrderID  string `json:"order_id,omitempty"`
	Amount   int64  `json:"amount,omitempty"`
}

type PaymentEvent struct {
	OrderId string `json:"order_id"`
	UserId  string `json:"user_id"`
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

// PublishEventCall implements [Publisher].
func (p *rabbitMQPublisher) PublishEventCall(ctx context.Context, event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		ctx,
		"chat.notifications", // exchange
		"call.event",         // routing key
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

func (p *rabbitMQPublisher) PublishTierUpgradeEvent(ctx context.Context, event *PaymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		ctx,
		"payment.exchange",     // exchange
		"payment.tier.upgrade", // routing key
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

// PublishEventSocket implements [Publisher].
func (p *rabbitMQPublisher) PublishEventSocket(ctx context.Context, event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		ctx,
		"chat.notifications", // exchange
		"socket.event",       // routing key
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

func (p *rabbitMQPublisher) PublishEmailEvent(ctx context.Context, event *EmailEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	routingKey := "email." + event.Type

	return p.channel.PublishWithContext(
		ctx,
		"email.notifications",
		routingKey,
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
