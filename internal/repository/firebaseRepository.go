package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type RepositoryFirebase interface {
	RepositoryBased[*models.Firebase]

	CreateFcmToken(ctx context.Context, firebase *models.Firebase) error
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
	_, err = tx.ExecContext(ctx, qDeactivateOldToken, firebase.UserID, firebase.InstallationID, firebase.FcmToken)
	if err != nil && err != sql.ErrNoRows {
		return errors.New("failed when updated deactive token")
	}

	qUpsertFcmToken := `
	INSERT INTO user_devices (
    user_id,
    installation_id,
    fcm_token,
    platform,
    is_active,
    logged_out_at,
    last_seen_at,
    created_at,
    updated_at
	)
	VALUES (
		$1, $2, $3, $4,
		TRUE,
		NULL,
		NOW(),
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
    last_seen_at = NOW(),
    updated_at = NOW();
	`
	_, err = tx.ExecContext(ctx, qUpsertFcmToken, firebase.UserID, firebase.InstallationID, firebase.FcmToken, firebase.Platform)
	if err != nil && err != sql.ErrNoRows {
		return errors.New("failed when upsert fcm token")
	}

	return tx.Commit()
}

func NewFirebaseRepository(db *sql.DB) RepositoryFirebase {
	return &RepositoryFirebaseImpl{
		RepositoryBased: NewBaseRepository[*models.Firebase](db).(*BaseRepository[*models.Firebase]),
		db:              db,
	}
}

var _ RepositoryFirebase = (*RepositoryFirebaseImpl)(nil)
