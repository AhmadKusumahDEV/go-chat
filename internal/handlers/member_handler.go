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
	HandlePromoteAdmin(c *gin.Context)
	HandleDemoteAdmin(c *gin.Context)
	HanleTransferOwnership(c *gin.Context)
}

type HandlerMemberImpl struct {
	srv services.MemberService
}

func NewMemberHandler(srv services.MemberService) HandlerMember {
	return &HandlerMemberImpl{
		srv: srv,
	}
}

func (h *HandlerMemberImpl) HandlePromoteAdmin(c *gin.Context) {
	roomID := c.Param("room_id")

	_, err := uuid.FromString(roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "room tidak valid",
		})
		return
	}

	var req request.ManageMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "format JSON tidak valid",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found",
		})
		return
	}

	err = h.srv.PromoteAdmin(c.Request.Context(), roomID, req.TargetUserID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    nil,
	})
}

func (h *HandlerMemberImpl) HandleDemoteAdmin(c *gin.Context) {
	roomID := c.Param("room_id")

	_, err := uuid.FromString(roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "room tidak valid",
		})
		return
	}

	var req request.ManageMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "format JSON tidak valid",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found",
		})
		return
	}

	err = h.srv.DemoteAdmin(c.Request.Context(), roomID, req.TargetUserID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusAccepted, response.ApiResponse{
		Status:  http.StatusAccepted,
		Message: "success",
		Data:    nil,
	})
}

func (h *HandlerMemberImpl) HanleTransferOwnership(c *gin.Context) {
	roomID := c.Param("room_id")

	_, err := uuid.FromString(roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "room tidak valid",
		})
		return
	}

	var req request.ManageMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "format JSON tidak valid",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found",
		})
		return
	}

	err = h.srv.TransferOwnership(c.Request.Context(), roomID, userID.(string), req.TargetUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusAccepted, response.ApiResponse{
		Status:  http.StatusAccepted,
		Message: "success",
		Data:    nil,
	})
}

// HandleAddMember implements HandlerMember.
func (h *HandlerMemberImpl) HandleAddMember(c *gin.Context) {
	var member request.AddMemberRequest

	err := c.ShouldBindJSON(&member)

	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "format JSON tidak valid",
		})
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
			c.JSON(http.StatusGatewayTimeout, response.ApiResponse{
				Status:  http.StatusGatewayTimeout,
				Message: "Request Timeout",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, response.ApiResponse{
		Status:  http.StatusCreated,
		Message: "members added successfully",
		Data:    nil,
	})
}

// HandleGetMembers implements HandlerMember.
func (h *HandlerMemberImpl) HandleGetMembers(c *gin.Context) {
	roomID := c.Param("room_id")

	_, err := uuid.FromString(roomID)

	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "id tidak valid",
		})
		return
	}

	members, err := h.srv.GetMembers(c.Request.Context(), roomID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, response.ApiResponse{
				Status:  http.StatusGatewayTimeout,
				Message: "Request Timeout",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
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
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid room ID format",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found",
		})
		return
	}

	err = h.srv.LeaveRoom(c.Request.Context(), roomID, userID.(string))
	if err != nil {
		c.JSON(http.StatusForbidden, response.ApiResponse{
			Status:  http.StatusForbidden,
			Message: err.Error(),
		})
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
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid room ID format",
		})
		return
	}

	var batchUser request.RemoveMemberRequest

	if err := c.ShouldBindJSON(&batchUser); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "format data tidak valid",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found",
		})
		return
	}

	err = h.srv.RemoveMember(c.Request.Context(), roomID, userID.(string), batchUser)
	if err != nil {
		c.JSON(http.StatusForbidden, response.ApiResponse{
			Status:  http.StatusForbidden,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "member removed successfully",
		Data:    nil,
	})
}
