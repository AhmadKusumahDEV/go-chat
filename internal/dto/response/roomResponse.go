package response

import "time"

// RoomResponse untuk response list rooms dengan last message
type RoomResponse struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	RoomType    string               `json:"room_type"`
	IsPrivate   bool                 `json:"is_private"`
	CreatedAt   time.Time            `json:"created_at"`
	LastMessage *LastMessageResponse `json:"last_message,omitempty"` // ✅ Tambahkan last message
}

// LastMessageResponse untuk detail message terakhir di room
type LastMessageResponse struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	UserID      *string   `json:"user_id,omitempty"` // Nullable karena ON DELETE SET NULL
	MessageType string    `json:"message_type"`
	Timestamp   time.Time `json:"timestamp"`
}

// RoomDetailResponse bisa digunakan jika ingin menampilkan data lebih detail,
// misalnya termasuk daftar member (walaupun sebaiknya di-fetch terpisah jika datanya banyak).
type RoomDetailResponse struct {
	RoomResponse
	MemberCount int `json:"member_count"`
	// Members []MemberResponse `json:"members,omitempty"` // Hati-hati, bisa sangat besar untuk grup besar.
}
