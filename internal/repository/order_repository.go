package repository

import (
	"context"
	"database/sql"
	"log"

	"github.com/AhmadKusumahDEV/go-chat/internal/models"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *models.Order) error
	GetOrderByUserId(ctx context.Context, userID string) ([]*models.Order, error)
	GetActiveOrder(ctx context.Context, userID string) (sql.NullString, error)
	UpdatedOrder(ctx context.Context, order *models.Order) error
}

type OrderRepositoryImpl struct {
	db *sql.DB
}

func (o *OrderRepositoryImpl) UpdatedOrder(ctx context.Context, order *models.Order) error {
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		log.Println(err)
		return errors.New("internal server error")
	}

	var (
		currentStatus string
		userId        uuid.UUID
		plan          string
	)

	err = tx.QueryRowContext(ctx,
		`
	select 
		status,
		user_id,
		plan
	from
		orders
	where 
		order_id = $1
	for update;`, order.OrderID).Scan(&currentStatus, &userId, &plan)
	if err != nil {
		log.Println(err)
		return errors.New("order id not found")
	}

	defer tx.Rollback()

	if order.Status == models.OrderStatus(currentStatus) {
		return models.ErrSameStatus
	}

	//rare case
	if models.MapMidtransStatus(currentStatus, "") == models.OrderStatusSettled {
		return models.ErrStatusAlreadySettled
	}

	query := `
	Updated
		order 
	set 
		status = $1,
		updated_at = now(),
		webhook_payload = $2,
		gateway_tx_id = $3,
		paid_at = now(),
		payment_method = $5
	where 
		order_id = $4
	`

	_, err = tx.ExecContext(ctx, query, order.Status, order.WebHookPayload, order.GatewayTxID.String, order.OrderID, order.PaymentMethod.String)
	if err != nil {
		log.Println(err)
		return errors.New("got error when updated to db")
	}

	if err := tx.Commit(); err != nil {
		return errors.New("error when save data")
	}

	return nil
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

	rows := o.db.QueryRowContext(ctx, query, userID, models.MapMidtransStatus("pending", ""))

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
func (o *OrderRepositoryImpl) GetOrderByUserId(ctx context.Context, userID string) ([]*models.Order, error) {
	query := `
	select
		o.order_id,
		o.user_id,
		o.plan,
		o.gross_amount,
		o.status,
		o.snap_token,
		o.payment_method,
		o.expired_at,
		u.username,
		u.email
	from 
		orders o
	join 
		users u on o.user_id = u.id
	where 
		o.user_id = $1
	order by
		o.created_at desc
	`

	rows, err := o.db.QueryContext(ctx, query, userID)
	if err != nil {
		log.Println(err)
		return nil, errors.New("error when get data")
	}

	order := []*models.Order{}
	for rows.Next() {
		var orderDetail models.Order
		if err := rows.Scan(
			&orderDetail.OrderID,
			&orderDetail.UserID,
			&orderDetail.Plan,
			&orderDetail.Amount,
			&orderDetail.Status,
			&orderDetail.SnapToken,
			&orderDetail.PaymentMethod,
			&orderDetail.ExpiretAt,
			&orderDetail.Username,
			&orderDetail.Email,
		); err != nil {
			log.Println(err)
			return nil, errors.New("error when scan data")
		}
		order = append(order, &orderDetail)
	}

	return order, nil
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
