package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Masterminds/squirrel"
	"go.uber.org/zap"
	"sufirmart/internal/domain"
	"time"
)

var ErrOrderNotFound = errors.New("order not found")

type OrderRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewOrderRepository(db *sql.DB, logger *zap.Logger) *OrderRepository {
	return &OrderRepository{db: db, logger: logger}
}

func (r *OrderRepository) GetByNumber(ctx context.Context, number domain.OrderNumber, lock bool) (*domain.Order, error) {
	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Select("user_id", "order_num", "status", "uploaded_at").
		From(`"sufirmart"."order"`).
		Where(squirrel.Eq{"order_num": number.String()})
	if lock {
		builder = builder.Suffix("FOR UPDATE")
	}

	var (
		userID     string
		orderNum   string
		status     int16
		uploadedAt time.Time
	)
	err := builder.RunWith(r.db).QueryRowContext(ctx).Scan(&userID, &orderNum, &status, &uploadedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}

		return nil, err
	}

	order, err := buildOrder(userID, orderNum, status, uploadedAt, nil)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (r *OrderRepository) Save(ctx context.Context, order *domain.Order) error {
	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Insert(`"sufirmart"."order"`).
		Columns("user_id", "order_num", "status").
		Values(order.UserID().String(), order.Number().String(), int16(order.Status())).
		Suffix(`ON CONFLICT ("user_id", "order_num") DO UPDATE SET status = EXCLUDED.status`)

	_, err := builder.RunWith(r.db).ExecContext(ctx)
	return err
}

func (r *OrderRepository) ListByUser(ctx context.Context, userID domain.UserID) ([]*domain.Order, error) {
	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Select(
			"user_id",
			"order_num",
			"status",
			"uploaded_at",
			`(SELECT t.accrual FROM "sufirmart"."transaction" t
			  WHERE t.user_id = "sufirmart"."order".user_id
			    AND t.order_num = "sufirmart"."order".order_num
			    AND t.status = 1          -- domain.TransactionStatusProcessed
			    AND t.accrual > 0
			  ORDER BY t.processed_at DESC
			  LIMIT 1) AS accrual`,
		).
		From(`"sufirmart"."order"`).
		Where(squirrel.Eq{"user_id": userID.String()}).
		OrderBy(`"uploaded_at" DESC`)

	rows, err := builder.RunWith(r.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	var orders []*domain.Order
	for rows.Next() {
		var (
			userIDStr string
			orderNum  string
			status    int16
			uploaded  time.Time
			accrual   sql.NullFloat64
		)
		if err := rows.Scan(&userIDStr, &orderNum, &status, &uploaded, &accrual); err != nil {
			return nil, err
		}

		var accrPtr *float64
		if accrual.Valid {
			accrPtr = &accrual.Float64
		}

		o, err := buildOrder(userIDStr, orderNum, status, uploaded, accrPtr)
		if err != nil {
			return nil, err
		}

		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *OrderRepository) ListByStatuses(ctx context.Context, statuses []domain.OrderStatus) ([]*domain.Order, error) {
	sb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	intStatuses := make([]int16, 0, len(statuses))
	for _, s := range statuses {
		intStatuses = append(intStatuses, int16(s))
	}

	builder := sb.
		Select("user_id", "order_num", "status", "uploaded_at").
		From(`"sufirmart"."order"`).
		Where(squirrel.Eq{"status": intStatuses}).
		OrderBy(`"uploaded_at" ASC`)

	rows, err := builder.RunWith(r.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) { _ = rows.Close() }(rows)

	var orders []*domain.Order
	for rows.Next() {
		var (
			userIDStr string
			orderNum  string
			status    int16
			uploaded  time.Time
		)
		if err := rows.Scan(&userIDStr, &orderNum, &status, &uploaded); err != nil {
			return nil, err
		}

		o, err := buildOrder(userIDStr, orderNum, status, uploaded, nil)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}
