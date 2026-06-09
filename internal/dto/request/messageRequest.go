package request

import (
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gofrs/uuid"
)

type CreateMessageRequest struct {
	MessageID string `json:"message_id"`

	RoomID string `json:"room_id" binding:"required,uuid"`

	Content string `json:"content"`

	Type string `json:"message_type" binding:"required,oneof=text image file mixed"`

	SenderID string `json:"sender_id,omitempty"`

	SenderName string `json:"sender_name,omitempty"`

	ReplyTo *uuid.UUID `json:"reply_to,omitempty"`
}

func (r *CreateMessageRequest) ToModel(senderID string) (*models.Message, error) {
	roomUUID, err := uuid.FromString(r.RoomID)
	if err != nil {
		return nil, err
	}

	senderUUID, err := uuid.FromString(senderID)
	if err != nil {
		return nil, err
	}

	if r.MessageID != "" {
		messageID, err := uuid.FromString(r.MessageID)
		if err != nil {
			return nil, err
		}

		return &models.Message{
			ID:         messageID,
			RoomID:     roomUUID,
			SenderID:   &senderUUID,
			SenderName: r.SenderName, // Include sender name from request
			Content:    r.Content,
			Type:       r.Type,
			ReplyTo:    r.ReplyTo,
		}, nil
	}

	return &models.Message{
		RoomID:     roomUUID,
		SenderID:   &senderUUID,
		SenderName: r.SenderName, // Include sender name from request
		Content:    r.Content,
		Type:       r.Type,
		ReplyTo:    r.ReplyTo,
	}, nil
}

type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required,min=1"`
}
