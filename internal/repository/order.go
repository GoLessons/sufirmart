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

type Repository struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewRepository(db *sql.DB, logger *zap.Logger) *Repository {
	return &Repository{db: db, logger: logger}
}

func (r *Repository) GetByNumber(ctx context.Context, number domain.OrderNumber) (*domain.Order, error) {
	builder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := builder.
		Select("user_id", "order_num", "status", "uploaded_at").
		From(`"sufirmart"."order"`).
		Where(squirrel.Eq{"order_num": number.String()}).
		ToSql()
	if err != nil {
		return nil, err
	}

	var (
		userID     string
		orderNum   string
		status     int16
		uploadedAt time.Time
	)
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&userID, &orderNum, &status, &uploadedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	order, err := buildOrder(userID, orderNum, status, uploadedAt, nil)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (r *Repository) Save(ctx context.Context, userID domain.UserID, number domain.OrderNumber) error {
	sb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	sqlQ, args, err := sb.
		Insert(`"sufirmart"."order"`).
		Columns("user_id", "order_num", "status").
		Values(userID.String(), number.String(), int16(domain.OrderStatusNew)).
		Suffix(`ON CONFLICT ("order_num") DO NOTHING`).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, sqlQ, args...)
	return err
}

func (r *Repository) ListByUser(ctx context.Context, userID domain.UserID) ([]*domain.Order, error) {
	sb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := sb.
		Select("user_id", "order_num", "status", "uploaded_at").
		From(`"sufirmart"."order"`).
		Where(squirrel.Eq{"user_id": userID.String()}).
		OrderBy(`"uploaded_at" DESC`).
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
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
