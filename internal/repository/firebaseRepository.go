package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/lib/pq"
)

type RepositoryFirebase interface {
	RepositoryBased[*models.Firebase]

	CreateFcmToken(ctx context.Context, firebase *models.Firebase) error
	GetTokensByUserIDs(ctx context.Context, userIDs []string) ([]string, error)
	DeactivateToken(ctx context.Context, fcmToken string) error
	DeactivateTokenByUserID(ctx context.Context, userID string, fcmToken string) (int, error)
	DeactivateTokens(ctx context.Context, tokens []string) (int, error)
	DeactivateAllTokensByUserID(ctx context.Context, userID string) (int, error)
}

type RepositoryFirebaseImpl struct {
	RepositoryBased[*models.Firebase]
	db *sql.DB
}

// CreateFcmToken implements [RepositoryFirebase].
func (r *RepositoryFirebaseImpl) CreateFcmToken(ctx context.Context, firebase *models.Firebase) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("terjadi kegagalan pada database")
	}
	defer tx.Rollback()

	qDeactivateOldToken := `
        UPDATE user_devices
        SET
            is_active = FALSE,
            updated_at = NOW()
        WHERE installation_id = $1
        AND fcm_token <> $2
        AND is_active = TRUE;
    `

	_, err = tx.ExecContext(
		ctx,
		qDeactivateOldToken,
		firebase.InstallationID,
		firebase.FcmToken,
	)
	if err != nil {
		return errors.New("failed when update deactivate token")
	}

	qUpsertFcmToken := `
        INSERT INTO user_devices (
			user_id,
            installation_id,
            fcm_token,
            platform,
            is_active,
            logged_out_at,
            created_at,
            updated_at
        )
        VALUES (
            $1, $2, $3, $4,
            TRUE,
            NULL,
            NOW(),
            NOW()
        )
        ON CONFLICT (fcm_token)
        DO UPDATE SET
            user_id = EXCLUDED.user_id,
            installation_id = EXCLUDED.installation_id,
            platform = EXCLUDED.platform,
            is_active = TRUE,
            logged_out_at = NULL,
            updated_at = NOW();
    `

	_, err = tx.ExecContext(
		ctx,
		qUpsertFcmToken,
		firebase.UserID,
		firebase.InstallationID,
		firebase.FcmToken,
		firebase.Platform,
	)
	if err != nil {
		return errors.New("failed when upsert fcm token")
	}

	if err := tx.Commit(); err != nil {
		return errors.New("failed when commit transaction")
	}

	return nil
}

// GetTokensByUserIDs implements RepositoryFirebase.
func (r *RepositoryFirebaseImpl) GetTokensByUserIDs(ctx context.Context, userIDs []string) ([]string, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	query := "SELECT fcm_token FROM user_devices WHERE is_active = TRUE AND user_id IN ("
	args := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		if i > 0 {
			query += ", "
		}
		query += "$" + strconv.Itoa(i+1)
		args[i] = id
	}
	query += ")"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

// DeactivateToken sets is_active = FALSE for a specific FCM token
func (r *RepositoryFirebaseImpl) DeactivateToken(ctx context.Context, fcmToken string) error {
	query := `
		UPDATE user_devices
		SET
			is_active = FALSE,
			updated_at = NOW()
		WHERE fcm_token = $1
	`
	result, err := r.db.ExecContext(ctx, query, fcmToken)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *RepositoryFirebaseImpl) DeactivateTokenByUserID(ctx context.Context, userID string, fcmToken string) (int, error) {
	query := `
        UPDATE user_devices
        SET
            is_active = FALSE,
            logged_out_at = NOW(),
            updated_at = NOW()
        WHERE fcm_token = $1
        AND user_id = $2
		AND is_active = TRUE
	`
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, errors.New("deactivate token got error: " + err.Error())
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

func (r *RepositoryFirebaseImpl) DeactivateTokens(ctx context.Context, tokens []string) (int, error) {
	if len(tokens) == 0 {
		return 0, nil
	}

	query := `
        UPDATE user_devices
        SET
            is_active = FALSE,
            updated_at = NOW()
        WHERE fcm_token = ANY($1)
        AND is_active = TRUE
    `

	result, err := r.db.ExecContext(ctx, query, pq.Array(tokens))
	if err != nil {
		return 0, fmt.Errorf("deactivate tokens: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// DeactivateAllTokensByUserID deactivates ALL FCM tokens for a specific user (used on logout)
func (r *RepositoryFirebaseImpl) DeactivateAllTokensByUserID(ctx context.Context, userID string) (int, error) {
	query := `
        UPDATE user_devices
        SET
            is_active = FALSE,
            logged_out_at = NOW(),
            updated_at = NOW()
        WHERE user_id = $1
        AND is_active = TRUE
    `

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("deactivate all tokens by user ID: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

func NewFirebaseRepository(db *sql.DB) RepositoryFirebase {
	return &RepositoryFirebaseImpl{
		RepositoryBased: NewBaseRepository[*models.Firebase](db).(*BaseRepository[*models.Firebase]),
		db:              db,
	}
}

var _ RepositoryFirebase = (*RepositoryFirebaseImpl)(nil)
