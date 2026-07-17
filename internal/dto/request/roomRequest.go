package request

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gofrs/uuid"
)

type CreateRoomRequest struct {
	Name        string   `json:"name" binding:"required,min=3,max=100"`
	Description string   `json:"description" binding:"max=500"`
	RoomType    string   `json:"room_type" binding:"required,oneof=group direct channel"`
	IsPrivate   *bool    `json:"is_private" binding:"required"` // Menggunakan pointer *bool untuk membedakan antara false dan tidak diisi.
	Members     []string `json:"members" binding:"omitempty"`   // Daftar UserID member awal (opsional)
}

func (r *CreateRoomRequest) ToModel(id string) (*models.Room, error) {
	currentDate := time.Now()
	strToUuid, err := uuid.FromString(id)
	defaultAvatar := fmt.Sprintf("https://api.dicebear.com/10.x/initials/png?seed=%s", r.Name)

	if err != nil {
		return nil, err
	}

	return &models.Room{
		Name:        r.Name,
		Description: r.Description,
		Roomtype:    r.RoomType,
		Isprivate:   *r.IsPrivate,
		AvatarUrl: sql.NullString{
			String: defaultAvatar,
			Valid:  true,
		},
		CreatedBy: strToUuid,
		CreatedAt: currentDate,
	}, nil
}

type CreateDirectRoomRequest struct {
	TargetUserId string `json:"target_user_id" binding:"required,uuid"`
	MessageType  string `json:"message_type" binding:"required,oneof=text image file"`
	Content      string `json:"content" binding:"required"`
}

type UpdateRoomRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=3,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
}

type GetRoomByName struct {
	Name string `json:"name" binding:"required,min=3,max=100"`
}

type MetaFIlePicture struct {
	RoomId      string `json:"room_id" binding:"required,uuid"`
	ContentType string `json:"content_type" binding:"required"`
	NameFile    string `json:"name_file" binding:"required"`
}
