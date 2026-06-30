package handlers

import (
	"context"
	"errors"
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

type HandlerRoom interface {
	HandleGetAllRoom(c *gin.Context)
	HandleGetRoomByUserID(c *gin.Context)
	HandleGetRoomByName(c *gin.Context)
	HandleCreateRoom(c *gin.Context)
	HandleGetRoomDetail(c *gin.Context)
	HandlerUpdatedRoom(c *gin.Context)
	HandlerDeleteRoom(c *gin.Context)
	HandleCreateDirectRoom(c *gin.Context)
	HandleCheckDirectRoom(c *gin.Context)
	UploadRoomAvatar(c *gin.Context)
}

type HandlerRoomImpl struct {
	srv services.RoomService
}

// HandleCheckDirectRoom implements [HandlerRoom].
func (r *HandlerRoomImpl) HandleCheckDirectRoom(c *gin.Context) {
	targetUserID := c.Param("id")

	_, err := uuid.FromString(targetUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "id tidak valid",
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

	id, res, err := r.srv.CheckDirectRoom(c.Request.Context(), userID.(string), targetUserID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, response.ApiResponse{
				Status:  http.StatusGatewayTimeout,
				Message: "Request Timeout",
			})
			return
		}
		if !res {
			c.JSON(http.StatusNotFound, response.ApiResponse{
				Status:  http.StatusNotFound,
				Message: err.Error(),
				Data:    false,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: "terjadi kesalahan pada server",
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status: http.StatusOK,
		Data: gin.H{
			"id":    id,
			"found": res,
		},
		Message: "success",
	})
}

// HandleCreateDirectRoom implements [HandlerRoom].
func (r *HandlerRoomImpl) HandleCreateDirectRoom(c *gin.Context) {
	var req request.CreateDirectRoomRequest

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found",
		})
		return
	}

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "data yang dikirim tidak valid",
		})
		return
	}

	id, err := r.srv.CreateDirectRoom(c.Request.Context(), &req, userID.(string))
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
		Data:    id,
		Message: "success",
	})

}

// HandlerDeleteRoom implements HandlerRoom.
func (r *HandlerRoomImpl) HandlerDeleteRoom(c *gin.Context) {
	roomID := c.Param("id")

	_, err := uuid.FromString(roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "id tidak valid",
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

	err = r.srv.DeleteRoom(c.Request.Context(), roomID, userID.(string))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, response.ApiResponse{
				Status:  http.StatusGatewayTimeout,
				Message: "Request Timeout",
			})
			return
		}
		c.JSON(http.StatusForbidden, response.ApiResponse{
			Status:  http.StatusForbidden,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "room deleted successfully",
		Data:    nil,
	})
}

// HandlerUpdatedRoom implements HandlerRoom.
func (r *HandlerRoomImpl) HandlerUpdatedRoom(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found",
		})
		return
	}

	roomID := c.Param("id")

	var req request.UpdateRoomRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "format JSON tidak valid",
		})
		return
	}

	_, err = uuid.FromString(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "id tidak valid",
		})
		return
	}

	err = r.srv.UpdateRoom(c.Request.Context(), roomID, userID.(string), &req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, response.ApiResponse{
				Status:  http.StatusGatewayTimeout,
				Message: "Request Timeout",
			})
			return
		}
		c.JSON(http.StatusForbidden, response.ApiResponse{
			Status:  http.StatusForbidden,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "room updated successfully",
		Data:    nil,
	})
}

// HandleCreateRoom implements HandlerRoom.
func (r *HandlerRoomImpl) HandleCreateRoom(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found in context",
		})
		return
	}

	var room request.CreateRoomRequest
	err := c.ShouldBind(&room)

	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	id, err := r.srv.CreateRoom(c.Request.Context(), &room, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	res := response.ApiResponse{
		Status:  http.StatusOK,
		Message: "success create room",
		Data:    id,
	}

	c.JSON(http.StatusOK, res)
}

// HandleGetRoomByName implements HandlerRoom.
func (r *HandlerRoomImpl) HandleGetRoomByName(c *gin.Context) {
	var name request.GetRoomByName
	err := c.ShouldBind(&name)

	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	rooms, err := r.srv.GetRoomByName(c.Request.Context(), name)

	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	res := response.ApiResponse{
		Status:  http.StatusOK,
		Data:    rooms,
		Message: "success get room by name " + name.Name,
	}

	c.JSON(http.StatusOK, res)
}

// GetAllRoom implements HandlerRoomImpl.
func (r *HandlerRoomImpl) HandleGetAllRoom(c *gin.Context) {
	rooms, err := r.srv.GetAllRoomUser(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	res := response.ApiResponse{
		Status:  http.StatusOK,
		Data:    rooms,
		Message: "success get all room",
	}

	c.JSON(http.StatusOK, res)
}

// GetRoomByID implements HandlerRoomImpl.
func (r *HandlerRoomImpl) HandleGetRoomByUserID(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found",
		})
		return
	}

	if _, err := uuid.FromString(userID.(string)); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "Invalid room ID format",
		})
		return
	}

	room, err := r.srv.GetRoomByUserID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	res := response.ApiResponse{
		Status:  http.StatusOK,
		Data:    room,
		Message: "success get room by userid ",
	}

	c.JSON(http.StatusOK, res)
}

func NewRoomHandler(srv services.RoomService) HandlerRoom {
	return &HandlerRoomImpl{srv: srv}
}

// HandleGetRoomDetail implements HandlerRoom - returns full room details with members list
func (r *HandlerRoomImpl) HandleGetRoomDetail(c *gin.Context) {
	roomID := c.Param("id")

	// Validate room ID format
	if _, err := uuid.FromString(roomID); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid room ID format",
		})
		return
	}

	// Get user ID from JWT middleware
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "unauthorized: user_id not found",
		})
		return
	}

	// Get room detail
	roomDetail, err := r.srv.GetRoomDetail(c.Request.Context(), roomID, userID.(string))
	if err != nil {
		if err.Error() == "forbidden: you are not a member of this room" {
			c.JSON(http.StatusForbidden, response.ApiResponse{
				Status:  http.StatusForbidden,
				Message: err.Error(),
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
		Data:    roomDetail,
		Message: "success get room detail",
	})
}

func (r *HandlerRoomImpl) UploadRoomAvatar(c *gin.Context) {
	ctx := c.Request.Context()

	roomID := c.Param("id")
	if _, err := uuid.FromString(roomID); err != nil {
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

	objectName := fmt.Sprintf("rooms/%s/avatar_%d%s",
		roomID,
		time.Now().Unix(),
		filepath.Ext(fileHeader.Filename),
	)

	avatarURL, err := r.srv.UpdateAvatar(ctx, roomID, userID.(string), src, fileHeader.Size, contentType, objectName)
	if err != nil {
		log.Printf("[ERR] Upload room avatar: %v", err)
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: "failed to upload avatar: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "avatar updated",
		Data:    avatarURL,
	})
}
