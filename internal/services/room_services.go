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
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
)

type RoomService interface {
	CreateRoom(ctx context.Context, req *request.CreateRoomRequest, create_by string) error
	GetAllRoomUser(ctx context.Context) ([]*response.RoomResponse, error)
	UpdateRoom(ctx context.Context, roomID string, userID string, req *request.UpdateRoomRequest) error
	DeleteRoom(ctx context.Context, roomID string, deletedBy string) error
	GetRoomByUserID(ctx context.Context, userID string) ([]*response.RoomResponse, error)
	GetRoomByName(ctx context.Context, room_name request.GetRoomByName) ([]*response.RoomResponse, error)
	AddRoomMember(ctx context.Context, member request.AddMember) error
	GetRoomMembers(ctx context.Context, roomID string) ([]*response.MemberResponse, error)
	LeaveRoom(ctx context.Context, roomID string, userID string) error
	RemoveRoomMember(ctx context.Context, roomID string, targetUserID string, removedByUserID string) error
}

type RoomServiceImpl struct {
	roomRepository   repository.RepositoryRoom
	memberRepository repository.RepositoryMembers
	cahce            cahce.CahceRedis
	validate         *validator.Validate
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

// AddRoomMember implements RoomService.
func (r *RoomServiceImpl) AddRoomMember(ctx context.Context, member request.AddMember) error {
	err := r.validate.Struct(member)
	if err != nil {
		log.Println("error on services layer with name AddRoomMember in validate process", err)
		return err
	}

	// Check if room is private — only admin can add members
	room, err := r.roomRepository.FindByID(ctx, member.RoomID)
	if err != nil {
		return errors.New("room not found")
	}

	if room.Isprivate {
		adder, err := r.memberRepository.FindMember(ctx, member.RoomID, member.AddMemberBy)
		if err != nil {
			return errors.New("forbidden: you are not a member of this room")
		}
		if adder.Role != "admin" {
			return errors.New("forbidden: only admin can add members to private rooms")
		}
	}

	members, err := member.ToModel()
	if err != nil {
		return err
	}

	err = r.memberRepository.Create(ctx, members)
	if err != nil {
		return err
	}

	go func() {
		bgctx := context.Background()
		key := "rooms:%s:members" + member.RoomID
		err := r.cahce.Del(bgctx, key)
		if err != nil {
			log.Println("error on services layer with name AddRoomMember when delete cache ", err)
		}
	}()

	return nil
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
	member := &models.Members{
		Userid:  creatorUUID,
		AddedBy: creatorUUID,
		Role:    "admin",
	}

	err = r.roomRepository.CreateWithMember(ctx, room, member)

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

// GetRoomMembers implements RoomService.
func (r *RoomServiceImpl) GetRoomMembers(ctx context.Context, roomID string) ([]*response.MemberResponse, error) {
	var member []*response.MemberResponse
	key := "rooms:%s:members" + roomID
	err := r.cahce.Get(ctx, key, &member)

	if err == nil {
		return member, nil
	}

	model, err := r.roomRepository.FindMemberRoom(ctx, roomID)

	if err != nil {
		log.Println("error on servies layer with name GetRoomMembers when get from database ", err)
		return nil, err
	}

	go func() {
		bgctx := context.Background()
		err := r.cahce.Set(bgctx, key, helpers.MemberResponses(model), 120*time.Minute)
		if err != nil {
			log.Println("error on servies layer with name GetRoomMembers when set redis ", err)
		}
	}()

	return helpers.MemberResponses(model), nil
}

// ListUserRooms implements RoomService.
func (r *RoomServiceImpl) GetRoomByUserID(ctx context.Context, userID string) ([]*response.RoomResponse, error) {
	var rooms []*response.RoomResponse
	key := "rooms:userid:" + userID
	err := r.cahce.Get(ctx, key, &rooms)

	if err != nil {
		if err == redis.Nil {
			log.Println("[CACHE MISS] Data belum ada, lanjut query ke DB.")
		} else {
			log.Printf("[CACHE ERROR] Gawat! Redis Error: %v", err)
		}
	} else {
		return rooms, nil
	}

	model, err := r.roomRepository.FindAllRoomByUserID(ctx, userID)

	if err != nil {
		log.Println("error on servies layer with name GetRoomByUserID ", err)
		return nil, err
	}

	go func() {
		bgctx := context.Background()
		err := r.cahce.Set(bgctx, key, helpers.RoomResponses(model), 15*time.Minute)

		if err != nil {
			log.Println("error on servies layer with name GetRoomByUserID when set redis ", err)
		}
	}()

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

// LeaveRoom implements RoomService.
func (r *RoomServiceImpl) LeaveRoom(ctx context.Context, roomID string, userID string) error {
	// Check membership
	member, err := r.memberRepository.FindMember(ctx, roomID, userID)
	if err != nil {
		return errors.New("you are not a member of this room")
	}

	// Prevent sole admin from leaving
	if member.Role == "admin" {
		members, err := r.roomRepository.FindMemberRoom(ctx, roomID)
		if err != nil {
			return err
		}

		adminCount := 0
		for _, m := range members {
			if m.Role == "admin" {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return errors.New("cannot leave: you are the only admin, transfer ownership first")
		}
	}

	err = r.memberRepository.RemoveMember(ctx, roomID, userID)
	if err != nil {
		return err
	}

	go func() {
		_ = r.cahce.Del(context.Background(), "rooms:%s:members"+roomID)
	}()

	return nil
}

// RemoveRoomMember implements RoomService.
func (r *RoomServiceImpl) RemoveRoomMember(ctx context.Context, roomID string, targetUserID string, removedByUserID string) error {
	// RBAC: only admin can remove members
	admin, err := r.memberRepository.FindMember(ctx, roomID, removedByUserID)
	if err != nil {
		return errors.New("forbidden: you are not a member of this room")
	}
	if admin.Role != "admin" {
		return errors.New("forbidden: only admin can remove members")
	}

	// Cannot remove yourself via this endpoint
	if targetUserID == removedByUserID {
		return errors.New("use leave endpoint to remove yourself")
	}

	err = r.memberRepository.RemoveMember(ctx, roomID, targetUserID)
	if err != nil {
		return errors.New("member not found in this room")
	}

	go func() {
		_ = r.cahce.Del(context.Background(), "rooms:%s:members"+roomID)
	}()

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
