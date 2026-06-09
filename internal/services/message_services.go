package services

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"log"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/AhmadKusumahDEV/go-chat/pkg/storage"
)

type MessageService interface {
	GetRoomMessages(ctx context.Context, roomID string, userID string, limit int, cursor *string) (*response.MessageListResponse, error)
	GetMessageByID(ctx context.Context, messageID string) (*response.MessageResponse, error)
	SendMessage(ctx context.Context, req *request.CreateMessageRequest, senderID string) (*response.MessageResponse, error)
	EditMessage(ctx context.Context, messageID string, userID string, req *request.UpdateMessageRequest) error
}

type MessageServicesImpl struct {
	messageRepo repository.MessageRepository
	memberRepo  repository.RepositoryMembers
	client      storage.ObjectStorage
}

func NewMessageServices(messageRepo repository.MessageRepository, memberRepo repository.RepositoryMembers, client storage.ObjectStorage) MessageService {
	return &MessageServicesImpl{
		messageRepo: messageRepo,
		memberRepo:  memberRepo,
		client:      client,
	}
}

// GetRoomMessages returns paginated messages for a room with cursor-based pagination.
func (s *MessageServicesImpl) GetRoomMessages(ctx context.Context, roomID string, userID string, limit int, cursor *string) (*response.MessageListResponse, error) {
	var targetTime time.Time
	_, err := s.memberRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, errors.New("forbidden: you are not a member of this room")
	}

	if limit <= 0 {
		limit = 20
	}

	if cursor != nil && *cursor != "" {
		targetTime, err = decodeCursor(*cursor)
		if err != nil {
			return nil, errors.New("invalid cursor format")
		}
	} else {
		targetTime = time.Now().UTC()
	}

	timeStr := targetTime.Format(time.RFC3339Nano)

	messages, hasMore, err := s.messageRepo.FindMessageByRoomID(ctx, roomID, limit, timeStr)
	if err != nil {
		log.Println("error on services layer GetRoomMessages", err)
		return nil, err
	}

	messageResponses := response.NewMessageResponses(messages)

	for _, msg := range messageResponses {
		if len(msg.Attachments) > 0 {
			for i := range msg.Attachments {
				objectname := msg.Attachments[i]
				url, err := s.client.GetObjectBySignedURL(ctx, "chat-app", objectname, time.Hour*24)
				if err != nil {
					log.Println("error on services layer GetRoomMessages", err)
					return nil, err
				}
				msg.Attachments[i] = url
			}
		}
	}

	var nextCursor *string
	if hasMore && len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		encoded := encodeCursor(lastMsg.Timestamp)
		nextCursor = &encoded
	}

	return &response.MessageListResponse{
		Data:       messageResponses,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}, nil
}

func (s *MessageServicesImpl) SendMessage(ctx context.Context, req *request.CreateMessageRequest, senderID string) (*response.MessageResponse, error) {
	// Verify membership
	_, err := s.memberRepo.FindMember(ctx, req.RoomID, senderID)
	if err != nil {
		return nil, errors.New("forbidden: you are not a member of this room")
	}

	msg, err := req.ToModel(senderID)
	if err != nil {
		return nil, err
	}

	err = s.messageRepo.Create(ctx, msg)
	if err != nil {
		log.Println("error on services layer SendMessage", err)
		return nil, err
	}

	return response.NewMessageResponse(msg), nil
}

// EditMessage updates the content of a message.
// Only the message owner can edit.
func (s *MessageServicesImpl) EditMessage(ctx context.Context, messageID string, userID string, req *request.UpdateMessageRequest) error {
	err := s.messageRepo.UpdateContent(ctx, messageID, userID, req.Content)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("message not found or you are not the owner")
		}
		log.Println("error on services layer EditMessage", err)
		return err
	}

	return nil
}

// decodeCursor decodes a base64 cursor string to timestamp
func decodeCursor(cursor string) (time.Time, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, string(decoded))
}

func encodeCursor(t time.Time) string {
	return base64.StdEncoding.EncodeToString([]byte(t.Format(time.RFC3339Nano)))
}

func (s *MessageServicesImpl) GetMessageByID(ctx context.Context, messageID string) (*response.MessageResponse, error) {
	message, err := s.messageRepo.FindOneMessageByRoomID(ctx, messageID)
	if err != nil {
		return nil, errors.New("gagal mengambil message by id")
	}

	dtoMessage := response.NewMessageResponse(message)

	for i := range dtoMessage.Attachments {
		objectname := dtoMessage.Attachments[i]
		url, err := s.client.GetObjectBySignedURL(ctx, "chat-app", objectname, time.Hour*24)
		if err != nil {
			log.Println("error on services layer GetRoomMessages", err)
			return nil, err
		}
		dtoMessage.Attachments[i] = url
	}

	return dtoMessage, nil
}
