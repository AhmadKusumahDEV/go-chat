package repository

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/lib/pq"
)

type RepositoryMembers interface {
	RepositoryBased[*models.Members]

	CreateBatch(ctx context.Context, members []*models.Members) error
	FindMember(ctx context.Context, roomID string, userID string) (*models.Members, error)
	RemoveMember(ctx context.Context, roomID string, userID string) error
	RemoveBatchMembers(ctx context.Context, roomID string, members []string) (int, error)
	CheckMembers(ctx context.Context, roomID string, members []string) (models.BatchInfoMemberByRoomId, error)
	GetRoomMemberIDs(ctx context.Context, roomID string) ([]string, error)
	UpdateRole(ctx context.Context, roomID, targetID, role string) error
	TransferRole(ctx context.Context, roomID, fromUserID, toUserID string) error
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

func (r *RepositoryMemberImpl) CheckMembers(ctx context.Context, roomID string, members []string) (models.BatchInfoMemberByRoomId, error) {
	var result models.BatchInfoMemberByRoomId

	query := `
	SELECT  
		COUNT(*) FILTER (WHERE rm.role = 'admin') AS total_admin,
		COUNT(*) FILTER (WHERE rm.role = 'member') AS total_member,
		COUNT(*) AS total
	FROM
		users u
	JOIN
		room_members rm ON u.id = rm.user_id 
	WHERE 
		rm.room_id = $1
		AND u.id = ANY($2::uuid[]);
	`

	err := r.db.QueryRowContext(ctx, query, roomID, pq.Array(members)).Scan(
		&result.SumAdmin,
		&result.SumMember,
		&result.Total,
	)

	if err != nil {
		log.Println(err)
		return models.BatchInfoMemberByRoomId{}, errors.New("terjadi kesalahan pengambilan data")
	}

	return result, nil
}

func (r *RepositoryMemberImpl) TransferRole(ctx context.Context, roomID, fromUserID, toUserID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Println(err)
		return errors.New("failed init for save transfer role")
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `
        UPDATE room_members 
        SET role = 'member' 
        WHERE room_id = $1 AND user_id = $2
    `, roomID, fromUserID)
	if err != nil {
		log.Println(err)
		return errors.New("failed update user role to member")
	}

	_, err = tx.ExecContext(ctx, `
        UPDATE room_members 
        SET role = 'admin' 
        WHERE room_id = $1 AND user_id = $2
    `, roomID, toUserID)
	if err != nil {
		log.Println(err)
		return errors.New("failed update user role to admin")
	}

	res, err := tx.ExecContext(ctx, `
        UPDATE rooms 
        SET created_by = $1 
        WHERE id = $2
    `, toUserID, roomID)
	if err != nil {
		log.Println(err)
		return errors.New("failed update room creator")
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("room not found during transfer")
	}

	if err := tx.Commit(); err != nil {
		return errors.New("failed to save data")
	}
	committed = true
	return nil
}

func (r *RepositoryMemberImpl) UpdateRole(ctx context.Context, roomID, targetID, role string) error {
	query := `UPDATE room_members SET role = $1 WHERE room_id = $2 AND user_id = $3`
	_, err := r.db.ExecContext(ctx, query, role, roomID, targetID)
	if err != nil {
		log.Println(err)
		return errors.New("failed promote role")
	}
	return nil
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

// RemoveMember implements RepositoryMembers.
func (r *RepositoryMemberImpl) RemoveBatchMembers(ctx context.Context, roomID string, members []string) (int, error) {
	query := `
    DELETE FROM room_members 
    WHERE room_id = $1 
      AND user_id = ANY($2::uuid[])
    RETURNING user_id;
    `

	// Langsung eksekusi, PostgreSQL menjamin ini atomik
	rows, err := r.db.QueryContext(ctx, query, roomID, pq.Array(members))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var deletedCount int

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		deletedCount++
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	return deletedCount, nil
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
