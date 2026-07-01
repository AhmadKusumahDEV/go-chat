package helpers

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gofrs/uuid"
)

type FieldInfo struct {
	Name     string // Field name in struct
	DBColumn string // Column name in database
	Value    any    // Field value
	IsPK     bool   // Is primary key?
	IsAuto   bool   // Auto-generated (skip on insert)?
}

type scanning interface {
	Scan(dest ...any) error
}

func ExtractFields(entity any) ([]FieldInfo, error) {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			val = reflect.New(val.Type().Elem()).Elem()
		} else {
			val = val.Elem()
		}
	}

	if val.Kind() != reflect.Struct {
		return nil, errors.New("entity must be a struct")
	}

	typ := val.Type()
	var fields []FieldInfo

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue
		}

		parts := strings.Split(dbTag, ",")
		dbColumn := parts[0]

		info := FieldInfo{
			Name:     field.Name,
			DBColumn: dbColumn,
			Value:    fieldValue.Interface(),
			IsPK:     false,
			IsAuto:   false,
		}

		for _, part := range parts[1:] {
			switch part {
			case "pk":
				info.IsPK = true
			case "auto":
				info.IsAuto = true
			}
		}

		// if info.IsPK && info.IsAuto {
		// 	if info.Value == nil || IsEmptyValue(info.Value) {
		// 		continue
		// 	}
		// }

		fields = append(fields, info)
	}

	return fields, nil
}

func ScanRow[T models.Entity](scanner scanning, fields []FieldInfo) (T, error) {
	var zero T

	// Create new instance
	entityType := reflect.TypeOf(zero)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}
	entityValue := reflect.New(entityType)

	// Prepare scan destinations
	var scanDest []any
	for _, field := range fields {
		fieldValue := entityValue.Elem().FieldByName(field.Name)
		if !fieldValue.IsValid() || !fieldValue.CanAddr() {
			continue
		}

		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				fieldValue = reflect.New(fieldValue.Type().Elem())
			}
			scanDest = append(scanDest, fieldValue.Interface())
		} else {
			scanDest = append(scanDest, fieldValue.Addr().Interface())
		}
	}

	if len(scanDest) == 0 {
		return zero, errors.New("no fields to scan")
	}

	if err := scanner.Scan(scanDest...); err != nil {
		return zero, err
	}

	return entityValue.Interface().(T), nil
}

func IsEmptyValue(v any) bool {
	if v == nil {
		return true
	}

	switch val := v.(type) {
	case string:
		return val == ""
	case uuid.UUID:
		return val == uuid.Nil
	default:
		// For pointer types, check if nil
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Ptr {
			return rv.IsNil()
		}
		return false
	}
}

func MemberResponse(member *models.MemberComposite) *response.MemberResponse {
	return &response.MemberResponse{
		UserName: member.Username,
		Role:     member.Role,
		UserID:   member.UserID,
		Avatar:   member.Avatar,
	}
}

func MemberResponses(member []*models.MemberComposite) []*response.MemberResponse {
	var members []*response.MemberResponse
	for _, r := range member {
		members = append(members, MemberResponse(r))
	}
	return members
}
func RoomResponses(rooms []*models.Room) []*response.RoomResponse {
	if rooms == nil {
		return []*response.RoomResponse{}
	}

	result := make([]*response.RoomResponse, 0, len(rooms))
	for _, r := range rooms {
		result = append(result, RoomResponse(r))
	}
	return result
}

// RoomResponse converts single Room model to RoomResponse DTO
func RoomResponse(room *models.Room) *response.RoomResponse {
	resp := &response.RoomResponse{
		ID:             room.ID.String(),
		Name:           room.Name,
		Description:    room.Description,
		RoomType:       room.Roomtype,
		IsPrivate:      room.Isprivate,
		CreatedAt:      room.CreatedAt,
		TargetUsername: room.TargetUsername,
	}

	if room.TargetUserID != nil {
		targetIDStr := room.TargetUserID.String()
		resp.TargetUserID = &targetIDStr
	}

	if room.AvatarUrl != nil {
		resp.AvatarUrl = room.AvatarUrl
	}

	if room.Roomtype == "direct" {
		resp.TargetAvatarUrl = room.TargetAvatarUrl
	}

	if room.LastMessage != nil {
		var userID *string
		var username string
		if room.LastMessage.SenderID != nil {
			uid := room.LastMessage.SenderID.String()
			userID = &uid
		}

		if room.LastMessage.SenderName != "" {
			username = room.LastMessage.SenderName
		}

		resp.LastMessage = &response.LastMessageResponse{
			ID:          room.LastMessage.ID.String(),
			Content:     room.LastMessage.Content,
			UserID:      userID,
			UserName:    &username,
			MessageType: room.LastMessage.Type,
			Timestamp:   room.LastMessage.Timestamp,
		}
	}

	return resp
}

func UserResponses(user []*models.Users) []*response.UserResponse {
	var rooms []*response.UserResponse
	for _, r := range user {
		rooms = append(rooms, UserResponse(r))
	}
	return rooms
}

func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func UserResponse(user *models.Users) *response.UserResponse {
	return &response.UserResponse{
		ID:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		About:     nullStringToPtr(user.About),
		CreatedAt: user.CreatedAt,
		AvatarUrl: nullStringToPtr(user.AvatarUrl),
	}
}

func BuildErrorRedirectURL(errorCode, errorMessage string) string {
	params := url.Values{}
	params.Set("error", errorCode)
	params.Set("message", errorMessage)
	params.Set("redirect", "/(auth)/login")

	return fmt.Sprintf(
		"chatapp://auth/callback?%s",
		params.Encode(),
	)
}

func BuildSuccessRedirectURL(result *models.OauthResult) string {
	params := url.Values{}
	params.Set("token", result.Accesstoken)
	params.Set("refresh_token", result.RefreshToken)
	params.Set("redirect", result.RedirectDeepLink)

	return fmt.Sprintf(
		"%s://auth/callback?%s",
		result.AppScheme,
		params.Encode(),
	)
}

func HashPassword(plainPassword string) string {
	h := sha256.New()
	h.Write([]byte(plainPassword))
	hash := h.Sum(nil)
	return fmt.Sprintf("%x", hash)
}

func ValidatePassword(plainPassword, hashedPassword string) bool {
	return HashPassword(plainPassword) == hashedPassword
}

// func ScanRow[T models.Entity](scanner interface{ Scan(...any) error }, fields []FieldInfo) (T, error) {
// 	var zero T

// 	// Create new instance
// 	entityType := reflect.TypeOf(zero)
// 	if entityType.Kind() == reflect.Ptr {
// 		entityType = entityType.Elem()
// 	}
// 	entityValue := reflect.New(entityType)

// 	// Prepare scan destinations
// 	var scanDest []any
// 	for _, field := range fields {
// 		fieldValue := entityValue.Elem().FieldByName(field.Name)
// 		if fieldValue.IsValid() && fieldValue.CanAddr() {
// 			scanDest = append(scanDest, fieldValue.Addr().Interface())
// 		}
// 	}

// 	// Scan
// 	if err := scanner.Scan(scanDest...); err != nil {
// 		return zero, err
// 	}

// 	return entityValue.Interface().(T), nil
// }
