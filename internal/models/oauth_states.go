package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

type OauthStates struct {
	ID        uuid.UUID `json:"id" db:"id,pk,auto"`
	State     string    `json:"state" db:"state"`
	Provider  string    `json:"provider" db:"provider"`
	Verifier  string    `json:"verifier" db:"verifier"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at,auto"`
}

func (o *OauthStates) GetID() interface{}   { return o.ID }
func (o *OauthStates) SetID(id interface{}) { o.ID = id.(uuid.UUID) }
func (o *OauthStates) TableName() string    { return "oauth_states" }
func (o *OauthStates) Validate() error {
	if o.State == "" {
		return errors.New("state is required")
	}
	if o.Provider == "" {
		return errors.New("provider is required")
	}
	return nil
}
