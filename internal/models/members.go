package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

type Members struct {
	Roomid  uuid.UUID `db:"room_id"`
	Userid  uuid.UUID `db:"user_id"`
	AddedBy uuid.UUID `db:"added_by"`
	Role    string    `db:"role"`
	JoinAt  time.Time `db:"joined_at,auto"`
}

func (u *Members) GetID() any        { return u.Userid }
func (u *Members) SetID(id any)      { u.Userid = id.(uuid.UUID) }
func (u *Members) TableName() string { return "room_members" }
func (u *Members) Validate() error {
	if u.Roomid == uuid.Nil {
		return errors.New("room id is required")
	}
	return nil
}
