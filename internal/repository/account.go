package repository

import (
	"context"
	"database/sql"
	"github.com/Masterminds/squirrel"
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
