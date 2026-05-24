package models

import (
	"time"

	"github.com/gofrs/uuid"
)

type MemberComposite struct {
	Username string `db:"username"`
	Role     string `db:"role"`
	UserID   string `db:"user_id"`
	Avatar   string `db:"avatar_url"`
}

type MemberDetail struct {
	UserID    uuid.UUID `db:"user_id"`
	Username  string    `db:"username"`
	Email     string    `db:"email"`
	AvatarUrl *string   `db:"avatar_url"`
	Role      string    `db:"role"`
	JoinedAt  time.Time `db:"joined_at"`
}

type RoomDetail struct {
	ID          uuid.UUID      `db:"id"`
	Name        string         `db:"name"`
	Description string         `db:"description"`
	RoomType    string         `db:"room_type"`
	IsPrivate   bool           `db:"is_private"`
	CreatedAt   time.Time      `db:"created_at"`
	CreatedBy   uuid.UUID      `db:"created_by"`
	MemberCount int            `db:"member_count"`
	Members     []MemberDetail `db:"-"`
}
