package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

type HandlerMember interface {
	HandleAddMember(c *gin.Context)
	HandleGetMembers(c *gin.Context)
	HandleLeaveRoom(c *gin.Context)
	HandleRemoveMember(c *gin.Context)
}

type HandlerMemberImpl struct {
	srv services.MemberService
}

func NewMemberHandler(srv services.MemberService) HandlerMember {
	return &HandlerMemberImpl{srv: srv}
}

// HandleAddMember implements HandlerMember.
func (h *HandlerMemberImpl) HandleAddMember(c *gin.Context) {
	var member request.AddMember

	err := c.ShouldBindJSON(&member)

	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("format JSON tidak valid"))
		return
	}

	// Assuming the requester is the one adding
	userID, exists := c.Get("user_id")
	if exists && member.AddMemberBy == "" {
		member.AddMemberBy = userID.(string)
	}

	err = h.srv.AddMember(c.Request.Context(), member)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.AbortWithError(http.StatusGatewayTimeout, errors.New("Request Timeout"))
			return
		}
		c.AbortWithError(http.StatusInternalServerError, errors.New("Error internal server: "+err.Error()))
		return
	}

	c.JSON(http.StatusCreated, response.ApiResponse{
		Status:  http.StatusCreated,
		Message: "success",
		Data:    nil,
	})
}

// HandleGetMembers implements HandlerMember.
func (h *HandlerMemberImpl) HandleGetMembers(c *gin.Context) {
	roomID := c.Param("room_id")

	_, err := uuid.FromString(roomID)

	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("id tidak valid"))
		return
	}

	members, err := h.srv.GetMembers(c.Request.Context(), roomID)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.AbortWithError(http.StatusGatewayTimeout, errors.New("Request Timeout"))
			return
		}
		c.AbortWithError(http.StatusInternalServerError, errors.New("Error internal server"))
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    members,
	})
}

// HandleLeaveRoom implements HandlerMember.
func (h *HandlerMemberImpl) HandleLeaveRoom(c *gin.Context) {
	roomID := c.Param("room_id")

	_, err := uuid.FromString(roomID)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("invalid room ID format"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.AbortWithError(http.StatusUnauthorized, errors.New("Unauthorized: user_id not found"))
		return
	}

	err = h.srv.LeaveRoom(c.Request.Context(), roomID, userID.(string))
	if err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "left room successfully",
		Data:    nil,
	})
}

// HandleRemoveMember implements HandlerMember.
func (h *HandlerMemberImpl) HandleRemoveMember(c *gin.Context) {
	roomID := c.Param("room_id")

	_, err := uuid.FromString(roomID)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("invalid room ID format"))
		return
	}

	var body struct {
		TargetUserID string `json:"target_user_id" binding:"required,uuid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("target_user_id is required"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.AbortWithError(http.StatusUnauthorized, errors.New("Unauthorized: user_id not found"))
		return
	}

	err = h.srv.RemoveMember(c.Request.Context(), roomID, body.TargetUserID, userID.(string))
	if err != nil {
		c.AbortWithError(http.StatusForbidden, err)
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "member removed successfully",
		Data:    nil,
	})
}
