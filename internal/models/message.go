package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

type Message struct {
	ID          uuid.UUID  `db:"id,pk"`
	RoomID      uuid.UUID  `db:"room_id"`
	SenderID    *uuid.UUID `db:"user_id"`
	SenderName  string     `db:"-"` // Added for joined query
	ReplyTo     *uuid.UUID `db:"reply_to"`
	Content     string     `db:"content"`
	Type        string     `db:"message_type"`
	Timestamp   time.Time  `db:"timestamp,auto"`
	Attachments []string   `db:"-"`
}

func (m *Message) GetID() any        { return m.ID }
func (m *Message) SetID(id any)      { m.ID = id.(uuid.UUID) }
func (m *Message) TableName() string { return "messages" }
func (m *Message) Validate() error {
	// Allow empty content for image/file/mixed message types
	if m.Content == "" {
		switch m.Type {
		case "image", "file", "mixed":
			return nil
		default:
			return errors.New("message content is required for text messages")
		}
	}
	return nil
}
