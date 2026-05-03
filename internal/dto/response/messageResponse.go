package response

import (
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gofrs/uuid"
)

type MessageResponse struct {
	ID          string              `json:"id"`
	RoomID      string              `json:"room_id"`
	SenderID    string              `json:"sender_id,omitempty"`
	Content     string              `json:"content"`
	MessageType string              `json:"message_type"`
	ReplyTo     string              `json:"reply_to,omitempty"`
	Attachments []models.Attachment `json:"attachments,omitempty"`
	Timestamp   time.Time           `json:"timestamp"`
}

func NewMessageResponse(msg *models.Message) *MessageResponse {
	resp := &MessageResponse{
		ID:          msg.ID.String(),
		RoomID:      msg.RoomID.String(),
		Content:     msg.Content,
		MessageType: msg.Type,
		Attachments: msg.Attachments,
		Timestamp:   msg.Timestamp,
	}

	if msg.SenderID != nil && *msg.SenderID != uuid.Nil {
		resp.SenderID = msg.SenderID.String()
	}

	if msg.ReplyTo != nil && *msg.ReplyTo != uuid.Nil {
		resp.ReplyTo = msg.ReplyTo.String()
	}

	return resp
}

func NewMessageResponses(msgs []*models.Message) []*MessageResponse {
	var responses []*MessageResponse
	for _, msg := range msgs {
		responses = append(responses, NewMessageResponse(msg))
	}
	return responses
}

type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	Error        string `json:"error,omitempty"`
}
