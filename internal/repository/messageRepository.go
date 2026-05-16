package repository

import (
	"context"
	"database/sql"
	"log"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type MessageRepository interface {
	RepositoryBased[*models.Message]

	FindMessageByRoomID(ctx context.Context, roomID string, limit int) ([]*models.Message, error)
	UpdateContent(ctx context.Context, messageID string, userID string, newContent string) error
}

type RepositoryMessageImpl struct {
	*BaseRepository[*models.Message]
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) MessageRepository {
	return &RepositoryMessageImpl{
		BaseRepository: NewBaseRepository[*models.Message](db).(*BaseRepository[*models.Message]),
		db:             db,
	}
}

// FindMessageByRoomID returns the latest messages for a room, ordered newest first.
func (r *RepositoryMessageImpl) FindMessageByRoomID(ctx context.Context, roomID string, limit int) ([]*models.Message, error) {
	query := `SELECT m.id, m.room_id, m.user_id, u.username, m.content, m.message_type, m.reply_to, m.attachments, m.timestamp
			  FROM messages m
			  LEFT JOIN users u ON m.user_id = u.id
			  WHERE m.room_id = $1
			  ORDER BY m.timestamp DESC
			  LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, roomID, limit)
	if err != nil {
		log.Println("err level database FindMessageByRoomID", err)
		return nil, err
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		var msg models.Message
		var username sql.NullString
		if err := rows.Scan(
			&msg.ID,
			&msg.RoomID,
			&msg.SenderID,
			&username,
			&msg.Content,
			&msg.Type,
			&msg.ReplyTo,
			&msg.Attachments,
			&msg.Timestamp,
		); err != nil {
			return nil, err
		}
		if username.Valid {
			msg.SenderName = username.String
		}
		messages = append(messages, &msg)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// UpdateContent updates a message's content. Only the message owner can edit (verified by SQL WHERE).
func (r *RepositoryMessageImpl) UpdateContent(ctx context.Context, messageID string, userID string, newContent string) error {
	query := `UPDATE messages SET content = $1 WHERE id = $2 AND user_id = $3`
	result, err := r.db.ExecContext(ctx, query, newContent, messageID, userID)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

var _ MessageRepository = (*RepositoryMessageImpl)(nil)
