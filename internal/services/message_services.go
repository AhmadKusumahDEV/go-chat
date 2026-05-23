package services

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
)

type MessageService interface {
	GetRoomMessages(ctx context.Context, roomID string, userID string, limit int, cursor *string) (*response.MessageListResponse, error)
	SendMessage(ctx context.Context, req *request.CreateMessageRequest, senderID string) (*response.MessageResponse, error)
	EditMessage(ctx context.Context, messageID string, userID string, req *request.UpdateMessageRequest) error
}

type MessageServicesImpl struct {
	messageRepo repository.MessageRepository
	memberRepo  repository.RepositoryMembers
}

func NewMessageServices(messageRepo repository.MessageRepository, memberRepo repository.RepositoryMembers) MessageService {
	return &MessageServicesImpl{
		messageRepo: messageRepo,
		memberRepo:  memberRepo,
	}
}

// GetRoomMessages returns paginated messages for a room with cursor-based pagination.
func (s *MessageServicesImpl) GetRoomMessages(ctx context.Context, roomID string, userID string, limit int, cursor *string) (*response.MessageListResponse, error) {
	_, err := s.memberRepo.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, errors.New("forbidden: you are not a member of this room")
	}

	if limit <= 0 {
		limit = 20
	}

	messages, hasMore, err := s.messageRepo.FindMessageByRoomID(ctx, roomID, limit, cursor)
	if err != nil {
		log.Println("error on services layer GetRoomMessages", err)
		return nil, err
	}

	messageResponses := response.NewMessageResponses(messages)

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

// encodeCursor encodes a timestamp to base64 for use as cursor
func encodeCursor(t interface{}) string {
	return repository.EncodeCursor(t)
}
