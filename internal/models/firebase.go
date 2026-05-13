package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

type Firebase struct {
	ID             uuid.UUID `db:"id,pk,auto"`
	UserID         uuid.UUID `db:"user_id"`
	InstallationID string    `db:"installation_id"`
	FcmToken       string    `db:"fcm_token"`
	Platform       string    `db:"platform"`
	IsActive       bool      `db:"is_active"`
	LoggedOutAt    time.Time `db:"log_ged_at"`
}

func (u *Firebase) GetID() any        { return u.ID }
func (u *Firebase) SetID(id any)      { u.ID = id.(uuid.UUID) }
func (u *Firebase) TableName() string { return "user_devices" }
func (u *Firebase) Validate() error {
	if u.Platform == "" {
		return errors.New("room name is required")
	}
	return nil
}
