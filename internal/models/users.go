package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

type Users struct {
	ID           uuid.UUID `json:"id" db:"id,pk,auto"` // pk=primary key, auto=auto increment
	Email        string    `json:"email" db:"email"`
	Password     string    `json:"-" db:"password_hash"`
	Username     string    `json:"username" db:"username"`
	CreatedAt    time.Time `json:"created_at" db:"created_at,auto"`
	AvatarUrl    string    `json:"avatar_url" db:"avatar_url"`
	ProviderName string    `json:"provider_name" db:"provider_name"`
	ProviderID   string    `json:"provider_id" db:"provider_id"`
}

func (u *Users) GetID() interface{}   { return u.ID }
func (u *Users) SetID(id interface{}) { u.ID = id.(uuid.UUID) }
func (u *Users) TableName() string    { return "users" }
func (u *Users) Validate() error {
	if u.Email == "" {
		return errors.New("email is required")
	}
	return nil
}
