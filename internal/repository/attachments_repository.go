package repository

import (
	"context"
	"database/sql"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type AttachmentsRepository interface {
	RepositoryBased[*models.Attachments]
	FindByMessageID(ctx context.Context, messageID string) ([]*models.Attachments, error)
	FindListGaleryByRoomID(ctx context.Context, messageID string) ([]*models.Attachments, error)
}

type AttachmentsRepositoryImpl struct {
	RepositoryBased[*models.Attachments]
	db *sql.DB
}

func NewAttachmentsRepository(db *sql.DB) AttachmentsRepository {
	return &AttachmentsRepositoryImpl{
		RepositoryBased: NewBaseRepository[*models.Attachments](db).(*BaseRepository[*models.Attachments]),
		db:              db,
	}
}

func (r *AttachmentsRepositoryImpl) FindByMessageID(ctx context.Context, messageID string) ([]*models.Attachments, error) {
	query := `SELECT id, message_id, room_id, url, file_name, file_type, file_size, created_at
			  FROM message_attachments WHERE message_id = $1 ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []*models.Attachments
	for rows.Next() {
		var att models.Attachments
		if err := rows.Scan(&att.ID, &att.MessageID, &att.RoomID, &att.URL, &att.FileName, &att.FileType, &att.FileSize, &att.CreatedAt); err != nil {
			return nil, err
		}
		attachments = append(attachments, &att)
	}

	return attachments, rows.Err()
}

func (r *AttachmentsRepositoryImpl) FindListGaleryByRoomID(ctx context.Context, roomid string) ([]*models.Attachments, error) {
	query := `
		SELECT 
			message_id,
			room_id,
			url,
			file_name,
			file_type,
			file_size,
			created_at
		FROM message_attachments
		WHERE room_id = $1
	`

	rows, err := r.db.QueryContext(ctx, query, roomid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []*models.Attachments
	for rows.Next() {
		var att models.Attachments
		if err := rows.Scan(&att.ID, &att.MessageID, &att.RoomID, &att.URL, &att.FileName, &att.FileType, &att.FileSize, &att.CreatedAt); err != nil {
			return nil, err
		}
		attachments = append(attachments, &att)
	}

	return attachments, rows.Err()
}
