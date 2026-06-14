package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

type RepositoryRoom interface {
	RepositoryBased[*models.Room]

	FindRoomByName(ctx context.Context, roomName string) ([]*models.Room, error)
	FindAllRoomByUserID(ctx context.Context, userID string) ([]*models.Room, error)
	FindMemberRoom(ctx context.Context, roomID string) ([]*models.MemberComposite, error)
	FindRoomDetail(ctx context.Context, roomID string) (*models.RoomDetail, error)
	FindRoomMembers(ctx context.Context, roomID string) ([]models.MemberDetail, error)
	FindRoomName(ctx context.Context, roomID string) (string, error)
	CreateWithMember(ctx context.Context, room *models.Room, members []*models.Members) error
}

type RepositoryRoomImpl struct {
	*BaseRepository[*models.Room]
	db *sql.DB
}

// CreateWithMember implements RepositoryRoom with a Database Transaction (ACID)
func (p *RepositoryRoomImpl) CreateWithMember(ctx context.Context, room *models.Room, members []*models.Members) error {
	v6, err := uuid.NewV6()
	if err != nil {
		return err
	}
	room.ID = v6

	if err := room.Validate(); err != nil {
		return err
	}
	for _, member := range members {
		member.Roomid = v6
		if err := member.Validate(); err != nil {
			return err
		}
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	roomQuery := `INSERT INTO rooms (id, room_name, room_type, description, is_private, created_by) 
					VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.ExecContext(ctx, roomQuery, room.ID, room.Name, room.Roomtype, room.Description, room.Isprivate, room.CreatedBy)
	if err != nil {
		return err
	}

	memberQuery := `INSERT INTO room_members (room_id, user_id, added_by, role) 
					VALUES ($1, $2, $3, $4)`
	for _, member := range members {
		_, err = tx.ExecContext(ctx, memberQuery, member.Roomid, member.Userid, member.AddedBy, member.Role)
		if err != nil {
			return err
		}
	}

	content := fmt.Sprintf("Room created at %s", room.CreatedAt.Format("2006-01-02"))

	systemMessageQuery := `INSERT INTO messages (room_id, user_id, content, message_type, timestamp) 
							VALUES ($1, $2, $3, $4, $5)`

	_, err = tx.ExecContext(ctx, systemMessageQuery, room.ID, room.CreatedBy, content, "system", room.CreatedAt)
	if err != nil {
		return errors.New("failed insert system message")
	}

	return tx.Commit()
}

// FindMemberRoom implements RepositoryRoom.
func (p *RepositoryRoomImpl) FindMemberRoom(ctx context.Context, roomID string) ([]*models.MemberComposite, error) {
	query := `
		WITH room_filter AS (
		select user_id , role
		from room_members
		where room_id = $1
		)

		SELECT u.username , rf.role , rf.user_id , u.avatar_url
		from room_filter rf
		join users u ON rf.user_id = u.id
	`

	rows, err := p.db.QueryContext(ctx, query, roomID)
	if err != nil {
		log.Println("err level database ", err)
		return nil, err
	}

	defer rows.Close()

	var members []*models.MemberComposite
	for rows.Next() {
		var member models.MemberComposite
		if err := rows.Scan(
			&member.Username,
			&member.Role,
			&member.UserID,
			&member.Avatar,
		); err != nil {
			return nil, err
		}
		members = append(members, &member)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return members, nil
}

// FindRoomDetail implements RepositoryRoom - returns full room details with member count
func (p *RepositoryRoomImpl) FindRoomDetail(ctx context.Context, roomID string) (*models.RoomDetail, error) {
	query := `
		SELECT
			r.id,
			r.room_name,
			r.description,
			r.room_type,
			r.is_private,
			r.created_at,
			r.created_by,
			COUNT(rm.user_id)::int AS member_count
		FROM rooms r
		LEFT JOIN room_members rm ON r.id = rm.room_id
		WHERE r.id = $1
		GROUP BY r.id
	`

	var room models.RoomDetail
	err := p.db.QueryRowContext(ctx, query, roomID).Scan(
		&room.ID,
		&room.Name,
		&room.Description,
		&room.RoomType,
		&room.IsPrivate,
		&room.CreatedAt,
		&room.CreatedBy,
		&room.MemberCount,
	)
	if err != nil {
		return nil, err
	}

	return &room, nil
}

// FindRoomMembers implements RepositoryRoom - returns all members with user details
func (p *RepositoryRoomImpl) FindRoomMembers(ctx context.Context, roomID string) ([]models.MemberDetail, error) {
	query := `
		SELECT
			u.id,
			u.username,
			u.email,
			u.avatar_url,
			rm.role,
			rm.joined_at
		FROM room_members rm
		JOIN users u ON rm.user_id = u.id
		WHERE rm.room_id = $1
		ORDER BY rm.joined_at ASC
	`

	rows, err := p.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.MemberDetail
	for rows.Next() {
		var member models.MemberDetail
		if err := rows.Scan(
			&member.UserID,
			&member.Username,
			&member.Email,
			&member.AvatarUrl,
			&member.Role,
			&member.JoinedAt,
		); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return members, nil
}

// FindRoomName implements RepositoryRoom - returns room name by room ID
func (p *RepositoryRoomImpl) FindRoomName(ctx context.Context, roomID string) (string, error) {
	query := `SELECT room_name FROM rooms WHERE id = $1`
	var roomName string
	err := p.db.QueryRowContext(ctx, query, roomID).Scan(&roomName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("room not found: %s", roomID)
		}
		return "", err
	}
	return roomName, nil
}

func NewRoomRepository(db *sql.DB) RepositoryRoom {
	return &RepositoryRoomImpl{
		BaseRepository: NewBaseRepository[*models.Room](db).(*BaseRepository[*models.Room]),
		db:             db,
	}
}

// FindByRoomName implements RepositoryRoom.
func (p *RepositoryRoomImpl) FindRoomByName(ctx context.Context, roomName string) ([]*models.Room, error) {
	query := "SELECT id, room_name, description, room_type, is_private, created_by FROM rooms WHERE room_name ILIKE $1"
	keysearch := "%" + roomName + "%"

	rows, err := p.db.QueryContext(ctx, query, keysearch)
	if err != nil {
		log.Println("err level database ", err)
		return nil, err
	}
	defer rows.Close()

	var rooms []*models.Room
	for rows.Next() {
		var room models.Room
		if err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.Description,
			&room.Roomtype,
			&room.Isprivate,
			&room.CreatedBy,
		); err != nil {
			return nil, err
		}
		rooms = append(rooms, &room)
	}

	return rooms, nil
}

// FindAllRoomByUserID implements RepositoryRoom.
func (p *RepositoryRoomImpl) FindAllRoomByUserID(ctx context.Context, userID string) ([]*models.Room, error) {
	query := `
        SELECT 
            r.id, 
            r.room_name, 
            r.description, 
            r.room_type, 
            r.is_private, 
            r.created_by,
            r.created_at,
            m.id AS last_message_id,
            m.content AS last_message_content,
            m.user_id AS last_message_user_id,
            m.message_type AS last_message_type,
            m.timestamp AS last_message_timestamp
        FROM rooms r
        JOIN room_members rm ON r.id = rm.room_id
        LEFT JOIN LATERAL (
            SELECT id, content, user_id, message_type, timestamp
            FROM messages
            WHERE room_id = r.id
            ORDER BY timestamp DESC
            LIMIT 1
        ) m ON true
        WHERE rm.user_id = $1
        ORDER BY m.timestamp DESC NULLS LAST
    `

	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make([]*models.Room, 0)

	for rows.Next() {
		var room models.Room
		var lastMsgID sql.NullString
		var lastMsgContent sql.NullString
		var lastMsgUserID sql.NullString
		var lastMsgType sql.NullString
		var lastMsgTimestamp sql.NullTime

		if err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.Description,
			&room.Roomtype,
			&room.Isprivate,
			&room.CreatedBy,
			&room.CreatedAt,
			&lastMsgID,
			&lastMsgContent,
			&lastMsgUserID,
			&lastMsgType,
			&lastMsgTimestamp,
		); err != nil {
			return nil, err
		}

		if lastMsgType.String == "image" {
			lastMsgContent.String = "Sent Photo"
		}

		// ✅ Set last message jika ada
		if lastMsgID.Valid {
			var userID *uuid.UUID
			if lastMsgUserID.Valid {
				uid, _ := uuid.FromString(lastMsgUserID.String)
				userID = &uid
			}

			msgID, _ := uuid.FromString(lastMsgID.String)
			room.LastMessage = &models.Message{
				ID:        msgID,
				Content:   lastMsgContent.String,
				SenderID:  userID,
				Type:      lastMsgType.String,
				Timestamp: lastMsgTimestamp.Time,
			}
		}

		rooms = append(rooms, &room)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rooms, nil
}

var _ RepositoryRoom = (*RepositoryRoomImpl)(nil)
