package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/cahce"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/helpers"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/AhmadKusumahDEV/go-chat/internal/websocket"
	"github.com/AhmadKusumahDEV/go-chat/pkg/storage"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"
)

type RoomService interface {
	CreateRoom(ctx context.Context, req *request.CreateRoomRequest, create_by string) (uuid.UUID, error)
	CheckDirectRoom(ctx context.Context, userID string, targetUserId string) (string, bool, error)
	CreateDirectRoom(ctx context.Context, req *request.CreateDirectRoomRequest, userID string) (uuid.UUID, error)
	GetAllRoomUser(ctx context.Context) ([]*response.RoomResponse, error)
	UpdateRoom(ctx context.Context, roomID string, userID string, req *request.UpdateRoomRequest) error
	DeleteRoom(ctx context.Context, roomID string, deletedBy string) error
	GetRoomByUserID(ctx context.Context, userID string) ([]*response.RoomResponse, error)
	GetSpecificRoomByUserID(ctx context.Context, userID string, roomID string) (*response.RoomResponse, error)
	GetRoomByName(ctx context.Context, room_name request.GetRoomByName) ([]*response.RoomResponse, error)
	GetRoomDetail(ctx context.Context, roomID string, userID string) (*response.RoomDetailResponse, error)
	UpdateAvatar(ctx context.Context, roomID, userID string, reader io.Reader, size int64, contentType, objectName string) (string, error)
}

type RoomServiceImpl struct {
	roomRepository   repository.RepositoryRoom
	memberRepository repository.RepositoryMembers
	attachment       repository.AttachmentsRepository
	cahce            cahce.CahceRedis
	validate         *validator.Validate
	manager          websocket.WebSocketManager
	minioS3          storage.ObjectStorage
}

func NewRoomServices(roomRepository repository.RepositoryRoom, memberRepository repository.RepositoryMembers, att repository.AttachmentsRepository, cahce cahce.CahceRedis, validate *validator.Validate, minio storage.ObjectStorage) RoomService {
	return &RoomServiceImpl{
		roomRepository:   roomRepository,
		memberRepository: memberRepository,
		attachment:       att,
		cahce:            cahce,
		validate:         validate,
		minioS3:          minio,
	}
}

// GetSpecificRoomByUserID implements [RoomService].
func (r *RoomServiceImpl) GetSpecificRoomByUserID(ctx context.Context, userID string, roomID string) (*response.RoomResponse, error) {
	res, err := r.roomRepository.FindOneRoomByUserID(ctx, userID, roomID)
	if err != nil {
		return nil, errors.New("failed to get spesific room")
	}

	return helpers.RoomResponse(res), nil
}

func (r *RoomServiceImpl) CheckDirectRoom(ctx context.Context, userID string, targetUserId string) (string, bool, error) {
	id, err := r.roomRepository.CheckDirectRoom(ctx, userID, targetUserId)
	if err != nil {
		return "", false, errors.New("room tidak di temukan")
	}

	return id, true, nil
}

// CreateDirectRoom implements [RoomService].
func (r *RoomServiceImpl) CreateDirectRoom(ctx context.Context, req *request.CreateDirectRoomRequest, userID string) (uuid.UUID, error) {
	err := r.validate.Struct(req)
	if err != nil {
		log.Println("error on servies layer with name CreateDirectRoom in validate ", err)
		return uuid.UUID{}, errors.New("data yang tidak memenuhi format")
	}

	if req.TargetUserId == userID {
		return uuid.UUID{}, errors.New("tidak bisa membuat room dengan diri sendiri")
	}

	targetId, _ := uuid.FromString(req.TargetUserId)
	userId, _ := uuid.FromString(userID)

	v6, err := uuid.NewV6()
	if err != nil {
		return uuid.UUID{}, errors.New("failed generate uuid")
	}

	idMessageV6, err := uuid.NewV6()
	if err != nil {
		return uuid.UUID{}, errors.New("failed generate uuid")
	}

	roomData := &models.Room{
		ID:        v6,
		Roomtype:  "direct",
		Isprivate: true,
		CreatedBy: userId,
	}

	membersData := []*models.Members{
		{
			Roomid:  v6,
			Userid:  userId,
			AddedBy: userId,
			Role:    "member",
		},
		{
			Roomid:  v6,
			Userid:  targetId,
			AddedBy: userId,
			Role:    "member",
		},
	}

	messageData := &models.Message{
		ID:        idMessageV6,
		RoomID:    v6,
		SenderID:  &userId,
		Content:   req.Content,
		Type:      req.MessageType,
		Timestamp: time.Now(),
	}

	err = r.roomRepository.CreateRoomDirect(ctx, roomData, membersData, messageData)
	if err != nil {
		return uuid.UUID{}, errors.New("gagal menyimpan data")
	}

	return v6, nil
}

// GetRoomByName implements RoomService.
func (r *RoomServiceImpl) GetRoomByName(ctx context.Context, room_name request.GetRoomByName) ([]*response.RoomResponse, error) {
	err := r.validate.Struct(room_name)

	if err != nil {
		log.Println("error on servies layer with name GetRoomByName in validate ", err)
		return nil, err
	}

	model, err := r.roomRepository.FindRoomByName(ctx, room_name.Name)

	if err != nil {
		log.Println("error on servies layer with name GetRoomByName ", err)
		return nil, err
	}

	return helpers.RoomResponses(model), nil
}

// CreateRoom implements RoomService.
func (r *RoomServiceImpl) CreateRoom(ctx context.Context, req *request.CreateRoomRequest, create_by string) (uuid.UUID, error) {
	err := r.validate.Struct(req)

	if err != nil {
		log.Println("error on servies layer with name CreateRoom in validate ", err)
		return uuid.UUID{}, err
	}

	room, err := req.ToModel(create_by)
	if err != nil {
		log.Println("error on servies layer with name CreateRoom when dto convert to model ", err)
		return uuid.UUID{}, err
	}

	// Create a new member profile for the creator as an admin
	creatorUUID, _ := uuid.FromString(create_by)

	members := []*models.Members{
		{
			Userid:  creatorUUID,
			AddedBy: creatorUUID,
			Role:    "admin",
		},
	}

	for _, memberIDStr := range req.Members {
		if memberIDStr == create_by {
			continue
		}

		memberUUID, err := uuid.FromString(memberIDStr)
		if err != nil {
			log.Printf("invalid member uuid provided: %s", memberIDStr)
			continue // or return error, but skipping is more tolerant
		}

		members = append(members, &models.Members{
			Userid:  memberUUID,
			AddedBy: creatorUUID,
			Role:    "member",
		})
	}

	uuidRoom, err := r.roomRepository.CreateWithMember(ctx, room, members)
	if err != nil {
		return uuid.UUID{}, err
	}

	return uuidRoom, nil
}

// DeleteRoom implements RoomService.
func (r *RoomServiceImpl) DeleteRoom(ctx context.Context, roomID string, deletedBy string) error {
	rUUID, err := uuid.FromString(roomID)
	if err != nil || rUUID == uuid.Nil {
		return errors.New("invalid room id format")
	}

	userUUID, err := uuid.FromString(deletedBy)
	if err != nil || userUUID == uuid.Nil {
		return errors.New("invalid user id format")
	}

	room, err := r.roomRepository.FindByID(ctx, rUUID)
	if err != nil {
		return err
	}

	if room.CreatedBy != userUUID {
		return errors.New("forbidden: only the room creator can delete this room")
	}

	member, err := r.memberRepository.FindMember(ctx, roomID, deletedBy)
	if err != nil {
		return errors.New("forbidden: you are not a member of this room")
	}

	if room.CreatedBy != userUUID && member.Role != "admin" {
		return errors.New("forbidden: insufficient permission to delete room")
	}

	err = r.roomRepository.Delete(ctx, rUUID)
	if err != nil {
		return err
	}

	go func() {
		_ = r.cahce.Del(context.Background(), "rooms:%s:members"+roomID)
	}()

	return nil
}

// GetAllRoomUser implements RoomService.
func (r *RoomServiceImpl) GetAllRoomUser(ctx context.Context) ([]*response.RoomResponse, error) {
	model, err := r.roomRepository.FindAll(ctx)
	if err != nil {
		log.Println("error on servies layer with name GetAllRoomUser ", err)
		return nil, err
	}
	return helpers.RoomResponses(model), nil
}

// ListUserRooms implements RoomService.
func (r *RoomServiceImpl) GetRoomByUserID(ctx context.Context, userID string) ([]*response.RoomResponse, error) {
	dataRoom, err := r.roomRepository.FindAllRoomByUserID(ctx, userID)
	if err != nil {
		log.Println("error on servies layer with name GetRoomByUserID ", err)
		return nil, err
	}

	for _, room := range dataRoom {
		if room.TargetAvatarUrl != nil && !ChechkPrefixHttps(*room.TargetAvatarUrl) {
			temp, err := r.minioS3.GetObjectURL(ctx, *room.TargetAvatarUrl, "chat-app")
			if err != nil {
				log.Printf("[ERR] Failed to resolve target avatar: %v", err)
			} else {
				room.TargetAvatarUrl = &temp
			}
		}

		if room.AvatarUrl != nil && !ChechkPrefixHttps(*room.AvatarUrl) {
			temp, err := r.minioS3.GetObjectURL(ctx, *room.AvatarUrl, "chat-app")
			if err != nil {
				log.Printf("[ERR] Failed to resolve avatar: %v", err)
			} else {
				room.AvatarUrl = &temp
			}
		}
	}

	return helpers.RoomResponses(dataRoom), nil
}

// UpdateRoom implements RoomService.
func (r *RoomServiceImpl) UpdateRoom(ctx context.Context, roomID string, userID string, req *request.UpdateRoomRequest) error {
	// RBAC: only admin can update room
	member, err := r.memberRepository.FindMember(ctx, roomID, userID)
	if err != nil {
		return errors.New("forbidden: you are not a member of this room")
	}
	if member.Role != "admin" {
		return errors.New("forbidden: only admin can update room settings")
	}

	_, err = r.roomRepository.FindByID(ctx, roomID)
	if err != nil {
		return errors.New("room not found")
	}

	err = r.roomRepository.UpdatedProfileInfo(ctx, roomID, *req.Name, *req.Description)
	if err != nil {
		return errors.New("terjadi kesalahan saat melakukan updated room")
	}

	return nil
}

// GetRoomDetail implements RoomService - returns full room details with members list
func (r *RoomServiceImpl) GetRoomDetail(ctx context.Context, roomID string, userID string) (*response.RoomDetailResponse, error) {
	// 1. Validate user is member of the room
	_, err := r.memberRepository.FindMember(ctx, roomID, userID)
	if err != nil {
		return nil, errors.New("forbidden: you are not a member of this room")
	}

	// 2. Get room details (includes member count)
	roomDetail, err := r.roomRepository.FindRoomDetail(ctx, roomID)
	if err != nil {
		log.Printf("error on services layer GetRoomDetail: %v", err)
		return nil, err
	}

	if roomDetail.AvatarUrl != "" && !ChechkPrefixHttps(roomDetail.AvatarUrl) {
		temp, err := r.minioS3.GetObjectURL(ctx, roomDetail.AvatarUrl, "chat-app")
		if err != nil {
			log.Printf("[ERR] Failed to resolve avatar: %v", err)
		} else {
			roomDetail.AvatarUrl = temp
		}
	}

	// 3. Get all members with user details
	members, err := r.roomRepository.FindRoomMembers(ctx, roomID)
	if err != nil {
		log.Printf("error on services layer GetRoomDetail (members): %v", err)
		return nil, err
	}

	// 4. Build response
	memberResponses := make([]response.MemberDetailResponse, 0, len(members))
	for _, m := range members {
		var avatarURL *string

		if m.AvatarUrl != nil && !strings.HasPrefix(*m.AvatarUrl, "https://") {
			s3URL, err := r.minioS3.GetObjectURL(ctx, *m.AvatarUrl, "chat-app")
			if err != nil {
				log.Printf("[ERR] Failed to build S3 URL for %s: %v", m.UserID, err)
			} else {
				avatarURL = &s3URL
			}
		} else {
			avatarURL = m.AvatarUrl
		}

		memberResponses = append(memberResponses, response.MemberDetailResponse{
			UserID:    m.UserID.String(),
			Username:  m.Username,
			Email:     m.Email,
			AvatarUrl: avatarURL,
			Role:      m.Role,
			JoinedAt:  m.JoinedAt,
		})
	}

	return &response.RoomDetailResponse{
		ID:          roomDetail.ID.String(),
		Name:        roomDetail.Name,
		Description: roomDetail.Description,
		RoomType:    roomDetail.RoomType,
		Avatar:      roomDetail.AvatarUrl,
		IsPrivate:   roomDetail.IsPrivate,
		CreatedAt:   roomDetail.CreatedAt,
		CreatedBy:   roomDetail.CreatedBy.String(),
		MemberCount: roomDetail.MemberCount,
		Members:     memberResponses,
	}, nil
}

func (r *RoomServiceImpl) UpdateAvatar(ctx context.Context, roomID, userID string, reader io.Reader, size int64, contentType, objectName string) (string, error) {
	member, err := r.memberRepository.FindMember(ctx, roomID, userID)
	if err != nil {
		return "", fmt.Errorf("forbidden: you are not a member of this room: %w", err)
	}
	if member.Role != "admin" {
		return "", errors.New("forbidden: only admin can change room avatar")
	}

	err = r.minioS3.UploadFile(ctx, "chat-app", objectName, reader, size, contentType)
	if err != nil {
		return "", fmt.Errorf("failed to upload to storage: %w", err)
	}

	avatarURL, _ := r.minioS3.GetObjectURL(ctx, objectName, "chat-app")

	err = r.roomRepository.UpdateProfilePicture(ctx, roomID, userID, avatarURL)
	if err != nil {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if rmErr := r.minioS3.DeleteObject(rollbackCtx, "chat-app", objectName); rmErr != nil {
			log.Printf("[ERR] Rollback MinIO failed for room %s: %v", roomID, rmErr)
		}
		return "", fmt.Errorf("failed to update avatar in db: %w", err)
	}

	return avatarURL, nil
}

func ChechkPrefixHttps(s string) bool {
	if strings.HasPrefix(s, "https://") {
		return true
	}
	return false
}
