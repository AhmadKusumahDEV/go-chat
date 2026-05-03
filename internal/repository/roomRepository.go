package repository

import (
	"context"
	"database/sql"
	"log"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gofrs/uuid"
)

type RepositoryRoom interface {
	RepositoryBased[*models.Room]

	FindRoomByName(ctx context.Context, roomName string) ([]*models.Room, error)
	FindAllRoomByUserID(ctx context.Context, userID string) ([]*models.Room, error)
	FindMemberRoom(ctx context.Context, roomID string) ([]*models.MemberComposite, error)
	CreateWithMember(ctx context.Context, room *models.Room, member *models.Members) error
}

type RepositoryRoomImpl struct {
	*BaseRepository[*models.Room]
	db *sql.DB
}

// CreateWithMember implements RepositoryRoom with a Database Transaction (ACID)
func (p *RepositoryRoomImpl) CreateWithMember(ctx context.Context, room *models.Room, member *models.Members) error {
	v6, err := uuid.NewV6()
	if err != nil {
		return err
	}
	room.ID = v6
	member.Roomid = v6

	if err := room.Validate(); err != nil {
		return err
	}
	if err := member.Validate(); err != nil {
		return err
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // Will do nothing if tx.Commit() succeeds

	// 1. Insert Room
	roomQuery := `INSERT INTO rooms (id, name, room_type, description, is_private, created_by) 
					VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.ExecContext(ctx, roomQuery, room.ID, room.Name, room.Roomtype, room.Description, room.Isprivate, room.CreatedBy)
	if err != nil {
		return err
	}

	// 2. Insert Member
	memberQuery := `INSERT INTO room_members (room_id, user_id, added_by, role) 
					VALUES ($1, $2, $3, $4)`
	_, err = tx.ExecContext(ctx, memberQuery, member.Roomid, member.Userid, member.AddedBy, member.Role)
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

		SELECT u.username , rf.role 
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
	query := `SELECT rs.id, rs.room_name, rs.description, rs.room_type, rs.is_private, rs.created_by 
			  FROM rooms rs JOIN room_members rms ON rs.id = rms.room_id WHERE rms.user_id = $1`

	rows, err := p.db.QueryContext(ctx, query, userID)
	if err != nil {
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

var _ RepositoryRoom = (*RepositoryRoomImpl)(nil)
