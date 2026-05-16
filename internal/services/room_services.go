package services

import (
	"context"
	"errors"
	"log"

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
	CreateRoom(ctx context.Context, req *request.CreateRoomRequest, create_by string) error
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
	cahce            cahce.CahceRedis
	validate         *validator.Validate
	manager          websocket.WebSocketManager
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
func (r *RoomServiceImpl) CreateRoom(ctx context.Context, req *request.CreateRoomRequest, create_by string) error {
	err := r.validate.Struct(req)

	if err != nil {
		log.Println("error on servies layer with name CreateRoom in validate ", err)
		return err
	}

	room, err := req.ToModel(create_by)

	if err != nil {
		log.Println("error on servies layer with name CreateRoom when dto convert to model ", err)
		return err
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

	err = r.roomRepository.CreateWithMember(ctx, room, members)

	if err != nil {
		return err
	}

	return nil
}

// DeleteRoom implements RoomService.
func (r *RoomServiceImpl) DeleteRoom(ctx context.Context, roomID string, deletedBy string) error {
	// 1. VALIDASI FORMAT UUID
	rUUID, err := uuid.FromString(roomID)
	if err != nil || rUUID == uuid.Nil {
		return errors.New("invalid room id format")
	}

	userUUID, err := uuid.FromString(deletedBy)
	if err != nil || userUUID == uuid.Nil {
		return errors.New("invalid user id format")
	}

	// 2. AMBIL DATA ROOM (EXISTENCE CHECK)
	room, err := r.roomRepository.FindByID(ctx, rUUID)
	if err != nil {
		// Asumsi repository return error specific jika tidak ketemu
		return err
	}

	// 3. AUTHORIZATION CHECK 🔒
	// Opsi A: Hanya Creator yang boleh hapus (Paling Aman)
	if room.CreatedBy != userUUID {
		return errors.New("forbidden: only the room creator can delete this room")
	}

	/* //
	   member, err := r.memberRepository.FindMember(ctx, rUUID, userUUID)
	   if err != nil {
	       return errors.New("forbidden: you are not a member of this room")
	   }

	   if room.CreatedBy != userUUID && member.Role != "admin" {
	       return errors.New("forbidden: insufficient permission to delete room")
	   }
	*/

	// 4. EKSEKUSI DELETE
	//    because room_members have  ON DELETE CASCADE
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

func NewRoomServices(roomRepository repository.RepositoryRoom, memberRepository repository.RepositoryMembers, cahce cahce.CahceRedis, validate *validator.Validate) RoomService {
	return &RoomServiceImpl{
		roomRepository:   roomRepository,
		memberRepository: memberRepository,
		cahce:            cahce,
		validate:         validate,
	}
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
