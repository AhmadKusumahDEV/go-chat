package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

type Room struct {
	ID              uuid.UUID  `db:"id,pk,auto"` // pk=primary key, auto=auto increment
	Name            string     `db:"room_name"`
	Roomtype        string     `db:"room_type"`
	Description     string     `db:"description"`
	Isprivate       bool       `db:"is_private"`
	CreatedAt       time.Time  `db:"created_at,auto"`
	CreatedBy       uuid.UUID  `db:"created_by"`
	AvatarUrl       *string    `db:"-"`
	TargetUserID    *uuid.UUID `db:"-"`
	TargetUsername  *string    `db:"-"`
	TargetAvatarUrl *string    `db:"-"`
	LastMessage     *Message   `db:"-"`
}

func (u *Room) GetID() any        { return u.ID }
func (u *Room) SetID(id any)      { u.ID = id.(uuid.UUID) }
func (u *Room) TableName() string { return "rooms" }
func (u *Room) Validate() error {
	if u.Name == "" {
		return errors.New("room name is required")
	}
	return nil
}
