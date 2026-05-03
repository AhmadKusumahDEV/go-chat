package request

import (
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gofrs/uuid"
)

type CreateMessageRequest struct {
	RoomID string `json:"room_id" binding:"required,uuid"`

	Content string `json:"content"`

	Type string `json:"message_type" binding:"required,oneof=text image file"`

	ReplyTo *uuid.UUID `json:"reply_to,omitempty"`

	Attachments []models.Attachment `json:"attachments,omitempty"`
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

	return &models.Message{
		RoomID:      roomUUID,
		SenderID:    &senderUUID,
		Content:     r.Content,
		Type:        r.Type,
		ReplyTo:     r.ReplyTo,
		Attachments: r.Attachments,
	}, nil
}

type UpdateMessageRequest struct {
	Content string `json:"content" binding:"required,min=1"`
}
