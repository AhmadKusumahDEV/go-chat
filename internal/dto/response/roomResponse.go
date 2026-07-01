package response

import "time"

// RoomResponse untuk response list rooms dengan last message
type RoomResponse struct {
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	Description     string               `json:"description,omitempty"`
	RoomType        string               `json:"room_type"`
	AvatarUrl       *string              `json:"avatar"`
	IsPrivate       bool                 `json:"is_private"`
	CreatedAt       time.Time            `json:"created_at"`
	TargetUserID    *string              `json:"target_user_id,omitempty"`
	TargetUsername  *string              `json:"target_username,omitempty"`
	TargetAvatarUrl *string              `json:"target_avatar_url,omitempty"`
	LastMessage     *LastMessageResponse `json:"last_message,omitempty"`
}

// LastMessageResponse untuk detail message terakhir di room
type LastMessageResponse struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	UserID      *string   `json:"user_id,omitempty"`
	UserName    *string   `json:"user_name,omitempty"`
	MessageType string    `json:"message_type"`
	Timestamp   time.Time `json:"timestamp"`
}

// RoomDetailResponse untuk menampilkan informasi lengkap sebuah room
type RoomDetailResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Avatar      string                 `json:"avatar"`
	RoomType    string                 `json:"room_type"`
	IsPrivate   bool                   `json:"is_private"`
	CreatedAt   time.Time              `json:"created_at"`
	CreatedBy   string                 `json:"created_by"`
	MemberCount int                    `json:"member_count"`
	Members     []MemberDetailResponse `json:"members"`
}

// MemberDetailResponse untuk menampilkan detail member dalam room
type MemberDetailResponse struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	AvatarUrl *string   `json:"avatar_url,omitempty"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
}

type EventNewRoomDirectResponse struct {
	RoomID string        `json:"room_id"`
	Type   string        `json:"type"`
	Data   *RoomResponse `json:"data"`
}
