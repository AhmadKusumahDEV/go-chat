package handlers

import (
	"errors"
	"net/http"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/AhmadKusumahDEV/go-chat/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

type HandlerMessage interface {
	HandleGetRoomMessages(c *gin.Context)
	HandleSendMessage(c *gin.Context)
	HandleEditMessage(c *gin.Context)
}

type HandlerMessageImpl struct {
	srv services.MessageService
}

func NewMessageHandler(srv services.MessageService) HandlerMessage {
	return &HandlerMessageImpl{srv: srv}
}

// HandleGetRoomMessages returns the latest 20 messages for a room.
func (h *HandlerMessageImpl) HandleGetRoomMessages(c *gin.Context) {
	roomID := c.Param("id")

	userInfo, exists := c.Get("user_info")
	if !exists {
		c.AbortWithError(http.StatusUnauthorized, errors.New("Unauthorized: user_id not found"))
		return
	}

	jwtUserInfo := userInfo.(*models.JwtUsersInfo)

	_, err := uuid.FromString(jwtUserInfo.UserID)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("invalid room ID format"))
		return
	}

	messages, err := h.srv.GetRoomMessages(c.Request.Context(), roomID, jwtUserInfo.UserID)
	if err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    messages,
	})
}

// HandleSendMessage creates a new message in a room.
func (h *HandlerMessageImpl) HandleSendMessage(c *gin.Context) {
	var req request.CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("invalid request body"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.AbortWithError(http.StatusUnauthorized, errors.New("Unauthorized: user_id not found"))
		return
	}

	msg, err := h.srv.SendMessage(c.Request.Context(), &req, userID.(string))
	if err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}

	c.JSON(http.StatusCreated, response.ApiResponse{
		Status:  http.StatusCreated,
		Message: "message sent",
		Data:    msg,
	})
}

// HandleEditMessage updates a message's content.
func (h *HandlerMessageImpl) HandleEditMessage(c *gin.Context) {
	messageID := c.Param("id")

	_, err := uuid.FromString(messageID)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("invalid message ID format"))
		return
	}

	var req request.UpdateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("content is required"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.AbortWithError(http.StatusUnauthorized, errors.New("Unauthorized: user_id not found"))
		return
	}

	err = h.srv.EditMessage(c.Request.Context(), messageID, userID.(string), &req)
	if err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "message updated",
		Data:    nil,
	})
}
