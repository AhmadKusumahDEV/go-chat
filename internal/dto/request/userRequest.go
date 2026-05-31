package request

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest digunakan saat user masuk
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UpdateProfileRequest (Opsional: Jika nanti user ingin ganti username)
type UpdateProfileRequest struct {
	Username string `json:"username" binding:"omitempty,min=3,max=50"`
	About    string `json:"about" binding:"omitempty,max=500"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type FcmRequest struct {
	Platform     string `json:"platform" binding:"required"`
	FcmToken     string `json:"fcm_token" binding:"required"`
	Installation string `json:"installation_id" binding:"required"`
}

// LogoutRequest for logging out from a specific device or all devices
type LogoutRequest struct {
	FcmToken string `json:"fcm_token" binding:"required"`
}
