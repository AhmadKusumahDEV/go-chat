package helpers

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/AhmadKusumahDEV/go-chat/internal/dto/response"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
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

	// Get actual struct (handle pointer)
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

		// Get db tag
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue // Skip fields without db tag
		}

		// Parse tag: "id,pk,auto" or "name" or "created_at,auto"
		parts := strings.Split(dbTag, ",")
		dbColumn := parts[0]

		info := FieldInfo{
			Name:     field.Name,
			DBColumn: dbColumn,
			Value:    fieldValue.Interface(),
			IsPK:     false,
			IsAuto:   false,
		}

		// Check for pk and auto flags
		for _, part := range parts[1:] {
			switch part {
			case "pk":
				info.IsPK = true
			case "auto":
				info.IsAuto = true
			}
		}

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
		if fieldValue.IsValid() && fieldValue.CanAddr() {
			scanDest = append(scanDest, fieldValue.Addr().Interface())
		}
	}

	// Scan
	if err := scanner.Scan(scanDest...); err != nil {
		return zero, err
	}

	return entityValue.Interface().(T), nil
}

func MemberResponse(member *models.MemberComposite) *response.MemberResponse {
	return &response.MemberResponse{
		UserName: member.Username,
		Role:     member.Role,
	}
}

func MemberResponses(member []*models.MemberComposite) []*response.MemberResponse {
	var members []*response.MemberResponse
	for _, r := range member {
		members = append(members, MemberResponse(r))
	}
	return members
}
func RoomResponse(room *models.Room) *response.RoomResponse {
	return &response.RoomResponse{
		ID:          room.ID.String(),
		Name:        room.Name,
		Description: room.Description,
		RoomType:    room.Roomtype,
		IsPrivate:   room.Isprivate,
		CreatedAt:   room.CreatedAt,
	}
}

func RoomResponses(room []*models.Room) []*response.RoomResponse {
	var rooms []*response.RoomResponse
	for _, r := range room {
		rooms = append(rooms, RoomResponse(r))
	}
	return rooms
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
