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

type HandlerRoom interface {
	HandleGetAllRoom(c *gin.Context)
	HandleGetRoomByUserID(c *gin.Context)
	HandleGetRoomByName(c *gin.Context)
	HandleJoinRoom(c *gin.Context)
	HandleCreateRoom(c *gin.Context)
	HandlerUpdatedRoom(c *gin.Context)
	HandlerDeleteRoom(c *gin.Context)
	HandlerGetRoomMembers(c *gin.Context)
	HandleLeaveRoom(c *gin.Context)
	HandleRemoveMember(c *gin.Context)
}

type HandlerRoomImpl struct {
	srv services.RoomService
}

// HandlerDeleteRoom implements HandlerRoom.
func (r *HandlerRoomImpl) HandlerDeleteRoom(c *gin.Context) {
	roomID := c.Param("id")

	_, err := uuid.FromString(roomID)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("id tidak valid"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.AbortWithError(http.StatusUnauthorized, errors.New("Unauthorized: user_id not found"))
		return
	}

	err = r.srv.DeleteRoom(c.Request.Context(), roomID, userID.(string))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.AbortWithError(http.StatusGatewayTimeout, errors.New("Request Timeout"))
			return
		}
		c.AbortWithError(http.StatusForbidden, err)
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "room deleted successfully",
		Data:    nil,
	})
}

// HandlerGetRoomMembers implements HandlerRoom.
func (r *HandlerRoomImpl) HandlerGetRoomMembers(c *gin.Context) {
	roomID := c.Param("id")

	_, err := uuid.FromString(roomID)

	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("id tidak valid"))
		return
	}

	members, err := r.srv.GetRoomMembers(c.Request.Context(), roomID)

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
		Data:    members, // Kirim object/struct data, bukan string
	})
}

// HandlerUpdatedRoom implements HandlerRoom.
func (r *HandlerRoomImpl) HandlerUpdatedRoom(c *gin.Context) {
	roomID := c.Param("id")

	var req request.UpdateRoomRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("format JSON tidak valid"))
		return
	}

	_, err = uuid.FromString(roomID)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("id tidak valid"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.AbortWithError(http.StatusUnauthorized, errors.New("Unauthorized: user_id not found"))
		return
	}

	err = r.srv.UpdateRoom(c.Request.Context(), roomID, userID.(string), &req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.AbortWithError(http.StatusGatewayTimeout, errors.New("Request Timeout"))
			return
		}
		c.AbortWithError(http.StatusForbidden, err)
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
	var room request.CreateRoomRequest
	err := c.ShouldBind(&room)

	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.AbortWithError(http.StatusUnauthorized, errors.New("Unauthorized: user_id not found in context"))
		return
	}

	err = r.srv.CreateRoom(c.Request.Context(), &room, userID.(string))

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	res := response.ApiResponse{
		Status:  http.StatusOK,
		Data:    room,
		Message: "success create room",
	}

	c.JSON(http.StatusOK, res)
}

// HandleGetRoomByName implements HandlerRoom.
func (r *HandlerRoomImpl) HandleGetRoomByName(c *gin.Context) {
	var name request.GetRoomByName
	err := c.ShouldBind(&name)

	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	rooms, err := r.srv.GetRoomByName(c.Request.Context(), name)

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
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
		c.AbortWithError(http.StatusInternalServerError, err)
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
	userid := c.Param("id")

	if _, err := uuid.FromString(userid); err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("Invalid room ID format"))
		return
	}

	room, err := r.srv.GetRoomByUserID(c.Request.Context(), userid)

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	res := response.ApiResponse{
		Status:  http.StatusOK,
		Data:    room,
		Message: "success get room by userid " + userid,
	}

	c.JSON(http.StatusOK, res)
}

// JoinRoom implements HandlerRoomImpl.
func (r *HandlerRoomImpl) HandleJoinRoom(c *gin.Context) {
	var member request.AddMember

	err := c.ShouldBindJSON(&member)

	if err != nil {
		c.AbortWithError(http.StatusBadRequest, errors.New("format JSON tidak valid"))
		return
	}

	err = r.srv.AddRoomMember(c.Request.Context(), member)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			c.AbortWithError(http.StatusGatewayTimeout, errors.New("Request Timeout"))
			return
		}
		c.AbortWithError(http.StatusInternalServerError, errors.New("Error internal server"))
		return
	}

	c.JSON(201, response.ApiResponse{
		Status:  201,
		Message: "success",
		Data:    nil,
	})
}

func NewRoomHandler(srv services.RoomService) HandlerRoom {
	return &HandlerRoomImpl{srv: srv}
}

// HandleLeaveRoom implements HandlerRoom.
func (r *HandlerRoomImpl) HandleLeaveRoom(c *gin.Context) {
	roomID := c.Param("id")

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

	err = r.srv.LeaveRoom(c.Request.Context(), roomID, userID.(string))
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

// HandleRemoveMember implements HandlerRoom.
func (r *HandlerRoomImpl) HandleRemoveMember(c *gin.Context) {
	roomID := c.Param("id")

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

	err = r.srv.RemoveRoomMember(c.Request.Context(), roomID, body.TargetUserID, userID.(string))
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
