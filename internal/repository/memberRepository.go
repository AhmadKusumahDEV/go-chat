package repository

import (
	"context"
	"database/sql"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type RepositoryMembers interface {
	RepositoryBased[*models.Members]

	FindMember(ctx context.Context, roomID string, userID string) (*models.Members, error)
	RemoveMember(ctx context.Context, roomID string, userID string) error
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

var _ RepositoryMembers = (*RepositoryMemberImpl)(nil)
