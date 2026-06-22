package request

import (
	"fmt"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gofrs/uuid"
)

type CreateRoomRequest struct {
	Name        string   `json:"name" validate:"required,min=3,max=100"`
	Description string   `json:"description" validate:"max=500"`
	RoomType    string   `json:"room_type" validate:"required,oneof=group direct channel"`
	IsPrivate   *bool    `json:"is_private" validate:"required"` // Menggunakan pointer *bool untuk membedakan antara false dan tidak diisi.
	Members     []string `json:"members" validate:"omitempty"`   // Daftar UserID member awal (opsional)
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
		AvatarUrl:   &defaultAvatar,
		CreatedBy:   strToUuid,
		CreatedAt:   currentDate,
	}, nil
}

type CreateDirectRoomRequest struct {
	TargetUserId string `json:"target_user_id" validate:"required,uuid"`
	MessageType  string `json:"message_type" validate:"required,oneof=text image file"`
	Content      string `json:"content" validate:"required"`
}

type UpdateRoomRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=3,max=100"`
	Description *string `json:"description" validate:"omitempty,max=500"`
}

type GetRoomByName struct {
	Name string `json:"name" validate:"required,min=3,max=100"`
}
