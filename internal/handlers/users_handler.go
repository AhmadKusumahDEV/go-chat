package handlers

import (
	"errors"
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
}

type UserHandlerImpl struct {
	services services.UsersServices
}

// HandlerGetAllUser implements [UserHandler].
func (u *UserHandlerImpl) HandlerGetAllUser(c *gin.Context) {
	users, err := u.services.GetAllUser(c.Request.Context())
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
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
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	resp, err := u.services.LoginUser(c.Request.Context(), userPayload)

	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, err)
		return
	}

	c.Set("user_info", resp.Userinfo)
	c.JSON(http.StatusOK, gin.H{"token": resp.AccessToken, "refresh_token": resp.RefreshToken})
}

// HandlerRefresh implements UserHandler.
func (u *UserHandlerImpl) HandlerRefresh(c *gin.Context) {
	refreshPayload := new(request.RefreshRequest)

	if err := c.ShouldBindJSON(refreshPayload); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	resp, err := u.services.RefreshUser(c.Request.Context(), refreshPayload)

	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, err)
		return
	}

	c.Set("user_info", resp.Userinfo)
	c.JSON(http.StatusOK, gin.H{"Token": resp.AccessToken, "refresh_token": resp.RefreshToken})
}

// HandlerRegister implements UserHandler.
func (u *UserHandlerImpl) HandlerRegister(c *gin.Context) {
	userRegister := new(request.RegisterRequest)

	if err := c.ShouldBindJSON(userRegister); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	err := u.services.RegisterUser(c.Request.Context(), userRegister)

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success register user"})
}

// HandlerRegister implements UserHandler.
func (u *UserHandlerImpl) HandlerFcmToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.AbortWithError(http.StatusUnauthorized, errors.New("Unauthorized: user_id not found in context"))
		return
	}

	id, err := uuid.FromString(userID.(string))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("id tidak valid"))
		return
	}

	fcmToken := new(request.FcmRequest)

	if err := c.ShouldBindJSON(fcmToken); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	err = u.services.StoreFirebaseToken(c.Request.Context(), fcmToken, id)

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success register user"})
}

func NewUserHandler(srv services.UsersServices) UserHandler {
	return &UserHandlerImpl{services: srv}
}
