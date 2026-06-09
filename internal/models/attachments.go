package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

type Attachments struct {
	ID        uuid.UUID `db:"id,pk"`
	MessageID string    `db:"message_id"`
	RoomID    string    `db:"room_id"`
	URL       string    `db:"url"`
	FileName  string    `db:"file_name"`
	FileType  string    `db:"file_type"`
	FileSize  int64     `db:"file_size"`
	CreatedAt time.Time `db:"created_at,auto"`
}

func (a *Attachments) GetID() any        { return a.ID }
func (a *Attachments) SetID(id any)      { a.ID = id.(uuid.UUID) }
func (a *Attachments) TableName() string { return "message_attachments" }
func (a *Attachments) Validate() error {
	if a.FileType == "" {
		return errors.New("file type is required")
	}
	return nil
}
