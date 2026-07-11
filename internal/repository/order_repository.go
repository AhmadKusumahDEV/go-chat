package repository

import (
	"context"
	"database/sql"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/pkg/errors"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *models.Order) error
	GetOrderByUserId(ctx context.Context, userID string) (*models.Order, error)
	GetActiveOrder(ctx context.Context, userID string) (sql.NullString, error)
}

type OrderRepositoryImpl struct {
	db *sql.DB
}

// GetActiveOrder implements [OrderRepository].
func (o *OrderRepositoryImpl) GetActiveOrder(ctx context.Context, userID string) (sql.NullString, error) {
	query := `
	select 
		snap_token
	from 
		orders
	where 
		user_id = $1
		and status = $2
		and plan = 'premium'
		and expired_at > now()
 	order by 
 		created_at desc
 	limit 1;
	`

	rows := o.db.QueryRowContext(ctx, query, userID, models.MapMidtransStatus("pending"))

	var snaptoken sql.NullString

	err := rows.Scan(
		&snaptoken,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return snaptoken, nil
		}
		return snaptoken, err
	}

	return snaptoken, nil
}

// GetOrderByUserId implements [OrderRepository].
func (o *OrderRepositoryImpl) GetOrderByUserId(ctx context.Context, userID string) (*models.Order, error) {
	panic("unimplemented")
}

func (o *OrderRepositoryImpl) CreateOrder(ctx context.Context, order *models.Order) error {
	query := `
	        INSERT INTO orders (
            order_id, user_id, plan, gross_amount,
            gateway, status, snap_token,
            expired_at
        ) VALUES (
            $1, $2, $3, $4, $5,
            $6, $7, $8
        )
	`
	_, err := o.db.ExecContext(ctx, query,
		order.OrderID, order.UserID, order.Plan, order.Amount, order.Gateway, order.Status, order.SnapToken, order.ExpiretAt)
	if err != nil {
		return errors.New("failed create order")
	}

	return nil
}

func NewOrderRepository(db *sql.DB) OrderRepository {
	return &OrderRepositoryImpl{
		db: db,
	}
}
