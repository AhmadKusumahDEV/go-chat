package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type RepositoryFirebase interface {
	RepositoryBased[*models.Firebase]

	CreateFcmToken(ctx context.Context, firebase *models.Firebase) error
	GetTokensByUserIDs(ctx context.Context, userIDs []string) ([]string, error)
	DeactivateToken(ctx context.Context, fcmToken string) error
	DeactivateTokensByUserIDs(ctx context.Context, userIDs []string, validTokens []string) (deactivatedCount int, err error)
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

func (r *RepositoryFirebaseImpl) DeactivateTokensByUserIDs(ctx context.Context, userIDs []string, validTokens []string) (int, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}

	// If no valid tokens, deactivate ALL tokens for these users
	if len(validTokens) == 0 {
		query := `UPDATE user_devices SET is_active = FALSE, updated_at = NOW() WHERE user_id IN (`
		args := make([]interface{}, len(userIDs))
		for i, id := range userIDs {
			if i > 0 {
				query += ", "
			}
			query += "$" + strconv.Itoa(i+1)
			args[i] = id
		}
		query += ")"

		result, err := r.db.ExecContext(ctx, query, args...)
		if err != nil {
			return 0, err
		}

		rowsAffected, _ := result.RowsAffected()
		return int(rowsAffected), nil
	}

	// Build query to deactivate tokens NOT in validTokens list
	query := `UPDATE user_devices SET is_active = FALSE, updated_at = NOW() WHERE user_id IN (`
	args := make([]interface{}, 0, len(userIDs)+len(validTokens))

	for i, id := range userIDs {
		if i > 0 {
			query += ", "
		}
		query += "$" + strconv.Itoa(i+1)
		args = append(args, id)
	}
	query += ") AND fcm_token NOT IN ("
	for i, token := range validTokens {
		if i > 0 {
			query += ", "
		}
		query += "$" + strconv.Itoa(len(userIDs)+i+1)
		args = append(args, token)
	}
	query += ")"

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
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
