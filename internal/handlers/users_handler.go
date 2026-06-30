package handlers

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

type UserHandler interface {
	HandlerRegister(c *gin.Context)
	HandlerLogin(c *gin.Context)
	HandlerRefresh(c *gin.Context)
	HandlerGetAllUser(c *gin.Context)
	HandlerGetDetailUser(c *gin.Context)
	HandlerGetUserByID(c *gin.Context)
	HandlerFcmToken(c *gin.Context)
	HandlerLogout(c *gin.Context)
	HandlerUpdateUser(c *gin.Context)
	UploadUserAvatar(c *gin.Context)
}

type UserHandlerImpl struct {
	services services.UsersServices
}

// HandlerGetAllUser implements [UserHandler].
func (u *UserHandlerImpl) HandlerGetAllUser(c *gin.Context) {
	users, err := u.services.GetAllUser(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Data:    users,
		Message: "success get all user",
	})
}

// HandlerGetDetailUser implements [UserHandler].
func (u *UserHandlerImpl) HandlerGetDetailUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found in context",
		})
		return
	}

	id, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid user id",
		})
		return
	}

	user, err := u.services.GetDetailUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Data:    user,
		Message: "success get detail user",
	})
}

// HandlerGetUserByID implements [UserHandler] - get other user's profile by ID
func (u *UserHandlerImpl) HandlerGetUserByID(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "user id is required",
		})
		return
	}

	id, err := uuid.FromString(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid user id",
		})
		return
	}

	user, err := u.services.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, response.ApiResponse{
			Status:  http.StatusNotFound,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Data:    user,
		Message: "success get user",
	})
}

func (u *UserHandlerImpl) HandlerUpdateUser(c *gin.Context) {
	userPayload := new(request.UpdateProfileRequest)

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found",
		})
		return
	}

	id, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid user id",
		})
		return
	}

	err = c.ShouldBindJSON(userPayload)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "data yang di kirim tidak valid",
		})
		return
	}

	err = u.services.UpdatedUserInfo(c.Request.Context(), id, userPayload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "success update user",
	})
}

// HandlerLogin implements UserHandler.
func (u *UserHandlerImpl) HandlerLogin(c *gin.Context) {
	userPayload := new(request.LoginRequest)

	if err := c.ShouldBindJSON(userPayload); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "data tidak valid",
		})
		return
	}

	resp, err := u.services.LoginUser(c.Request.Context(), userPayload)

	if err != nil {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: err.Error(),
		})
		return
	}

	c.Set("user_info", resp.Userinfo)
	c.JSON(http.StatusOK, gin.H{"token": resp.AccessToken, "refresh_token": resp.RefreshToken})
}

// Reads refresh token from Authorization header: "Bearer <token>"
func (u *UserHandlerImpl) HandlerRefresh(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "authorization header is required",
		})
		return
	}

	resp, err := u.services.RefreshUser(c.Request.Context(), authHeader)

	if err != nil {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: err.Error(),
		})
		return
	}

	c.Set("user_info", resp.Userinfo)
	c.JSON(http.StatusOK, gin.H{"token": resp.AccessToken, "refresh_token": resp.RefreshToken})
}

// HandlerRegister implements UserHandler.
func (u *UserHandlerImpl) HandlerRegister(c *gin.Context) {
	userRegister := new(request.RegisterRequest)

	if err := c.ShouldBindJSON(userRegister); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	err := u.services.RegisterUser(c.Request.Context(), userRegister)

	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success register user"})
}

// HandlerRegister implements UserHandler.
func (u *UserHandlerImpl) HandlerFcmToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found in context",
		})
		return
	}

	id, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "id tidak valid",
		})
		return
	}

	fcmToken := new(request.FcmRequest)

	if err := c.ShouldBindJSON(fcmToken); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	err = u.services.StoreFirebaseToken(c.Request.Context(), fcmToken, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success register user"})
}

// HandlerLogout deactivates FCM tokens for the logged-in user
func (u *UserHandlerImpl) HandlerLogout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found in context",
		})
		return
	}

	id, err := uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid user id",
		})
		return
	}

	logoutReq := new(request.LogoutRequest)
	if err := c.ShouldBindJSON(logoutReq); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid request body",
		})
		// Allow empty body (logout from all devices)
		// logoutReq = &request.LogoutRequest{}
	}

	count, err := u.services.LogoutUser(c.Request.Context(), id, logoutReq.FcmToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "success logout user",
		Data:    count,
	})
}

func (h *UserHandlerImpl) UploadUserAvatar(c *gin.Context) {
	ctx := c.Request.Context()

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized",
		})
		return
	}

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "avatar file is required",
		})
		return
	}

	// Validasi tipe
	contentType := fileHeader.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
	if !allowedTypes[contentType] {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid file type, only jpg/png/webp allowed",
		})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: "failed to open file",
		})
		return
	}
	defer src.Close()

	objectName := fmt.Sprintf("users/%s/avatar_%d%s",
		userID,
		time.Now().Unix(),
		filepath.Ext(fileHeader.Filename),
	)

	avatarURL, err := h.services.UpdateAvatar(ctx, userID.(string), src, fileHeader.Size, contentType, objectName)
	if err != nil {
		log.Printf("[ERR] Upload user avatar: %v", err)
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: "failed to upload avatar: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "avatar updated",
		Data:    gin.H{"avatar_url": avatarURL},
	})
}

func NewUserHandler(srv services.UsersServices) UserHandler {
	return &UserHandlerImpl{services: srv}
}
