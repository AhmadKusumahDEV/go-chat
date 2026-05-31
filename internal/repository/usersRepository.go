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
	FindByIDs(ctx context.Context, userIDs []string) ([]models.Users, error)
}

type RepositoryUserImpl struct {
	RepositoryBased[*models.Users]
	db *sql.DB
}

// FindByEmail implements RepositoryUser.
func (r *RepositoryUserImpl) FindByEmail(ctx context.Context, email string) (models.Users, error) {
	query := `SELECT id, username, email, password_hash, avatar_url, about, created_at, provider_name, provider_id
			  FROM users WHERE email = $1`
	var users models.Users

	result := r.db.QueryRowContext(ctx, query, email)

	err := result.Scan(
		&users.ID,
		&users.Username,
		&users.Email,
		&users.Password,
		&users.AvatarUrl,
		&users.About,
		&users.CreatedAt,
		&users.ProviderName,
		&users.ProviderID,
	)
	if err != nil {
		return models.Users{}, err
	}

	return users, nil
}

// FindByProviderID implements RepositoryUser.
func (r *RepositoryUserImpl) FindByProviderID(ctx context.Context, providerName string, providerID string) (*models.Users, error) {
	query := `SELECT id, username, email, password_hash, avatar_url, about, created_at, provider_name, provider_id
			  FROM users WHERE provider_name = $1 AND provider_id = $2`
	var user models.Users

	err := r.db.QueryRowContext(ctx, query, providerName, providerID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.AvatarUrl,
		&user.About,
		&user.CreatedAt,
		&user.ProviderName,
		&user.ProviderID,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// FindByIDs implements RepositoryUser - finds multiple users by their IDs
func (r *RepositoryUserImpl) FindByIDs(ctx context.Context, userIDs []string) ([]models.Users, error) {
	if len(userIDs) == 0 {
		return []models.Users{}, nil
	}

	// Build query with placeholder
	query := `SELECT id, username, email, password_hash, avatar_url, about, created_at, provider_name, provider_id FROM users WHERE id = ANY($1)`
	rows, err := r.db.QueryContext(ctx, query, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.Users
	for rows.Next() {
		var user models.Users
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Password,
			&user.AvatarUrl,
			&user.About,
			&user.CreatedAt,
			&user.ProviderName,
			&user.ProviderID,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *RepositoryUserImpl) Logout(ctx context.Context, userID string) error { return nil }

func NewUserRepository(db *sql.DB) RepositoryUser {
	return &RepositoryUserImpl{
		RepositoryBased: NewBaseRepository[*models.Users](db).(*BaseRepository[*models.Users]),
		db:              db,
	}
}

var _ RepositoryUser = (*RepositoryUserImpl)(nil)
