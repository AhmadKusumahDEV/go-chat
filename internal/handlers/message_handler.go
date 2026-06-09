package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/AhmadKusumahDEV/go-chat/internal/services"
	"github.com/AhmadKusumahDEV/go-chat/internal/worker"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
)

type HandlerMessage interface {
	HandleGetRoomMessages(c *gin.Context)
	HandleSendMessage(c *gin.Context)
	HandleEditMessage(c *gin.Context)
	UploadMultipleImages(c *gin.Context)
}

type HandlerMessageImpl struct {
	srv        services.MessageService
	dispatcher *worker.Dispatcher
	cfg        config.Cfg
	redis      *redis.Client
}

func NewMessageHandler(srv services.MessageService, dispatcher *worker.Dispatcher, cfg config.Cfg, rds *redis.Client) HandlerMessage {
	return &HandlerMessageImpl{
		srv:        srv,
		dispatcher: dispatcher,
		cfg:        cfg,
		redis:      rds,
	}
}

// HandleGetRoomMessages returns paginated messages for a room.
func (h *HandlerMessageImpl) HandleGetRoomMessages(c *gin.Context) {
	roomID := c.Param("room_id")

	query := c.Request.URL.Query()

	nextCursor := query.Get("next_cursor")
	var cursorPtr *string
	if nextCursor != "" {
		cursorPtr = &nextCursor
	}

	limitStr := query.Get("limit")
	limit := 20 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if _, err := uuid.FromString(roomID); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid room ID format",
		})
		return
	}

	userInfo, exists := c.Get("user_info")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found",
		})
		return
	}

	jwtUserInfo := userInfo.(*models.JwtUsersInfo)

	if _, err := uuid.FromString(jwtUserInfo.UserID); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid user ID format",
		})
		return
	}

	result, err := h.srv.GetRoomMessages(c.Request.Context(), roomID, jwtUserInfo.UserID, limit, cursorPtr)
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

	log.Println(result)

	c.JSON(http.StatusOK, result)
}

// HandleSendMessage creates a new message in a room.
func (h *HandlerMessageImpl) HandleSendMessage(c *gin.Context) {
	var req request.CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid request body",
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

	msg, err := h.srv.SendMessage(c.Request.Context(), &req, userID.(string))
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
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "invalid message ID format",
		})
		return
	}

	var req request.UpdateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "content is required",
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

	err = h.srv.EditMessage(c.Request.Context(), messageID, userID.(string), &req)
	if err != nil {
		c.JSON(http.StatusForbidden, response.ApiResponse{
			Status:  http.StatusForbidden,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse{
		Status:  http.StatusOK,
		Message: "message updated",
		Data:    nil,
	})
}

func (h *HandlerMessageImpl) UploadMultipleImages(c *gin.Context) {
	ctx := c.Request.Context()

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.ApiResponse{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized: user_id not found",
		})
		return
	}

	var (
		roomID  = c.PostForm("room_id")
		content = c.PostForm("content")
	)

	tempDir := h.cfg.PathTemp
	if tempDir == "" {
		tempDir = os.TempDir()
	} else {
		tempDir = filepath.Clean(tempDir)
	}

	if roomID == "" {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "room_id are required",
		})
		return
	}

	messageID, err := uuid.NewV6()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: "failed to generate message ID",
		})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse{
			Status:  http.StatusBadRequest,
			Message: "data yang di kirim tidak valid",
		})
		return
	}

	files := form.File["images"]

	batchID := fmt.Sprintf("%s_%d", userID, time.Now().UnixNano())

	redisKey := "batch:status:" + batchID
	h.redis.HSet(ctx, redisKey, map[string]any{
		"total":     len(files),
		"remaining": len(files),
		"failed":    0,
	})

	h.redis.Expire(ctx, redisKey, 1*time.Hour)

	_, err = h.srv.SendMessage(ctx, &request.CreateMessageRequest{
		RoomID:    roomID,
		MessageID: messageID.String(),
		Type:      "image",
		Content:   content,
		SenderID:  userID.(string),
	}, userID.(string))
	if err != nil {
		log.Printf("[ERR] Gagal create message untuk upload: %v", err)
		c.JSON(http.StatusInternalServerError, response.ApiResponse{
			Status:  http.StatusInternalServerError,
			Message: "Failed to create message: " + err.Error(),
		})
		return
	}

	for _, fileHeader := range files {

		src, err := fileHeader.Open()
		if err != nil {
			log.Printf("[ERR] Gagal membuka file: %v", err)
			continue
		}

		tempFile, err := os.CreateTemp(tempDir, "chat-upload-*.tmp")
		if err != nil {
			log.Printf("[ERR] Gagal membuat file temporary: %v", err)
			src.Close()
			continue
		}

		_, err = io.Copy(tempFile, src)
		src.Close()
		tempFile.Close()
		if err != nil {
			os.Remove(tempFile.Name())
			continue
		}

		log.Printf("chats/%s/%s_%s", userID, batchID, fileHeader.Filename)

		job := worker.UploadJob{
			UserID:         userID.(string),
			RoomID:         roomID,
			BatchID:        batchID,
			MessageID:      messageID.String(),
			MessageType:    "image",
			MessageContent: content,
			BucketName:     "chat-app",
			ObjectName:     fmt.Sprintf("chats/%s/%s_%s", userID, batchID, fileHeader.Filename),
			TempFilePath:   tempFile.Name(),
			ContentType:    fileHeader.Header.Get("Content-Type"),
			FileName:       fileHeader.Filename,
		}

		h.dispatcher.EnqueueJob(job)
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":  "accepted",
		"message": "Files are being processed please wait for a message",
	})
}
