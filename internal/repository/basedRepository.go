package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"

	"github.com/AhmadKusumahDEV/go-chat/internal/helpers"
	"github.com/AhmadKusumahDEV/go-chat/internal/models"
)

type RepositoryBased[T models.Entity] interface {
	Create(ctx context.Context, entity T) error
	FindByID(ctx context.Context, id any) (T, error)
	FindAll(ctx context.Context) ([]T, error)
	Update(ctx context.Context, entity T) error
	Delete(ctx context.Context, id any) error
}

type BaseRepository[T models.Entity] struct {
	db *sql.DB
}

func NewBaseRepository[T models.Entity](db *sql.DB) RepositoryBased[T] {
	return &BaseRepository[T]{
		db: db,
	}
}

// Create implements Repository.
func (r *BaseRepository[T]) Create(ctx context.Context, entity T) error {
	// generate uuid
	v6, err := uuid.NewV6()

	if err != nil {
		return fmt.Errorf("gagal membuat UUIDv6: %w", err)
	}

	// 1. Validate
	if err := entity.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// 2. Extract fields using reflection
	fields, err := helpers.ExtractFields(entity)
	if err != nil {
		return err
	}

	// 3. Build INSERT query dynamically
	var columns []string
	var placeholders []string
	var values []any
	placeholderIndex := 1

	for _, field := range fields {

		if field.IsPK {
			columns = append(columns, field.DBColumn)
			placeholders = append(placeholders, fmt.Sprintf("$%d", placeholderIndex))
			values = append(values, v6)
			placeholderIndex++
			continue
		}
		// Skip auto-generated fields (like ID, created_at if auto)
		if field.IsAuto {
			continue
		}

		columns = append(columns, field.DBColumn)
		placeholders = append(placeholders, fmt.Sprintf("$%d", placeholderIndex))
		values = append(values, field.Value)
		placeholderIndex++
	}

	// Build query: INSERT INTO users (email, name, password) VALUES ($1, $2, $3) RETURNING id
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		entity.TableName(),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	fmt.Printf("Generated Query: %s\n", query)
	fmt.Printf("Values: %v\n", values)

	// 4. Execute query
	var id any
	err = r.db.QueryRowContext(ctx, query, values...).Scan(&id)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}

	return nil
}

// FindByID - Automatically generates SELECT query
func (r *BaseRepository[T]) FindByID(ctx context.Context, id any) (T, error) {
	var zero T

	// 1. Get fields using reflection
	fields, err := helpers.ExtractFields(zero)
	if err != nil {
		return zero, err
	}

	// 2. Build SELECT query
	var columns []string
	for _, field := range fields {
		columns = append(columns, field.DBColumn)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE id = $1",
		strings.Join(columns, ", "),
		zero.TableName(),
	)

	// 3. Execute query
	row := r.db.QueryRowContext(ctx, query, id)

	// 4. Scan into entity using reflection
	entity, err := helpers.ScanRow[T](row, fields)
	if err != nil {
		if err == sql.ErrNoRows {
			return zero, errors.New("entity not found")
		}
		return zero, err
	}

	return entity, nil
}

// FindAll - Automatically generates SELECT query
func (r *BaseRepository[T]) FindAll(ctx context.Context) ([]T, error) {
	var zero T

	fields, err := helpers.ExtractFields(zero)
	if err != nil {
		return nil, err
	}

	var columns []string
	for _, field := range fields {
		columns = append(columns, field.DBColumn)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s ORDER BY id",
		strings.Join(columns, ", "),
		zero.TableName(),
	)

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		entity, err := helpers.ScanRow[T](rows, fields)
		if err != nil {
			return nil, err
		}
		results = append(results, entity)
	}

	return results, nil
}

// Update - Automatically generates UPDATE query
func (r *BaseRepository[T]) Update(ctx context.Context, entity T) error {
	var zero T
	if err := entity.Validate(); err != nil {
		return err
	}

	fields, err := helpers.ExtractFields(zero)
	if err != nil {
		return err
	}

	var setClauses []string
	var values []any
	var pkValue any
	placeholderIndex := 1

	for _, field := range fields {
		if field.IsPK {
			pkValue = field.Value
			continue
		}

		// Skip auto fields on update
		if field.IsAuto && field.DBColumn != "updated_at" {
			continue
		}

		// Set updated_at to now if it's auto
		if field.DBColumn == "updated_at" && field.IsAuto {
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field.DBColumn, placeholderIndex))
			values = append(values, time.Now())
		} else {
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", field.DBColumn, placeholderIndex))
			values = append(values, field.Value)
		}
		placeholderIndex++
	}

	// Add PK value at the end
	values = append(values, pkValue)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d",
		entity.TableName(),
		strings.Join(setClauses, ", "),
		placeholderIndex,
	)

	fmt.Printf("Generated Update Query: %s\n", query)

	_, err = r.db.ExecContext(ctx, query, values...)
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	return nil
}

// Delete - Simple DELETE query
func (r *BaseRepository[T]) Delete(ctx context.Context, id any) error {
	var zero T
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", zero.TableName())

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("entity not found")
	}

	return nil
}
