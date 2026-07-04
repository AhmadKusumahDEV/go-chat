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
	FindOneRoomByUserID(ctx context.Context, userID string, roomID string) (*models.Room, error)
	FindMemberRoom(ctx context.Context, roomID string) ([]*models.MemberComposite, error)
	FindRoomDetail(ctx context.Context, roomID string) (*models.RoomDetail, error)
	FindRoomMembers(ctx context.Context, roomID string) ([]models.MemberDetail, error)
	FindRoomName(ctx context.Context, roomID string) (string, error)
	CreateWithMember(ctx context.Context, room *models.Room, members []*models.Members) (uuid.UUID, error)
	CreateRoomDirect(ctx context.Context, room *models.Room, members []*models.Members, msg *models.Message) error
	CheckDirectRoom(ctx context.Context, userId string, userTargetId string) (string, error)
	UpdateProfilePicture(ctx context.Context, roomID, userID, avatarURL string) error
	UpdatedProfileInfo(ctx context.Context, roomID, roomName, description string) error
}

type RepositoryRoomImpl struct {
	*BaseRepository[*models.Room]
	db *sql.DB
}

func (p *RepositoryRoomImpl) CheckDirectRoom(ctx context.Context, userId string, userTargetId string) (string, error) {
	var roomID string
	query := `
		SELECT 
			r.id
		FROM
			rooms r
		JOIN
			room_members rm
			ON r.id = rm.room_id
		JOIN 
			room_members rm2
			ON r.id = rm2.room_id
		WHERE
			r.room_type = 'direct'
			AND rm.user_id = $1
          	AND rm2.user_id = $2
	`
	err := p.db.QueryRowContext(ctx, query, userId, userTargetId).Scan(&roomID)
	if err != nil {
		log.Println(err)
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("tidak ditemukan room direct")
		}
		return "", err
	}

	return roomID, nil
}

// CreateWithMember implements RepositoryRoom with a Database Transaction (ACID)
func (p *RepositoryRoomImpl) CreateWithMember(ctx context.Context, room *models.Room, members []*models.Members) (uuid.UUID, error) {
	v6, err := uuid.NewV6()
	if err != nil {
		return uuid.UUID{}, err
	}
	room.ID = v6

	if err := room.Validate(); err != nil {
		return uuid.UUID{}, err
	}
	for _, member := range members {
		member.Roomid = v6
		if err := member.Validate(); err != nil {
			return uuid.UUID{}, err
		}
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.UUID{}, err
	}
	defer tx.Rollback()

	roomQuery := `INSERT INTO rooms (id, room_name, room_type, description, is_private, avatar_url , created_by) 
					VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx.ExecContext(ctx, roomQuery, room.ID, room.Name, room.Roomtype, room.Description, room.Isprivate, room.AvatarUrl, room.CreatedBy)
	if err != nil {
		return uuid.UUID{}, err
	}

	memberQuery := `INSERT INTO room_members (room_id, user_id, added_by, role) 
					VALUES ($1, $2, $3, $4)`
	for _, member := range members {
		_, err = tx.ExecContext(ctx, memberQuery, member.Roomid, member.Userid, member.AddedBy, member.Role)
		if err != nil {
			return uuid.UUID{}, err
		}
	}

	content := fmt.Sprintf("Room created at %s", room.CreatedAt.Format("2006-01-02"))

	systemMessageQuery := `INSERT INTO messages (room_id, user_id, content, message_type, timestamp) 
							VALUES ($1, $2, $3, $4, $5)`

	_, err = tx.ExecContext(ctx, systemMessageQuery, room.ID, room.CreatedBy, content, "system", room.CreatedAt)
	if err != nil {
		return uuid.UUID{}, errors.New("failed insert system message")
	}

	return v6, tx.Commit()
}

func (p *RepositoryRoomImpl) CreateRoomDirect(ctx context.Context, room *models.Room, members []*models.Members, msg *models.Message) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	roomQuery := `
		INSERT INTO
			rooms
		(
			id,
			room_type,
			is_private,
			created_by
		)
		VALUES
			($1, $2, $3, $4)
	`
	_, err = tx.ExecContext(ctx, roomQuery, room.ID, room.Roomtype, room.Isprivate, room.CreatedBy)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO room_members (room_id, user_id, role, added_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (room_id, user_id) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range members {
		_, err := stmt.ExecContext(ctx, m.Roomid, m.Userid, m.Role, m.AddedBy)
		if err != nil {
			return err
		}
	}

	messageQuery := `INSERT INTO messages (id, room_id, content, message_type, user_id, timestamp) VALUES ($1, $2, $3, $4, $5 , $6)`
	_, err = tx.ExecContext(ctx, messageQuery, msg.ID, room.ID, msg.Content, msg.Type, msg.SenderID, msg.Timestamp)
	if err != nil {
		return err
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
			r.avatar_url,
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
	avatarUrl := sql.NullString{}
	err := p.db.QueryRowContext(ctx, query, roomID).Scan(
		&room.ID,
		&room.Name,
		&room.Description,
		&room.RoomType,
		&avatarUrl,
		&room.IsPrivate,
		&room.CreatedAt,
		&room.CreatedBy,
		&room.MemberCount,
	)

	room.AvatarUrl = fmt.Sprintf("https://api.dicebear.com/10.x/initials/png?seed=%s", room.Name)

	if avatarUrl.Valid {
		room.AvatarUrl = avatarUrl.String
	}

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

func (p *RepositoryRoomImpl) FindOneRoomByUserID(ctx context.Context, userID string, roomID string) (*models.Room, error) {
	query := `
        SELECT 
            r.id, 
            r.room_name, 
            r.description, 
            r.room_type, 
            r.is_private, 
            r.created_by,
            r.created_at,
            r.avatar_url,
            rm2.user_id AS target_user_id,
            u.avatar_url,
            u.username,
            m.id AS last_message_id,
            m.content AS last_message_content,
            m.user_id AS last_message_user_id,
            m.message_type AS last_message_type,
            m.timestamp AS last_message_timestamp
        FROM rooms r
        JOIN room_members rm ON r.id = rm.room_id
        LEFT JOIN 
            room_members rm2
            ON r.id = rm2.room_id
            AND r.room_type = 'direct'
            AND rm2.user_id != rm.user_id
        LEFT JOIN 
            users u
            ON u.id = rm2.user_id
        LEFT JOIN LATERAL (
            SELECT id, content, user_id, message_type, timestamp
            FROM messages
            WHERE room_id = r.id
            ORDER BY timestamp DESC
            LIMIT 1
        ) m ON true
        WHERE rm.user_id = $1
            AND r.id = $2
        ORDER BY m.timestamp DESC NULLS LAST
        LIMIT 1
    `

	row := p.db.QueryRowContext(ctx, query, userID, roomID)

	var room models.Room
	var avatarUrl sql.NullString
	var lastMsgID sql.NullString
	var lastMsgContent sql.NullString
	var lastMsgUserID sql.NullString
	var lastMsgType sql.NullString
	var lastMsgTimestamp sql.NullTime
	var targetUserID sql.NullString
	var targetAvatarUrl sql.NullString
	var username sql.NullString
	var roomName sql.NullString
	var description sql.NullString

	err := row.Scan(
		&room.ID,
		&roomName,
		&description,
		&room.Roomtype,
		&room.Isprivate,
		&room.CreatedBy,
		&room.CreatedAt,
		&avatarUrl,
		&targetUserID,
		&targetAvatarUrl,
		&username,
		&lastMsgID,
		&lastMsgContent,
		&lastMsgUserID,
		&lastMsgType,
		&lastMsgTimestamp,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Handle nullable fields
	if lastMsgType.String == "image" {
		lastMsgContent.String = "Sent Photo"
	}

	if targetUserID.Valid {
		id, _ := uuid.FromString(targetUserID.String)
		room.TargetUserID = &id
		room.TargetUsername = &username.String
		if targetAvatarUrl.Valid {
			room.TargetAvatarUrl = &targetAvatarUrl.String
		}
	}

	if avatarUrl.Valid {
		room.AvatarUrl = &avatarUrl.String
	}

	if roomName.Valid {
		room.Name = roomName.String
	}

	if description.Valid {
		room.Description = description.String
	}

	// Set last message jika ada
	if lastMsgID.Valid {
		var senderID *uuid.UUID
		if lastMsgUserID.Valid {
			uid, _ := uuid.FromString(lastMsgUserID.String)
			senderID = &uid
		}
		msgID, _ := uuid.FromString(lastMsgID.String)
		room.LastMessage = &models.Message{
			ID:        msgID,
			Content:   lastMsgContent.String,
			SenderID:  senderID,
			Type:      lastMsgType.String,
			Timestamp: lastMsgTimestamp.Time,
		}
	}

	return &room, nil
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
			r.avatar_url,
			rm2.user_id AS target_user_id,
			u.avatar_url,
			u.username,
            m.id AS last_message_id,
            m.content AS last_message_content,
            m.user_id AS last_message_user_id,
            m.message_type AS last_message_type,
            m.timestamp AS last_message_timestamp,
			m.username AS last_message_username
        FROM rooms r
        JOIN room_members rm ON r.id = rm.room_id
		LEFT JOIN 
			room_members rm2
			ON r.id = rm2.room_id
			AND r.room_type = 'direct'
			AND rm2.user_id != rm.user_id
		LEFT JOIN 
			users u
			ON u.id = rm2.user_id
        LEFT JOIN LATERAL (
            SELECT msg.id, msg.content, msg.user_id, msg.message_type, msg.timestamp, u_msg.username
            FROM messages msg
			LEFT JOIN users u_msg ON msg.user_id = u_msg.id
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
		var avatarUrl sql.NullString
		var lastMsgID sql.NullString
		var lastMsgContent sql.NullString
		var lastMsgUserID sql.NullString
		var lastMsgType sql.NullString
		var lastMsgTimestamp sql.NullTime
		var lastMessageUsername sql.NullString
		var targetUserID sql.NullString
		var targetAvatarUrl sql.NullString
		var username sql.NullString
		var roomName sql.NullString
		var decsription sql.NullString

		if err := rows.Scan(
			&room.ID,
			&roomName,
			&decsription,
			&room.Roomtype,
			&room.Isprivate,
			&room.CreatedBy,
			&room.CreatedAt,
			&avatarUrl,
			&targetUserID,
			&targetAvatarUrl,
			&username,
			&lastMsgID,
			&lastMsgContent,
			&lastMsgUserID,
			&lastMsgType,
			&lastMsgTimestamp,
			&lastMessageUsername,
		); err != nil {
			return nil, err
		}

		if lastMsgType.String == "image" {
			lastMsgContent.String = "Sent Photo"
		}

		if targetUserID.Valid {
			Id, _ := uuid.FromString(targetUserID.String)
			room.TargetUserID = &Id
			room.TargetUsername = &username.String

			if targetAvatarUrl.Valid {
				room.TargetAvatarUrl = &targetAvatarUrl.String
			}
		}

		if avatarUrl.Valid {
			room.AvatarUrl = &avatarUrl.String
		}

		if roomName.Valid {
			room.Name = roomName.String
		}

		if decsription.Valid {
			room.Description = decsription.String
		}

		// ✅ Set last message jika ada
		if lastMsgID.Valid {
			var userID *uuid.UUID
			var username string
			if lastMsgUserID.Valid {
				uid, _ := uuid.FromString(lastMsgUserID.String)
				userID = &uid
			}

			if lastMessageUsername.Valid {
				username = lastMessageUsername.String
			}

			msgID, _ := uuid.FromString(lastMsgID.String)
			room.LastMessage = &models.Message{
				ID:         msgID,
				Content:    lastMsgContent.String,
				SenderID:   userID,
				SenderName: username,
				Type:       lastMsgType.String,
				Timestamp:  lastMsgTimestamp.Time,
			}
		}

		rooms = append(rooms, &room)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rooms, nil
}

func (p *RepositoryRoomImpl) UpdateProfilePicture(ctx context.Context, roomID, userID, avatarURL string) error {
	query := `UPDATE rooms SET avatar_url = $1 WHERE id = $2`
	result, err := p.db.ExecContext(ctx, query, avatarURL, roomID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("room not found")
	}
	return nil
}

func (p *RepositoryRoomImpl) UpdatedProfileInfo(ctx context.Context, roomID, roomName, description string) error {
	query := `UPDATE rooms SET name = $1, description = $2 WHERE id = $3`
	result, err := p.db.ExecContext(ctx, query, roomName, description, roomID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("room not found")
	}
	return nil
}

var _ RepositoryRoom = (*RepositoryRoomImpl)(nil)
