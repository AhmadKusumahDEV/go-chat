package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/config"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/AhmadKusumahDEV/go-chat/internal/websocket"
	"github.com/AhmadKusumahDEV/go-chat/pkg/storage"
	"github.com/redis/go-redis/v9"
)

// UploadStep indicates at which stage the upload failed
type UploadStep string

const (
	StepTempFile UploadStep = "temp_file"
	StepMinIO    UploadStep = "minio_upload"
	StepDBWrite  UploadStep = "db_write"
	StepRollback UploadStep = "rollback"
)

// FailedFileInfo stores details of a failed upload
type FailedFileInfo struct {
	FileName string     `json:"file_name"`
	Error    string     `json:"error"`
	Step     UploadStep `json:"step"`
}

// UploadBatchResponse is the WebSocket response sent to the client (sender only)
type UploadBatchResponse struct {
	Type         string           `json:"type"`   // "upload_batch_status"
	Status       string           `json:"status"` // "success" | "failed" | "partial_failed"
	MessageID    string           `json:"message_id"`
	RoomID       string           `json:"room_id"`
	TotalFiles   int              `json:"total_files"`
	FailedCount  int              `json:"failed_count"`
	SuccessCount int              `json:"success_count"`
	FailedFiles  []FailedFileInfo `json:"failed_files,omitempty"`
}

// NewMessageBroadcast is sent to OTHER room members when message is created
type NewMessageBroadcast struct {
	Type        string           `json:"type"` // "new_message"
	MessageID   string           `json:"message_id"`
	RoomID      string           `json:"room_id"`
	SenderID    string           `json:"sender_id"`
	Content     string           `json:"content"`
	MessageType string           `json:"message_type"`
	Timestamp   string           `json:"timestamp"`
	Attachments []AttachmentInfo `json:"attachments,omitempty"`
}

// AttachmentInfo represents attachment data in broadcast
type AttachmentInfo struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	FileSize int64  `json:"file_size"`
}

type UploadJob struct {
	UserID         string
	RoomID         string
	BatchID        string
	MessageID      string
	MessageType    string // "text" | "image"
	MessageContent string // content text (can be empty)
	ObjectName     string
	BucketName     string
	TempFilePath   string
	ContentType    string
	FileName       string
}

type Dispatcher struct {
	queue         chan UploadJob
	minio         config.Cfg
	storageClient storage.ObjectStorage
	attachments   repository.AttachmentsRepository
	redis         *redis.Client
	wsManager     websocket.WebSocketManager
	maxWorkers    int
}

func NewDispatcher(storageClient storage.ObjectStorage, rds *redis.Client, cfg config.Cfg, attch repository.AttachmentsRepository, wsManager websocket.WebSocketManager, maxWorkers, queueSize int) *Dispatcher {
	return &Dispatcher{
		queue:         make(chan UploadJob, queueSize),
		redis:         rds,
		storageClient: storageClient,
		minio:         cfg,
		wsManager:     wsManager,
		maxWorkers:    maxWorkers,
		attachments:   attch,
	}
}

func (d *Dispatcher) StartWorkerPool() {
	for i := 1; i <= d.maxWorkers; i++ {
		go func(workerID int) {
			for job := range d.queue {
				d.processUpload(job)
			}
		}(i)
	}
}

func (d *Dispatcher) EnqueueJob(job UploadJob) {
	d.queue <- job
}

func (d *Dispatcher) processUpload(job UploadJob) {
	file, err := os.Open(job.TempFilePath)
	if err != nil {
		log.Printf("[ERR] Gagal membaca file temporary %s: %v", job.FileName, err)
		d.checkBatchStatus(job, false, string(StepTempFile), err.Error())
		return
	}
	defer func() {
		log.Println("execute remove file temp " + job.TempFilePath)
		os.Remove(job.TempFilePath)
	}()
	defer file.Close()

	fileInfo, _ := file.Stat()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = d.storageClient.UploadFile(ctx, job.BucketName, job.ObjectName, file, fileInfo.Size(), job.ContentType)
	if err != nil {
		log.Printf("[ERR] Gagal upload ke MinIO untuk file %s: %v", job.FileName, err)
		d.checkBatchStatus(job, false, string(StepMinIO), err.Error())
		return
	}

	attchments := models.Attachments{
		MessageID: job.MessageID,
		RoomID:    job.RoomID,
		URL:       job.ObjectName,
		FileName:  job.FileName,
		FileType:  job.ContentType,
		FileSize:  fileInfo.Size(),
	}

	err = d.attachments.Create(ctx, &attchments)
	if err != nil {
		log.Printf("Gagal membuat attachments di DB untuk file %s: %v. Memulai proses rollback MinIO...", job.FileName, err)
		rollbackCtx, cancelRollback := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelRollback()

		errRollback := d.storageClient.DeleteObject(rollbackCtx, job.BucketName, job.ObjectName)

		if errRollback != nil {
			log.Printf("Gagal rollback MinIO untuk file %s: %v", job.FileName, errRollback)
			d.checkBatchStatus(job, false, string(StepRollback), fmt.Sprintf("db_error: %v, rollback_error: %v", err, errRollback))
			return
		}

		log.Printf("Berhasil rollback MinIO untuk file %s", job.FileName)
		d.checkBatchStatus(job, false, string(StepDBWrite), err.Error())
		return
	}

	d.checkBatchStatus(job, true, "", "")
}

func (d *Dispatcher) checkBatchStatus(job UploadJob, isSuccess bool, step string, errMsg string) {
	ctx := context.Background()
	redisKey := "batch:status:" + job.BatchID
	failedFilesKey := "batch:failed_files:" + job.BatchID

	if !isSuccess {
		failedInfo := FailedFileInfo{
			FileName: job.FileName,
			Error:    errMsg,
			Step:     UploadStep(step),
		}
		failedInfoJSON, _ := json.Marshal(failedInfo)

		pipe := d.redis.Pipeline()
		pipe.HIncrBy(ctx, redisKey, "failed", 1)
		pipe.SAdd(ctx, failedFilesKey, string(failedInfoJSON))

		_, err := pipe.Exec(ctx)
		if err != nil {
			log.Printf("[ERR] Gagal execute Redis pipeline untuk batch %s: %v", job.BatchID, err)
		}
	} else {
		_, err := d.redis.HIncrBy(ctx, redisKey, "remaining", -1).Result()
		if err != nil {
			log.Printf("[ERR] Gagal decrement remaining counter untuk batch %s: %v", job.BatchID, err)
			return
		}
	}

	remaining, err := d.redis.HGet(ctx, redisKey, "remaining").Int()
	if err != nil && err != redis.Nil {
		log.Printf("[ERR] Gagal get remaining count untuk batch %s: %v", job.BatchID, err)
		return
	}

	if remaining <= 0 {
		res, _ := d.redis.HGetAll(ctx, redisKey).Result()

		failedFiles, _ := d.redis.SMembers(ctx, failedFilesKey).Result()

		d.redis.Del(ctx, redisKey)
		d.redis.Del(ctx, failedFilesKey)

		failedCount, _ := strconv.Atoi(res["failed"])
		totalCount, _ := strconv.Atoi(res["total"])

		d.broadcastMessageToRoom(job, failedCount, totalCount)

		d.sendUploadStatusToUser(job.UserID, job.RoomID, job.MessageID, totalCount, failedCount, failedFiles)
	}
}

func (d *Dispatcher) broadcastMessageToRoom(job UploadJob, failedCount, totalCount int) {
	successCount := totalCount - failedCount
	if successCount <= 0 {
		log.Printf("[WS] Tidak ada upload sukses, skip broadcast ke room %s", job.RoomID)
		return
	}

	messageByID, err := d.wsManager.MessageByID(job.MessageID)
	if err != nil {
		log.Printf("Gagal fetch message %s: %v", job.MessageID, err)
		return
	}

	// attachments, err := d.attachments.FindByMessageID(ctx, job.MessageID)
	// if err != nil {
	// 	log.Printf("[ERR] Gagal fetch attachments untuk message %s: %v", job.MessageID, err)
	// 	return
	// }

	// var attachmentList []AttachmentInfo
	// for _, att := range attachments {
	// 	attachmentList = append(attachmentList, AttachmentInfo{
	// 		URL:      att.URL,
	// 		FileName: att.FileName,
	// 		FileType: att.FileType,
	// 		FileSize: att.FileSize,
	// 	})
	// }

	// messageType := job.MessageType
	// if messageType == "" {
	// 	if job.MessageContent != "" && len(attachmentList) > 0 {
	// 		messageType = "mixed"
	// 	} else if len(attachmentList) > 0 {
	// 		messageType = "image"
	// 	} else {
	// 		messageType = "text"
	// 	}
	// }

	// broadcast := NewMessageBroadcast{
	// 	Type:        "new_message",
	// 	MessageID:   job.MessageID,
	// 	RoomID:      job.RoomID,
	// 	SenderID:    job.UserID,
	// 	Content:     job.MessageContent,
	// 	MessageType: messageType,
	// 	Timestamp:   time.Now().Format(time.RFC3339),
	// }

	// if broadcast.Content == "" && len(attachmentList) > 0 {
	// 	if len(attachmentList) == 1 {
	// 		broadcast.Content = "Sent an image"
	// 	} else {
	// 		broadcast.Content = fmt.Sprintf("Sent %d images", len(attachmentList))
	// 	}
	// }

	// if len(attachmentList) > 0 {
	// 	broadcast.Attachments = attachmentList
	// }

	broadcastJSON, err := json.Marshal(messageByID)
	if err != nil {
		log.Printf("[ERR] Gagal marshal broadcast message: %v", err)
		return
	}

	// Use BroadcastToRoomExcept to exclude sender
	d.wsManager.BroadcastToRoomExcept(job.RoomID, broadcastJSON, job.UserID)
	log.Printf("[WS] Broadcast message ke room %s (excludes sender %s)", job.RoomID, job.UserID)
}

func (d *Dispatcher) sendUploadStatusToUser(userID string, roomID string, messageID string, total, failed int, failedFiles []string) {
	var failedFileDetails []FailedFileInfo
	for _, f := range failedFiles {
		var info FailedFileInfo
		if err := json.Unmarshal([]byte(f), &info); err == nil {
			failedFileDetails = append(failedFileDetails, info)
		}
	}

	var status string
	switch failed {
	case 0:
		status = "success"
	case total:
		status = "failed"
	default:
		status = "partial_failed"
	}

	// Build response
	response := UploadBatchResponse{
		Type:         "upload_batch_status",
		Status:       status,
		MessageID:    messageID,
		RoomID:       roomID,
		TotalFiles:   total,
		FailedCount:  failed,
		SuccessCount: total - failed,
	}

	if failed > 0 {
		response.FailedFiles = failedFileDetails
	}

	// Send to sender only
	responseJSON, _ := json.Marshal(response)
	if err := d.wsManager.SendNotificationToUser(userID, responseJSON); err != nil {
		log.Printf("[ERR] Gagal kirim WS status ke user %s: %v", userID, err)
	} else {
		log.Printf("[WS] Kirim upload status ke sender %s: %s", userID, status)
	}
}
