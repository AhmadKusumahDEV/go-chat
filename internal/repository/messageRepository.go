package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"log"
	"time"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type MessageRepository interface {
	RepositoryBased[*models.Message]

	FindMessageByRoomID(ctx context.Context, roomID string, limit int, cursor *string) ([]*models.Message, bool, error)
	FindMessageByRoomIDCount(ctx context.Context, roomID string) (int, error)
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

// FindMessageByRoomID returns messages for a room with cursor-based pagination.
// Messages are ordered newest first (DESC).
// Returns messages, hasMore boolean, and error.
func (r *RepositoryMessageImpl) FindMessageByRoomID(ctx context.Context, roomID string, limit int, cursor *string) ([]*models.Message, bool, error) {
	var query string
	var args []any

	if cursor != nil && *cursor != "" {
		timestamp, decodeErr := decodeCursor(*cursor)
		if decodeErr == nil {
			query = `SELECT m.id, m.room_id, m.user_id, u.username, m.content, m.message_type, m.reply_to, m.attachments, m.timestamp
					FROM messages m
					LEFT JOIN users u ON m.user_id = u.id
					WHERE m.room_id = $1 AND m.timestamp < $2
					ORDER BY m.timestamp DESC
					LIMIT $3`
			args = []any{roomID, timestamp, limit + 1}
		} else {
			query = `SELECT m.id, m.room_id, m.user_id, u.username, m.content, m.message_type, m.reply_to, m.attachments, m.timestamp
					FROM messages m
					LEFT JOIN users u ON m.user_id = u.id
					WHERE m.room_id = $1
					ORDER BY m.timestamp DESC
					LIMIT $2`
			args = []any{roomID, limit + 1}
		}
	} else {
		query = `SELECT m.id, m.room_id, m.user_id, u.username, m.content, m.message_type, m.reply_to, m.attachments, m.timestamp
				FROM messages m
				LEFT JOIN users u ON m.user_id = u.id
				WHERE m.room_id = $1
				ORDER BY m.timestamp DESC
				LIMIT $2`
		args = []any{roomID, limit + 1}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Println("err level database FindMessageByRoomID", err)
		return nil, false, err
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
			return nil, false, err
		}
		if username.Valid {
			msg.SenderName = username.String
		}
		messages = append(messages, &msg)
	}

	if err = rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := len(messages) > limit
	if len(messages) > limit {
		messages = messages[:limit]
	}

	return messages, hasMore, nil
}

// FindMessageByRoomIDCount returns total count of messages in a room
func (r *RepositoryMessageImpl) FindMessageByRoomIDCount(ctx context.Context, roomID string) (int, error) {
	query := `SELECT COUNT(*) FROM messages WHERE room_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, roomID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// decodeCursor decodes a base64 cursor string to timestamp
func decodeCursor(cursor string) (time.Time, error) {
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, string(decoded))
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
