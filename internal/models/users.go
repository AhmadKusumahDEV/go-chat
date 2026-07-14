package models

import (
	"database/sql"
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

type Users struct {
	ID           uuid.UUID      `json:"id" db:"id,pk,auto"` // pk=primary key, auto=auto increment
	Email        string         `json:"email" db:"email"`
	Password     string         `json:"-" db:"password_hash"`
	Username     string         `json:"username" db:"username"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at,auto"`
	AvatarUrl    sql.NullString `json:"avatar_url" db:"avatar_url"`
	About        sql.NullString `json:"about" db:"about"`
	ProviderName sql.NullString `json:"provider_name" db:"provider_name"`
	ProviderID   sql.NullString `json:"provider_id" db:"provider_id"`
	Tier         sql.NullString `json:"tier" db:"tier"`
	Verify       sql.NullBool   `json:"verify" db:"verify"`
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
