package response

import "time"

// RoomResponse adalah representasi standar room saat dikirim ke client.
type RoomResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"` // omitempty jika ingin menyembunyikan field kosong
	RoomType    string    `json:"room_type"`
	IsPrivate   bool      `json:"is_private"`
	CreatedAt   time.Time `json:"created_at"`
}

// RoomDetailResponse bisa digunakan jika ingin menampilkan data lebih detail,
// misalnya termasuk daftar member (walaupun sebaiknya di-fetch terpisah jika datanya banyak).
type RoomDetailResponse struct {
	RoomResponse
	MemberCount int `json:"member_count"`
	// Members []MemberResponse `json:"members,omitempty"` // Hati-hati, bisa sangat besar untuk grup besar.
}
