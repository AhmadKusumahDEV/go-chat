package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/helpers"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/AhmadKusumahDEV/go-chat/pkg/storage"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
)

type UsersServices interface {
	LoginUser(ctx context.Context, req *request.LoginRequest) (*response.JwtReponse, error)
	RegisterUser(ctx context.Context, req *request.RegisterRequest) error
	RefreshUser(ctx context.Context, refreshToken string) (*response.JwtReponse, error)
	GetAllUser(ctx context.Context) ([]*response.UserResponse, error)
	GetDetailUser(ctx context.Context, userId uuid.UUID) (*response.UserResponse, error)
	GetUserByID(ctx context.Context, userId uuid.UUID) (*response.UserResponse, error)
	StoreFirebaseToken(ctx context.Context, fcm *request.FcmRequest, userId uuid.UUID) error
	LogoutUser(ctx context.Context, userId uuid.UUID, installationID string) (int, error)
	VerifyEmail(ctx context.Context, userId uuid.UUID) error
	VerifyOtp(ctx context.Context, userId, otp string) error
	UpdatedUserInfo(ctx context.Context, userId uuid.UUID, req *request.UpdateProfileRequest) error
	UpdateAvatar(ctx context.Context, userID string, reader io.Reader, size int64, contentType, objectName string) (string, error)
}

type UsersServivesImpl struct {
	userRepository     repository.RepositoryUser
	firebaseRepository repository.RepositoryFirebase
	minioS3            storage.ObjectStorage
	rds                *redis.Client
	httpClient         *http.Client
	jwtConfig          config.JwtConfig
	espConfig          config.Esp
}

// VerifyOtp implements [UsersServices].
func (u *UsersServivesImpl) VerifyOtp(ctx context.Context, userId, otp string) error {
	key := fmt.Sprintf("otp:verify:%s", otp)

	val, err := u.rds.GetDel(ctx, key).Result()
	if err != nil || val != otp {
		return errors.New("invalid otp")
	}

	err = u.userRepository.UpdateVerifyUser(ctx, userId)
	if err != nil {
		log.Println(err)
		return errors.New("tidak dapat melakukan updated user data tidak valid")
	}
	return nil
}

// VerifyEmail implements [UsersServices].
func (u *UsersServivesImpl) VerifyEmail(ctx context.Context, userId uuid.UUID) error {
	user, err := u.userRepository.FindByID(ctx, userId)
	if err != nil {
		log.Println(err)
		return errors.New("user not found")
	}

	if user.Verify {
		return errors.New("user already verify")
	}

	otp, err := GenerateOtp(6)
	if err != nil {
		otp = models.RandomString(6)
	}

	payload := BuildMailOtp(ctx, user.Email, otp)

	key := fmt.Sprintf("otp:verify:%s", otp)

	req, _ := http.NewRequest("POST", u.espConfig.Brevo.Url, bytes.NewBuffer(payload))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", u.espConfig.Brevo.ApiKey)

	resp, err := u.httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return errors.New("tidak dapat melakukan kirim email")
	}

	err = u.rds.Set(ctx, key, otp, 5*time.Minute).Err()
	if err != nil {
		log.Println(err)
		return errors.New("tidak dapat melakukan kirim email")
	}
	defer resp.Body.Close()

	return nil
}

// LoginUser implements UsersServices.
func (u *UsersServivesImpl) LoginUser(ctx context.Context, req *request.LoginRequest) (*response.JwtReponse, error) {
	keyLock := fmt.Sprintf("login:lock:%s", req.Email)

	lockStat, err := u.rds.SetNX(ctx, keyLock, "1", 2*time.Second).Result()
	if err != nil || !lockStat {
		log.Println(err)
		return nil, errors.New("sedang melakukan proses login mohon meunggu")
	}

	users, err := u.userRepository.FindByEmail(ctx, req.Email)

	if err != nil {
		return nil, errors.New("user not found")
	}

	userinfo := models.JwtUsersInfo{
		UserID:   users.ID.String(),
		Email:    users.Email,
		Username: users.Username,
	}

	if !helpers.ValidatePassword(req.Password, users.Password) {
		return nil, errors.New("password not valid")
	}

	accessToken, refreshToken, err := helpers.GenerateAuthTokens(
		userinfo,
		u.jwtConfig.SecretKeyAccess,
		u.jwtConfig.SecretKeyrefresh,
		u.jwtConfig.AccessTokenExpiration,
		u.jwtConfig.RefreshTokenExpiration,
	)

	if err != nil {
		return nil, err
	}

	return &response.JwtReponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Userinfo:     &userinfo,
	}, nil
}

// RefreshUser implements UsersServices.
// Accepts refresh token as string parameter
func (u *UsersServivesImpl) RefreshUser(ctx context.Context, refreshToken string) (*response.JwtReponse, error) {
	claims, err := helpers.GetUserJWT(refreshToken, u.jwtConfig.SecretKeyrefresh)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}

	userInfoVal, ok := claims["user_info"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid claims format")
	}

	userID := fmt.Sprintf("%v", userInfoVal["user_id"])
	email := fmt.Sprintf("%v", userInfoVal["email"])
	username := fmt.Sprintf("%v", userInfoVal["username"])

	userinfo := models.JwtUsersInfo{
		UserID:   userID,
		Username: username,
		Email:    email,
	}

	accessToken, refreshToken, err := helpers.GenerateAuthTokens(
		userinfo,
		u.jwtConfig.SecretKeyAccess,
		u.jwtConfig.SecretKeyrefresh,
		u.jwtConfig.AccessTokenExpiration,
		u.jwtConfig.RefreshTokenExpiration,
	)

	if err != nil {
		return nil, err
	}

	return &response.JwtReponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Userinfo:     &userinfo,
	}, nil
}

// RegisterUser implements UsersServices.
func (u *UsersServivesImpl) RegisterUser(ctx context.Context, req *request.RegisterRequest) error {
	// 1. Check if email already exists (might be registered via OAuth)
	existingUser, err := u.userRepository.FindByEmail(ctx, req.Email)
	if err == nil {
		// Email exists! Check if user has OAuth provider linked
		if existingUser.ProviderName.Valid && existingUser.ProviderName.String != "" {
			return fmt.Errorf("email %s is already registered with %s. Please use %s login or use a different email.",
				req.Email, existingUser.ProviderName.String, existingUser.ProviderName.String)
		}
		// User exists but no provider - this is a duplicate registration
		return fmt.Errorf("email %s is already registered. Please login instead.", req.Email)
	}

	// 2. Hash password and create user
	password := helpers.HashPassword(req.Password)

	avatarDefault := fmt.Sprintf("https://api.dicebear.com/10.x/initials/png?seed=%s", req.Username)

	dtoToModels := models.Users{
		Username: req.Username,
		Email:    req.Email,
		Password: password,
		AvatarUrl: sql.NullString{
			String: avatarDefault,
			Valid:  true,
		},
	}

	err = u.userRepository.Create(ctx, &dtoToModels)
	if err != nil {
		return errors.New("failed to create user")
	}

	return nil
}

func (u *UsersServivesImpl) GetAllUser(ctx context.Context) ([]*response.UserResponse, error) {
	users, err := u.userRepository.FindAll(ctx)

	for _, user := range users {
		if user.AvatarUrl.Valid && !ChechkPrefixHttps(user.AvatarUrl.String) {
			result, _ := u.minioS3.GetObjectURL(ctx, user.AvatarUrl.String, "chat-app")
			user.AvatarUrl.String = result
		}
	}

	userConvert := helpers.UserResponses(users)

	if err != nil {
		return nil, errors.New("failed to get users")
	}

	return userConvert, nil
}

func (u *UsersServivesImpl) GetDetailUser(ctx context.Context, userId uuid.UUID) (*response.UserResponse, error) {
	user, err := u.userRepository.FindByID(ctx, userId)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if user.AvatarUrl.Valid && !ChechkPrefixHttps(user.AvatarUrl.String) {
		result, _ := u.minioS3.GetObjectURL(ctx, user.AvatarUrl.String, "chat-app")
		user.AvatarUrl.String = result
	}

	return helpers.UserResponse(user), nil
}

func (u *UsersServivesImpl) GetUserByID(ctx context.Context, userId uuid.UUID) (*response.UserResponse, error) {
	user, err := u.userRepository.FindByID(ctx, userId)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if user.AvatarUrl.Valid && !ChechkPrefixHttps(user.AvatarUrl.String) {
		result, _ := u.minioS3.GetObjectURL(ctx, user.AvatarUrl.String, "chat-app")
		user.AvatarUrl.String = result
	}

	return helpers.UserResponse(user), nil
}

func (u *UsersServivesImpl) StoreFirebaseToken(ctx context.Context, fcm *request.FcmRequest, userId uuid.UUID) error {
	dtoToModels := models.Firebase{
		UserID:         userId,
		InstallationID: fcm.Installation,
		FcmToken:       fcm.FcmToken,
		Platform:       fcm.Platform,
	}

	err := u.firebaseRepository.CreateFcmToken(ctx, &dtoToModels)
	if err != nil {
		return err
	}

	return nil
}

func (u *UsersServivesImpl) LogoutUser(ctx context.Context, userId uuid.UUID, fcmToken string) (int, error) {
	var count int
	var err error

	if fcmToken != "" {
		count, err = u.firebaseRepository.DeactivateTokenByUserID(ctx, userId.String(), fcmToken)
		if err != nil {
			return 0, fmt.Errorf("failed to deactivate token on logout: %w", err)
		}
	} else {
		count, err = u.firebaseRepository.DeactivateAllTokensByUserID(ctx, userId.String())
		if err != nil {
			return 0, fmt.Errorf("failed to deactivate tokens on logout: %w", err)
		}
	}
	return count, nil
}

func (u *UsersServivesImpl) UpdatedUserInfo(ctx context.Context, userId uuid.UUID, req *request.UpdateProfileRequest) error {
	userinfo, err := u.userRepository.FindByID(ctx, userId)
	if err != nil {
		return errors.New("user not found")
	}

	if req.Username != nil {
		userinfo.Username = *req.Username
	}

	if req.About != nil {
		userinfo.About.String = *req.About
		userinfo.About.Valid = true
	}

	err = u.userRepository.Update(ctx, userinfo)
	if err != nil {
		return errors.New("tidak dapat melakukan updated user data tidak valid")
	}

	return nil
}

func (u *UsersServivesImpl) UpdateAvatar(ctx context.Context, userID string, reader io.Reader, size int64, contentType, objectName string) (string, error) {
	// 1. Upload ke MinIO
	err := u.minioS3.UploadFile(ctx, "chat-app", objectName, reader, size, contentType)
	if err != nil {
		return "", fmt.Errorf("failed to upload to storage: %w", err)
	}

	// 2. Build URL
	avatarURL, _ := u.minioS3.GetObjectURL(ctx, objectName, "chat-app")

	// 3. Update DB (rollback MinIO kalau gagal)
	err = u.userRepository.ChangeAvatar(ctx, userID, avatarURL)
	if err != nil {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if rmErr := u.minioS3.DeleteObject(rollbackCtx, "chat-app", objectName); rmErr != nil {
			log.Printf("[ERR] Rollback MinIO failed for user %s: %v", userID, rmErr)
		}
		return "", fmt.Errorf("failed to update avatar in db: %w", err)
	}

	return avatarURL, nil
}

func BuildMailOtp(ctx context.Context, recipientemail, otp string) []byte {

	htmlContent := fmt.Sprintf(`
		<html>
		<body style="font-family: sans-serif;">
			<h2>Verifikasi Upgrade Tier</h2>
			<p>Masukkan kode berikut ke dalam aplikasi:</p>
			<div style="font-size: 24px; font-weight: bold;">%s</div>
		</body>
		</html>
	`, otp)

	payload := map[string]interface{}{
		"sender": map[string]string{
			"name":  "Aku Admin",
			"email": "noreply@madgo.my.id",
		},
		"to": []map[string]string{
			{
				"email": recipientemail,
			},
		},
		"subject":     "Kode Verifikasi Anda",
		"htmlContent": htmlContent,
	}

	jsonPayload, _ := json.Marshal(payload)

	return jsonPayload
}

func GenerateOtp(n int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)

	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[num.Int64()]
	}

	return string(b), nil
}

func NewUsersServices(userRepository repository.RepositoryUser, firebase repository.RepositoryFirebase, jwtConfig config.JwtConfig, espconfig config.Esp, minio storage.ObjectStorage, redis *redis.Client, httpClient *http.Client) UsersServices {
	return &UsersServivesImpl{
		userRepository:     userRepository,
		firebaseRepository: firebase,
		minioS3:            minio,
		jwtConfig:          jwtConfig,
		espConfig:          espconfig,
		httpClient:         httpClient,
		rds:                redis,
	}
}
