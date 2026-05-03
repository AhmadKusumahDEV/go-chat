package request

import (
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gofrs/uuid"
)

type CreateRoomRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=100"`
	Description string `json:"description" validate:"max=500"`
	RoomType    string `json:"room_type" validate:"required,oneof=group direct channel"`
	IsPrivate   *bool  `json:"is_private" validate:"required"` // Menggunakan pointer *bool untuk membedakan antara false dan tidak diisi.
}

func (r *CreateRoomRequest) ToModel(id string) (*models.Room, error) {
	strToUuid, err := uuid.FromString(id)

	if err != nil {
		return nil, err
	}

	return &models.Room{
		Name:        r.Name,
		Description: r.Description,
		Roomtype:    r.RoomType,
		Isprivate:   *r.IsPrivate,
		CreatedBy:   strToUuid,
	}, nil
}

type UpdateRoomRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=3,max=100"`
	Description *string `json:"description" validate:"omitempty,max=500"`
}

type GetRoomByName struct {
	Name string `json:"name" validate:"required,min=3,max=100"`
}
