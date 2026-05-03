package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

type Attachment struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	FileSize int64  `json:"file_size"`
}

type Attachments []Attachment

type Message struct {
	ID          uuid.UUID   `db:"id,pk"`
	RoomID      uuid.UUID   `db:"room_id"`
	SenderID    *uuid.UUID  `db:"user_id"`
	ReplyTo     *uuid.UUID  `db:"reply_to"`
	Content     string      `db:"content"`
	Type        string      `db:"message_type"`
	Attachments Attachments `db:"attachments"`
	Timestamp   time.Time   `db:"timestamp,auto"`
}

func (a Attachments) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "[]", nil
	}
	return json.Marshal(a)
}

func (a *Attachments) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &a)
}

func (m *Message) GetID() any        { return m.ID }
func (m *Message) SetID(id any)      { m.ID = id.(uuid.UUID) }
func (m *Message) TableName() string { return "messages" }
func (m *Message) Validate() error {
	if m.Content == "" {
		return errors.New("message is required")
	}
	return nil
}
