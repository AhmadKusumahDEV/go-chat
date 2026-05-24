package response

// MemberResponse representasi member dalam sebuah room.
type MemberResponse struct {
	UserName string  `json:"user_name,omitempty"`
	Role     string  `json:"role"`
	UserID   string  `json:"user_id"`
	Avatar   *string `json:"avatar_url"`
}
