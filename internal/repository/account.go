package repository

import (
	"context"
	"database/sql"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"sufirmart/internal/domain"
)

type AccountRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewAccountRepository(db *sql.DB, logger *zap.Logger) *AccountRepository {
	return &AccountRepository{db: db, logger: logger}
}

func (r *AccountRepository) GetBalance(ctx context.Context, userID domain.UserID) (*domain.Balance, error) {
	sb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := sb.
		Select(
			`COALESCE(SUM("accrual"), 0)`,
			`COALESCE(SUM("withdraw"), 0)`,
		).
		From(`"sufirmart"."transaction"`).
		Where(squirrel.And{
			squirrel.Eq{"user_id": userID.String()},
			squirrel.Eq{"status": int16(domain.TransactionStatusProcessed)},
		}).
		ToSql()
	if err != nil {
		return nil, err
	}

	var accrued, withdrawn sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&accrued, &withdrawn); err != nil {
		return nil, err
	}

	balance := domain.NewBalance(userID, accrued.Float64, withdrawn.Float64)

	return &balance, nil
}

func (r *AccountRepository) RegisterTransaction(ctx context.Context, userID domain.UserID, orderNum string, accrual float64, comment string) (string, error) {
	transactionID := uuid.NewString()
	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Insert(`"sufirmart"."transaction"`).
		Columns("id", "user_id", "order_num", "accrual", "withdraw", "status", "comment").
		Values(transactionID, userID.String(), orderNum, accrual, 0.0, int16(domain.TransactionStatusPlanned), comment)

	_, err := builder.RunWith(r.db).ExecContext(ctx)
	if err != nil {
		return "", err
	}

	return transactionID, nil
}

func (r *AccountRepository) CancelTransaction(ctx context.Context, transactionID string, reason string) error {
	sb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	update := sb.Update(`"sufirmart"."transaction"`).
		Set("status", int16(domain.TransactionStatusCanceled)).
		Set("comment", reason).
		Set("processed_at", squirrel.Expr("NOW()"))

	builder := update.Where(squirrel.Eq{"id": transactionID})

	_, err := builder.RunWith(r.db).ExecContext(ctx)
	return err
}

func (r *AccountRepository) ApproveTransaction(ctx context.Context, transactionID string, accrual float64) error {
	sb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	update := sb.Update(`"sufirmart"."transaction"`).
		Set("status", int16(domain.TransactionStatusProcessed)).
		Set("accrual", accrual).
		Set("processed_at", squirrel.Expr("NOW()"))

	builder := update.Where(squirrel.Eq{"id": transactionID})

	_, err := builder.RunWith(r.db).ExecContext(ctx)
	return err
}
