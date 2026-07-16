package services

import (
	"context"
	"errors"
	"fmt"
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
)

type MemberService interface {
	AddMember(ctx context.Context, member request.AddMemberRequest) error
	GetMembers(ctx context.Context, roomID string) ([]*response.MemberResponse, error)
	LeaveRoom(ctx context.Context, roomID string, userID string) error
	RemoveMember(ctx context.Context, roomID string, userId string, batchTargetUserId request.RemoveMemberRequest) error
	PromoteAdmin(ctx context.Context, roomID, targetUserID, requesterUserID string) error
	DemoteAdmin(ctx context.Context, roomID, targetUserID, requesterUserID string) error
	TransferOwnership(ctx context.Context, roomID, fromUserID, toUserID string) error
}

type MemberServiceImpl struct {
	roomRepository    repository.RepositoryRoom
	memberRepository  repository.RepositoryMembers
	userRepository    repository.RepositoryUser
	messageRepository repository.MessageRepository
	cahce             cahce.CahceRedis
	validate          *validator.Validate
}

func NewMemberServices(
	roomRepository repository.RepositoryRoom,
	memberRepository repository.RepositoryMembers,
	userRepository repository.RepositoryUser,
	messageRepository repository.MessageRepository,
	cahce cahce.CahceRedis,
	validate *validator.Validate,
) MemberService {
	return &MemberServiceImpl{
		roomRepository:    roomRepository,
		memberRepository:  memberRepository,
		userRepository:    userRepository,
		messageRepository: messageRepository,
		cahce:             cahce,
		validate:          validate,
	}
}

func (m *MemberServiceImpl) PromoteAdmin(ctx context.Context, roomID, targetUserID, requesterUserID string) error {
	roomInfo, err := m.roomRepository.FindByID(ctx, roomID)
	if err != nil {
		return errors.New("room not found")
	}

	if roomInfo.Roomtype != "group" {
		return errors.New("only promote in group")
	}

	if targetUserID == requesterUserID {
		return errors.New("canno't promote yourself")
	}

	adder, err := m.memberRepository.FindMember(ctx, roomID, requesterUserID)
	if err != nil {
		return errors.New("forbidden: you are not a member of this room")
	}

	if adder.Role != "admin" {
		return errors.New("forbidden: only admin can promote members")
	}

	target, err := m.memberRepository.FindMember(ctx, roomID, targetUserID)
	if err != nil {
		return errors.New("target user is not a member of this room")
	}

	if target.Role == "admin" {
		return errors.New("target user is already an admin")
	}

	return m.memberRepository.UpdateRole(ctx, roomID, targetUserID, "admin")
}

func (m *MemberServiceImpl) DemoteAdmin(ctx context.Context, roomID, targetUserID, requesterUserID string) error {
	adder, err := m.memberRepository.FindMember(ctx, roomID, requesterUserID)
	if err != nil {
		return errors.New("forbidden: you are not a member of this room")
	}

	if adder.Role != "admin" {
		return errors.New("forbidden: only admin can demote members")
	}

	roomInfo, err := m.roomRepository.FindByID(ctx, roomID)
	if err != nil {
		return errors.New("room not found")
	}

	if roomInfo.CreatedBy == uuid.Nil {
		return errors.New("room has no creator")
	}

	if roomInfo.Roomtype != "group" {
		return errors.New("only demote in group")
	}

	if roomInfo.CreatedBy != adder.Userid {
		return errors.New("only origin admin can demote")
	}

	target, err := m.memberRepository.FindMember(ctx, roomID, targetUserID)
	if err != nil {
		return errors.New("target user is not a member of this room")
	}

	if target.Role != "admin" {
		return errors.New("target user is not an admin")
	}

	if requesterUserID == targetUserID {
		return errors.New("cannot demote yourself, use transfer ownership instead")
	}

	return m.memberRepository.UpdateRole(ctx, roomID, targetUserID, "member")
}

func (m *MemberServiceImpl) TransferOwnership(ctx context.Context, roomID, fromUserID, toUserID string) error {
	room, err := m.roomRepository.FindByID(ctx, roomID)
	if err != nil {
		return errors.New("room not found")
	}
	log.Println(room.CreatedBy)
	log.Println(fromUserID)

	if room.CreatedBy.String() != fromUserID {
		return errors.New("forbidden: only the room creator can transfer ownership")
	}

	target, err := m.memberRepository.FindMember(ctx, roomID, toUserID)
	if err != nil {
		return errors.New("target user is not a member of this room")
	}

	if target.Role != "admin" {
		return errors.New("target user must be admin to receive ownership")
	}

	if fromUserID == toUserID {
		return errors.New("cannot transfer ownership to yourself")
	}

	err = m.memberRepository.TransferRole(ctx, roomID, fromUserID, toUserID)
	if err != nil {
		return errors.New(err.Error())
	}

	if m.cahce != nil {
		_ = m.cahce.Del(ctx, "room:detail:"+roomID)
		_ = m.cahce.Del(ctx, "room:members:"+roomID)
	}

	return nil
}

// AddMember implements MemberService.
func (m *MemberServiceImpl) AddMember(ctx context.Context, member request.AddMemberRequest) error {
	err := m.validate.Struct(member)
	if err != nil {
		log.Println("error on services layer with name AddMember in validate process", err)
		return err
	}

	// 	_, err = s.memberRepo.FindMember(ctx, member.RoomID, senderID)
	// if err != nil {
	// 	return errors.New("forbidden: you are already add this users")
	// }

	room, err := m.roomRepository.FindByID(ctx, member.RoomID)
	if err != nil {
		return errors.New("room not found")
	}

	if room.Isprivate {
		adder, err := m.memberRepository.FindMember(ctx, member.RoomID, member.AddMemberBy)
		if err != nil {
			return errors.New("forbidden: you are not a member of this room")
		}
		if adder.Role != "admin" {
			return errors.New("forbidden: only admin can add members to private rooms")
		}
	}

	addedByUUID, err := uuid.FromString(member.AddMemberBy)
	if err != nil {
		return errors.New("invalid added_by UUID format")
	}

	members := make([]*models.Members, 0, len(member.Members))
	for _, memberIDStr := range member.Members {
		memberUUID, err := uuid.FromString(memberIDStr)
		if err != nil {
			log.Printf("invalid member uuid provided: %s", memberIDStr)
			continue
		}

		members = append(members, &models.Members{
			Roomid:  room.ID,
			Userid:  memberUUID,
			AddedBy: addedByUUID,
			Role:    member.Role,
		})
	}

	if len(members) == 0 {
		return errors.New("no valid members to add")
	}

	// Get names for system message
	addedByName, err := m.getUserName(ctx, member.AddMemberBy)
	if err != nil {
		addedByName = "Someone"
	}

	memberNames, err := m.getMemberNames(ctx, member.Members)
	if err != nil {
		memberNames = fmt.Sprintf("%d member(s)", len(member.Members))
	}

	err = m.memberRepository.CreateBatch(ctx, members)
	if err != nil {
		return err
	}

	content := fmt.Sprintf("%s added %s to the room", addedByName, memberNames)
	err = m.messageRepository.CreateSystemMessage(ctx, room.ID.String(), content)
	if err != nil {
		log.Printf("warning: failed to create system message for add member: %v", err)
	}

	go func() {
		bgctx := context.Background()
		key := "rooms:%s:members" + member.RoomID
		err := m.cahce.Del(bgctx, key)
		if err != nil {
			log.Println("error on services layer with name AddMember when delete cache ", err)
		}
	}()

	return nil
}

func (m *MemberServiceImpl) getUserName(ctx context.Context, userID string) (string, error) {
	user, err := m.userRepository.FindByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return user.Username, nil
}

func (m *MemberServiceImpl) getMemberNames(ctx context.Context, memberIDs []string) (string, error) {
	if len(memberIDs) == 0 {
		return "", nil
	}

	users, err := m.userRepository.FindByIDs(ctx, memberIDs)
	if err != nil {
		return "", err
	}

	if len(users) == 0 {
		return "", nil
	}

	switch len(users) {
	case 1:
		return users[0].Username, nil
	case 2:
		return users[0].Username + " and " + users[1].Username, nil
	default:
		names := make([]string, 0, len(users))
		for _, u := range users {
			names = append(names, u.Username)
		}
		return users[0].Username + " and " + fmt.Sprintf("%d others", len(users)-1), nil
	}
}

// GetMembers implements MemberService.
func (m *MemberServiceImpl) GetMembers(ctx context.Context, roomID string) ([]*response.MemberResponse, error) {
	var member []*response.MemberResponse
	key := "rooms:%s:members" + roomID
	err := m.cahce.Get(ctx, key, &member)

	if err == nil {
		return member, nil
	}

	model, err := m.roomRepository.FindMemberRoom(ctx, roomID)

	if err != nil {
		log.Println("error on servies layer with name GetMembers when get from database ", err)
		return nil, err
	}

	go func() {
		bgctx := context.Background()
		err := m.cahce.Set(bgctx, key, helpers.MemberResponses(model), 120*time.Minute)
		if err != nil {
			log.Println("error on servies layer with name GetMembers when set redis ", err)
		}
	}()

	return helpers.MemberResponses(model), nil
}

// LeaveRoom implements MemberService.
func (m *MemberServiceImpl) LeaveRoom(ctx context.Context, roomID string, userID string) error {
	// Check membership
	member, err := m.memberRepository.FindMember(ctx, roomID, userID)
	if err != nil {
		return errors.New("you are not a member of this room")
	}

	// Prevent sole admin from leaving
	if member.Role == "admin" {
		members, err := m.roomRepository.FindMemberRoom(ctx, roomID)
		if err != nil {
			return err
		}

		if len(members) <= 1 {
			err = m.memberRepository.RemoveMember(ctx, roomID, userID)
			if err != nil {
				return errors.New("failed Leave room")
			}
			return nil
		}

		adminCount := 0
		for _, mem := range members {
			if mem.Role == "admin" {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return errors.New("cannot leave: you are the only admin, transfer ownership first")
		}
	}

	err = m.memberRepository.RemoveMember(ctx, roomID, userID)
	if err != nil {
		return err
	}

	go func() {
		_ = m.cahce.Del(context.Background(), "rooms:%s:members"+roomID)
	}()

	return nil
}

// RemoveMember implements MemberService.
func (m *MemberServiceImpl) RemoveMember(ctx context.Context, roomID string, userId string, batchTargetUserId request.RemoveMemberRequest) error {
	admin, err := m.memberRepository.FindMember(ctx, roomID, userId)
	if err != nil {
		return errors.New("forbidden: you are not a member of this room")
	}

	if admin.Role != "admin" {
		return errors.New("forbidden: only admin can remove members")
	}

	for j := range batchTargetUserId.Members {
		if batchTargetUserId.Members[j] == userId {
			return errors.New("cannot remove yourself")
		}
	}

	sumMemberInfo, err := m.memberRepository.CheckMembers(ctx, roomID, batchTargetUserId.Members)
	if err != nil {
		return err
	}

	if sumMemberInfo.Total == 0 {
		return errors.New("member not found in this room")
	}

	if sumMemberInfo.Total < len(batchTargetUserId.Members) {
		return errors.New("member data yang dikirim terdapat beberapa member tidak ada di room ini")
	}

	if sumMemberInfo.SumAdmin > 0 {
		return errors.New("cannot remove member with role admin")
	}

	_, err = m.memberRepository.RemoveBatchMembers(ctx, roomID, batchTargetUserId.Members)
	if err != nil {
		return errors.New("member not found in this room")
	}

	go func() {
		_ = m.cahce.Del(context.Background(), "rooms:%s:members"+roomID)
	}()

	return nil
}
