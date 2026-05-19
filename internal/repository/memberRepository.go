package repository

import (
	"context"
	"database/sql"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type RepositoryMembers interface {
	RepositoryBased[*models.Members]

	CreateBatch(ctx context.Context, members []*models.Members) error
	FindMember(ctx context.Context, roomID string, userID string) (*models.Members, error)
	RemoveMember(ctx context.Context, roomID string, userID string) error
	GetRoomMemberIDs(ctx context.Context, roomID string) ([]string, error)
}

type RepositoryMemberImpl struct {
	*BaseRepository[*models.Members]
	db *sql.DB
}

func NewMemberRepository(db *sql.DB) RepositoryMembers {
	return &RepositoryMemberImpl{
		BaseRepository: NewBaseRepository[*models.Members](db).(*BaseRepository[*models.Members]),
		db:             db,
	}
}

// FindMember implements RepositoryMembers.
func (r *RepositoryMemberImpl) FindMember(ctx context.Context, roomID string, userID string) (*models.Members, error) {
	query := `SELECT room_id, user_id, added_by, role, joined_at FROM room_members WHERE room_id = $1 AND user_id = $2`
	var member models.Members

	err := r.db.QueryRowContext(ctx, query, roomID, userID).Scan(
		&member.Roomid,
		&member.Userid,
		&member.AddedBy,
		&member.Role,
		&member.JoinAt,
	)
	if err != nil {
		return nil, err
	}

	return &member, nil
}

// CreateBatch implements RepositoryMembers.
func (r *RepositoryMemberImpl) CreateBatch(ctx context.Context, members []*models.Members) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO room_members (room_id, user_id, added_by, role, joined_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (room_id, user_id) DO NOTHING
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range members {
		_, err := stmt.ExecContext(ctx, m.Roomid, m.Userid, m.AddedBy, m.Role)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// RemoveMember implements RepositoryMembers.
func (r *RepositoryMemberImpl) RemoveMember(ctx context.Context, roomID string, userID string) error {
	query := `DELETE FROM room_members WHERE room_id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, roomID, userID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// GetRoomMemberIDs implements RepositoryMembers.
func (r *RepositoryMemberImpl) GetRoomMemberIDs(ctx context.Context, roomID string) ([]string, error) {
	query := `SELECT user_id FROM room_members WHERE room_id = $1`
	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userIDs, nil
}

var _ RepositoryMembers = (*RepositoryMemberImpl)(nil)
