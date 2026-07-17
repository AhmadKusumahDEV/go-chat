package request

// UpdateMemberRoleRequest jika nanti ada fitur ubah role.
type UpdateMemberRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member moderator"`
}

type AddMemberRequest struct {
	Members     []string `json:"members" binding:"required,min=1"` // Daftar UserID member
	Role        string   `json:"role" binding:"required,oneof=admin member moderator"`
	RoomID      string   `json:"room_id" binding:"required"`
	AddMemberBy string   `json:"added_by" binding:"required"`
}

type RemoveMemberRequest struct {
	Members []string `json:"members" binding:"required,min=1"`
}

type ManageMemberRequest struct {
	TargetUserID string `json:"target_user_id" binding:"required,uuid"`
}
