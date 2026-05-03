package request

import (
	"log"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gofrs/uuid"
)

// UpdateMemberRoleRequest jika nanti ada fitur ubah role.
type UpdateMemberRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=admin member moderator"`
}

type AddMember struct {
	UserID      string `json:"user_id" validate:"required" binding:"uuid"`
	Role        string `json:"role" validate:"required,oneof=admin member moderator"`
	RoomID      string `json:"room_id" validate:"required" binding:"uuid"`
	AddMemberBy string `json:"added_by" validate:"required" binding:"uuid"`
}

func (m *AddMember) ToModel() (*models.Members, error) {
	var err error

	parse := func(s string) uuid.UUID {
		if err != nil {
			return uuid.Nil
		}

		id, parseErr := uuid.FromString(s)
		if parseErr != nil {
			err = parseErr
			return uuid.Nil
		}
		return id
	}

	roomID := parse(m.RoomID)
	userID := parse(m.UserID)
	addedBy := parse(m.AddMemberBy)

	// Cek error cukup sekali di akhir
	if err != nil {
		log.Printf("[ToModel] Failed to parse UUID: %v", err)
		return nil, err
	}

	return &models.Members{
		Roomid:  roomID,
		Userid:  userID,
		AddedBy: addedBy,
		Role:    m.Role,
	}, nil
}
