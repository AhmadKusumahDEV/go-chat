package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type MessageRepository interface {
	RepositoryBased[*models.Message]

	FindMessageByRoomID(ctx context.Context, roomID string, limit int, cursor string) ([]*models.Message, bool, error)
	FindOneMessageByRoomID(ctx context.Context, messageID string) (*models.Message, error)
	FindMessageByRoomIDCount(ctx context.Context, roomID string) (int, error)
	UpdateContent(ctx context.Context, messageID string, userID string, newContent string) error
	CreateSystemMessage(ctx context.Context, roomID string, content string) error
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

func (r *RepositoryMessageImpl) FindOneMessageByRoomID(ctx context.Context, messageID string) (*models.Message, error) {
	query := `	SELECT 
					m.id, 
					m.room_id, 
					m.user_id, 
					u.username, 
					m.content, 
					m.message_type, 
					m.reply_to, 
					m.timestamp,
                    COALESCE(json_agg(ma.url) FILTER (WHERE ma.url IS NOT NULL), '[]'::json) AS attachment_urls
             	FROM 
					messages m
             	LEFT JOIN 
					users u ON m.user_id = u.id
             	LEFT JOIN 
					message_attachments ma ON m.id = ma.message_id
             	WHERE 
					m.id = $1
             	GROUP BY 
					m.id, 
					u.username
`

	var msg models.Message
	var username sql.NullString
	var attachmentBytes []byte
	err := r.db.QueryRowContext(ctx, query, messageID).Scan(
		&msg.ID,
		&msg.RoomID,
		&msg.SenderID,
		&username,
		&msg.Content,
		&msg.Type,
		&msg.ReplyTo,
		&msg.Timestamp,
		&attachmentBytes,
	)
	if err != nil {
		return nil, err
	}
	var urls []string
	if len(attachmentBytes) > 0 {
		_ = json.Unmarshal(attachmentBytes, &urls)
	}

	msg.Attachments = urls

	if username.Valid {
		msg.SenderName = username.String
	}

	return &msg, nil
}

func (r *RepositoryMessageImpl) FindMessageByRoomID(ctx context.Context, roomID string, limit int, cursor string) ([]*models.Message, bool, error) {
	query := `	SELECT 
					m.id, 
					m.room_id, 
					m.user_id, 
					u.username, 
					m.content, 
					m.message_type, 
					m.reply_to, 
					m.timestamp,
                    COALESCE(json_agg(ma.url) FILTER (WHERE ma.url IS NOT NULL), '[]'::json) AS attachment_urls
             	FROM 
					messages m
             	LEFT JOIN 
					users u ON m.user_id = u.id
             	LEFT JOIN 
					message_attachments ma ON m.id = ma.message_id
             	WHERE 
					m.room_id = $1 AND m.timestamp < $2
             	GROUP BY 
					m.id, 
					u.username
             	ORDER BY
					 m.timestamp DESC
             	LIMIT $3`

	queryLimit := limit + 1
	args := []any{roomID, cursor, queryLimit}
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
		var attachmentBytes []byte

		if err := rows.Scan(
			&msg.ID,
			&msg.RoomID,
			&msg.SenderID,
			&username,
			&msg.Content,
			&msg.Type,
			&msg.ReplyTo,
			&msg.Timestamp,
			&attachmentBytes,
		); err != nil {
			return nil, false, err
		}

		var urls []string
		if len(attachmentBytes) > 0 {
			_ = json.Unmarshal(attachmentBytes, &urls)
		}

		msg.Attachments = urls

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

func (r *RepositoryMessageImpl) FindMessageByRoomIDCount(ctx context.Context, roomID string) (int, error) {
	query := `SELECT COUNT(*) FROM messages WHERE room_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, roomID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
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

// CreateSystemMessage creates a system message in a room (no user_id)
func (r *RepositoryMessageImpl) CreateSystemMessage(ctx context.Context, roomID string, content string) error {
	query := `INSERT INTO messages (room_id, content, message_type, timestamp) VALUES ($1, $2, $3, NOW())`
	_, err := r.db.ExecContext(ctx, query, roomID, content, "system")
	return err
}

var _ MessageRepository = (*RepositoryMessageImpl)(nil)
