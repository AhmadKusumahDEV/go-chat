package request

// UpdateMemberRoleRequest jika nanti ada fitur ubah role.
type UpdateMemberRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=admin member moderator"`
}

type AddMemberRequest struct {
	Members     []string `json:"members" validate:"required,min=1"` // Daftar UserID member
	Role        string   `json:"role" validate:"required,oneof=admin member moderator"`
	RoomID      string   `json:"room_id" validate:"required"`
	AddMemberBy string   `json:"added_by" validate:"required"`
}
