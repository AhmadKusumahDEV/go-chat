package repository

import (
	"context"
	"database/sql"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type RepositoryUser interface {
	RepositoryBased[*models.Users]

	FindByEmail(ctx context.Context, email string) (models.Users, error)
	FindByProviderID(ctx context.Context, providerName string, providerID string) (*models.Users, error)
}

type RepositoryUserImpl struct {
	RepositoryBased[*models.Users]
	db *sql.DB
}

// FindByEmail implements RepositoryUser.
func (r *RepositoryUserImpl) FindByEmail(ctx context.Context, email string) (models.Users, error) {
	query := "SELECT id , username, password_hash from users WHERE email = $1"
	var users models.Users

	result := r.db.QueryRowContext(ctx, query, email)

	err := result.Scan(&users.ID, &users.Username, &users.Password)
	if err != nil {
		return models.Users{}, err
	}

	return users, nil
}

// FindByProviderID implements RepositoryUser.
func (r *RepositoryUserImpl) FindByProviderID(ctx context.Context, providerName string, providerID string) (*models.Users, error) {
	query := `SELECT id, username, email, password_hash, created_at, avatar_url, provider_name, provider_id 
			  FROM users WHERE provider_name = $1 AND provider_id = $2`
	var user models.Users

	err := r.db.QueryRowContext(ctx, query, providerName, providerID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.AvatarUrl,
		&user.ProviderName,
		&user.ProviderID,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}



func NewUserRepository(db *sql.DB) RepositoryUser {
	return &RepositoryUserImpl{
		RepositoryBased: NewBaseRepository[*models.Users](db).(*BaseRepository[*models.Users]),
		db:              db,
	}
}

var _ RepositoryUser = (*RepositoryUserImpl)(nil)
