package services

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/cahce"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/request"
	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/helpers"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/AhmadKusumahDEV/go-chat/internal/repository"
	"github.com/AhmadKusumahDEV/go-chat/internal/websocket"
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
	GetRoomByName(ctx context.Context, room_name request.GetRoomByName) ([]*response.RoomResponse, error)
	GetRoomDetail(ctx context.Context, roomID string, userID string) (*response.RoomDetailResponse, error)
}

type RoomServiceImpl struct {
	roomRepository   repository.RepositoryRoom
	memberRepository repository.RepositoryMembers
	attachment       repository.AttachmentsRepository
	cahce            cahce.CahceRedis
	validate         *validator.Validate
	manager          websocket.WebSocketManager
}

func NewRoomServices(roomRepository repository.RepositoryRoom, memberRepository repository.RepositoryMembers, att repository.AttachmentsRepository, cahce cahce.CahceRedis, validate *validator.Validate) RoomService {
	return &RoomServiceImpl{
		roomRepository:   roomRepository,
		memberRepository: memberRepository,
		attachment:       att,
		cahce:            cahce,
		validate:         validate,
	}
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
	model, err := r.roomRepository.FindAllRoomByUserID(ctx, userID)

	if err != nil {
		log.Println("error on servies layer with name GetRoomByUserID ", err)
		return nil, err
	}

	return helpers.RoomResponses(model), nil
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

	existingRoom, err := r.roomRepository.FindByID(ctx, roomID)
	if err != nil {
		return err
	}

	if req.Name != nil && *req.Name != "" {
		existingRoom.Name = *req.Name
	}

	if req.Description != nil && *req.Description != "" {
		existingRoom.Description = *req.Description
	}

	if err := r.roomRepository.Update(ctx, existingRoom); err != nil {
		return err
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

	// 3. Get all members with user details
	members, err := r.roomRepository.FindRoomMembers(ctx, roomID)
	if err != nil {
		log.Printf("error on services layer GetRoomDetail (members): %v", err)
		return nil, err
	}

	// 4. Build response
	memberResponses := make([]response.MemberDetailResponse, 0, len(members))
	for _, m := range members {
		memberResponses = append(memberResponses, response.MemberDetailResponse{
			UserID:    m.UserID.String(),
			Username:  m.Username,
			Email:     m.Email,
			AvatarUrl: m.AvatarUrl,
			Role:      m.Role,
			JoinedAt:  m.JoinedAt,
		})
	}

	return &response.RoomDetailResponse{
		ID:          roomDetail.ID.String(),
		Name:        roomDetail.Name,
		Description: roomDetail.Description,
		RoomType:    roomDetail.RoomType,
		IsPrivate:   roomDetail.IsPrivate,
		CreatedAt:   roomDetail.CreatedAt,
		CreatedBy:   roomDetail.CreatedBy.String(),
		MemberCount: roomDetail.MemberCount,
		Members:     memberResponses,
	}, nil
}
