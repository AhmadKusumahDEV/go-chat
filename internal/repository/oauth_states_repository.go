package repository

import (
	"context"
	"database/sql"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type RepositoryOauthStates interface {
	RepositoryBased[*models.OauthStates]

	FindByState(ctx context.Context, state string) (models.OauthStates, error)
	DeleteByState(ctx context.Context, state string) error
}

type RepositoryOauthStatesImpl struct {
	RepositoryBased[*models.OauthStates]
	db *sql.DB
}

// FindByState implements RepositoryOauthStates.
func (r *RepositoryOauthStatesImpl) FindByState(ctx context.Context, state string) (models.OauthStates, error) {
	query := "SELECT id, state, provider, expires_at, created_at FROM oauth_states WHERE state = $1"
	var oauthState models.OauthStates

	result := r.db.QueryRowContext(ctx, query, state)

	err := result.Scan(
		&oauthState.ID,
		&oauthState.State,
		&oauthState.Provider,
		&oauthState.ExpiresAt,
		&oauthState.CreatedAt,
	)
	if err != nil {
		return models.OauthStates{}, err
	}

	return oauthState, nil
}

func (r *RepositoryOauthStatesImpl) DeleteByState(ctx context.Context, state string) error {
	query := "DELETE FROM oauth_states WHERE state = $1"
	_, err := r.db.ExecContext(ctx, query, state)
	return err
}

func NewOauthStatesRepository(db *sql.DB) RepositoryOauthStates {
	return &RepositoryOauthStatesImpl{
		RepositoryBased: NewBaseRepository[*models.OauthStates](db).(*BaseRepository[*models.OauthStates]),
		db:              db,
	}
}

var _ RepositoryOauthStates = (*RepositoryOauthStatesImpl)(nil)
