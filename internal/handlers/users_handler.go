package handlers

import (
	"net/http"

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
	HandlerFcmToken(c *gin.Context)
	HandlerLogout(c *gin.Context)
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

// HandlerLogin implements UserHandler.
func (u *UserHandlerImpl) HandlerLogin(c *gin.Context) {
	userPayload := new(request.LoginRequest)

	if err := c.ShouldBindJSON(userPayload); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "json tidak memnuhi validation rules",
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

func NewUserHandler(srv services.UsersServices) UserHandler {
	return &UserHandlerImpl{services: srv}
}
