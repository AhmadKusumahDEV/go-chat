package response

// MemberResponse representasi member dalam sebuah room.
type MemberResponse struct {
	UserName string `json:"user_name,omitempty"` // Seringkali client butuh nama user juga, perlu join table users.
	Role     string `json:"role"`
}
