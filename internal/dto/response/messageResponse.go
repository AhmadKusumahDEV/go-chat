package response

import (
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gofrs/uuid"
)

type MessageResponse struct {
	ID          string    `json:"id"`
	RoomID      string    `json:"room_id"`
	SenderID    string    `json:"sender_id,omitempty"`
	SenderName  string    `json:"sender_name,omitempty"`
	Content     string    `json:"content"`
	MessageType string    `json:"message_type"`
	ReplyTo     string    `json:"reply_to,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Attachments []string  `json:"attachments,omitempty"`
}

func NewMessageResponse(msg *models.Message) *MessageResponse {
	resp := &MessageResponse{
		ID:          msg.ID.String(),
		RoomID:      msg.RoomID.String(),
		MessageType: msg.Type,
		Timestamp:   msg.Timestamp,
		Content:     msg.Content,
		Attachments: msg.Attachments,
	}

	if msg.SenderName != "" {
		resp.SenderName = msg.SenderName
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

// MessageListResponse wraps paginated message list with metadata
type MessageListResponse struct {
	Data       []*MessageResponse `json:"messages"`
	HasMore    bool               `json:"has_more"`
	NextCursor *string            `json:"next_cursor,omitempty"`
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

// {
//   "room_id": "uuid-room-1",
//   "type": "message_group",
//   "data": {
//     "title": "Room Nongkrong",
//     "body": "Ahmad: Hey everyone!",
//     "sender_id": "uuid-user-1",
//     "sender_name": "Ahmad Fatur",
//     "message_id": "uuid-msg-1"
//   }
// }

type NotificationResponse struct {
	RoomID     string `json:"room_id"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	SenderID   string `json:"sender_id"`
	SenderName string `json:"sender_name"`
	MessageID  string `json:"message_id"`
}
