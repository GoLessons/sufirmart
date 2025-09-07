package repository

import (
	"context"
	"database/sql"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"math"
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

	builder := sb.
		Select(
			`COALESCE(SUM("accrual"), 0)`,
			`COALESCE(SUM("withdraw"), 0)`,
		).
		From(`"sufirmart"."transaction"`).
		Where(squirrel.And{
			squirrel.Eq{"user_id": userID.String()},
			squirrel.Eq{"status": int16(domain.TransactionStatusProcessed)},
		})
	var accrued, withdrawn sql.NullFloat64
	if err := builder.RunWith(r.db).QueryRowContext(ctx).Scan(&accrued, &withdrawn); err != nil {
		return nil, err
	}

	balance := domain.NewBalance(userID, accrued.Float64, withdrawn.Float64)

	return &balance, nil
}

func (r *AccountRepository) RegisterTransaction(ctx context.Context, userID domain.UserID, orderNum string, sum float64, comment string) (string, error) {
	transactionID := uuid.NewString()

	withdraw, accrual := 0.0, 0.0
	if sum < 0 {
		withdraw = math.Abs(sum)
	} else {
		accrual = math.Abs(sum)
	}

	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Insert(`"sufirmart"."transaction"`).
		Columns("id", "user_id", "order_num", "accrual", "withdraw", "status", "comment").
		Values(transactionID, userID.String(), orderNum, accrual, withdraw, int16(domain.TransactionStatusPlanned), comment)

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

func (r *AccountRepository) ListWithdrawals(ctx context.Context, userID domain.UserID) ([]domain.Withdrawal, error) {
	builder := squirrel.StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Select("order_num", "withdraw", "processed_at").
		From(`"sufirmart"."transaction"`).
		Where(squirrel.And{
			squirrel.Eq{"user_id": userID.String()},
			squirrel.Eq{"status": int16(domain.TransactionStatusProcessed)},
			squirrel.Gt{"withdraw": 0},
		}).
		OrderBy(`"processed_at" DESC`)

	rows, err := builder.RunWith(r.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []domain.Withdrawal
	for rows.Next() {
		var w domain.Withdrawal
		if err := rows.Scan(&w.OrderNum, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
