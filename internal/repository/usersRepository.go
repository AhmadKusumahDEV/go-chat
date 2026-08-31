package repository

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type RepositoryUser interface {
	RepositoryBased[*models.Users]

	FindByEmail(ctx context.Context, email string) (models.Users, error)
	FindByProviderID(ctx context.Context, providerName string, providerID string) (*models.Users, error)
	FindByIDs(ctx context.Context, userIDs []string) ([]models.Users, error)
	ChangeAvatar(ctx context.Context, userID, avatarURL string) error
	UpdateVerifyUser(ctx context.Context, userID string) error
	UpdateTierUser(ctx context.Context, userID string, tier string) error
	ProfileUser(ctx context.Context, userID string) (models.Users, error)
}

type RepositoryUserImpl struct {
	RepositoryBased[*models.Users]
	db *sql.DB
}

func (r *RepositoryUserImpl) ProfileUser(ctx context.Context, userID string) (models.Users, error) {
	var user models.Users
	query := `
	select 
		id,
		username,
		email,
		avatar_url,
		about,
		tier,
		verify
	from 
		users
	where 
		id = $1;
	`

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.AvatarUrl,
		&user.About,
		&user.Tier,
		&user.Verify,
	)

	if err != nil {
		return models.Users{}, err
	}

	return user, nil
}

func (r *RepositoryUserImpl) UpdateVerifyUser(ctx context.Context, userID string) error {
	query := `
	update
		users
	set
		verify = true
	where
		id = $1
	`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		log.Println(err)
		return errors.New("failed update user")
	}
	return nil
}

// UpdateTierUser updates the user's subscription tier (e.g., "free" -> "premium")
func (r *RepositoryUserImpl) UpdateTierUser(ctx context.Context, userID string, tier string) error {
	query := `
	update
		users
	set
		tier = $1,
		updated_at = now()
	where
		id = $2
	`
	result, err := r.db.ExecContext(ctx, query, tier, userID)
	if err != nil {
		log.Println(err)
		return errors.New("failed update user tier")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	log.Printf("✅ User %s tier updated to %s", userID, tier)
	return nil
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

func (r *RepositoryUserImpl) ChangeAvatar(ctx context.Context, userID, avatarURL string) error {
	query := `UPDATE users SET avatar_url = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, avatarURL, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("user not found")
	}
	return nil
}

var _ RepositoryUser = (*RepositoryUserImpl)(nil)
