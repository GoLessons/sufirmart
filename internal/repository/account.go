package repository

import (
	"context"
	"database/sql"
	"errors"
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

var ErrInsufficientFunds = errors.New("insufficient funds")

func NewAccountRepository(db *sql.DB, logger *zap.Logger) *AccountRepository {
	return &AccountRepository{db: db, logger: logger}
}

func (r *AccountRepository) GetBalance(ctx context.Context, userID domain.UserID) (*domain.Balance, error) {
	sb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	var current sql.NullFloat64
	err := sb.
		Select(`"current_balance"`).
		From(`"sufirmart"."account"`).
		Where(squirrel.Eq{"user_id": userID.String()}).
		RunWith(r.db).
		QueryRowContext(ctx).
		Scan(&current)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if errors.Is(err, sql.ErrNoRows) || (!current.Valid || current.Float64 == 0) {
		var currentFromHistory sql.NullFloat64
		if err := sb.
			Select(`COALESCE(SUM("accrual" - "withdraw"), 0)`).
			From(`"sufirmart"."transaction"`).
			Where(squirrel.And{
				squirrel.Eq{"user_id": userID.String()},
				squirrel.Eq{"status": int16(domain.TransactionStatusProcessed)},
			}).
			RunWith(r.db).
			QueryRowContext(ctx).
			Scan(&currentFromHistory); err != nil {
			return nil, err
		}
		var withdrawn sql.NullFloat64
		if err := sb.
			Select(`COALESCE(SUM("withdraw"), 0)`).
			From(`"sufirmart"."transaction"`).
			Where(squirrel.And{
				squirrel.Eq{"user_id": userID.String()},
				squirrel.Eq{"status": int16(domain.TransactionStatusProcessed)},
				squirrel.Gt{"withdraw": 0},
			}).
			RunWith(r.db).
			QueryRowContext(ctx).
			Scan(&withdrawn); err != nil {
			return nil, err
		}
		accrued := currentFromHistory.Float64 + withdrawn.Float64
		balance := domain.NewBalance(userID, accrued, withdrawn.Float64)
		return &balance, nil
	}

	var withdrawn sql.NullFloat64
	if err := sb.
		Select(`COALESCE(SUM("withdraw"), 0)`).
		From(`"sufirmart"."transaction"`).
		Where(squirrel.And{
			squirrel.Eq{"user_id": userID.String()},
			squirrel.Eq{"status": int16(domain.TransactionStatusProcessed)},
			squirrel.Gt{"withdraw": 0},
		}).
		RunWith(r.db).
		QueryRowContext(ctx).
		Scan(&withdrawn); err != nil {
		return nil, err
	}

	accrued := current.Float64 + withdrawn.Float64
	balance := domain.NewBalance(userID, accrued, withdrawn.Float64)

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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var userID string
	var plannedWithdraw float64
	if err := tx.QueryRowContext(
		ctx,
		`SELECT "user_id", "withdraw" FROM "sufirmart"."transaction" WHERE "id" = $1 FOR UPDATE`,
		transactionID,
	).Scan(&userID, &plannedWithdraw); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO "sufirmart"."account" ("user_id") VALUES ($1) ON CONFLICT ("user_id") DO NOTHING`,
		userID,
	); err != nil {
		return err
	}

	if plannedWithdraw > 0 {
		var currentBalance sql.NullFloat64
		err := tx.QueryRowContext(
			ctx,
			`SELECT "current_balance" FROM "sufirmart"."account" WHERE "user_id" = $1 FOR UPDATE`,
			userID,
		).Scan(&currentBalance)

		if err != nil {
			return err
		}

		res, err := tx.ExecContext(
			ctx,
			`UPDATE "sufirmart"."account"
             SET "current_balance" = "current_balance" - $2
             WHERE "user_id" = $1 AND "current_balance" >= $2`,
			userID, plannedWithdraw,
		)
		if err != nil {
			return err
		}

		affected, _ := res.RowsAffected()
		if affected == 0 {
			if _, err := tx.ExecContext(ctx,
				`UPDATE "sufirmart"."transaction"
                 SET "status" = $2, "comment" = $3, "processed_at" = NOW()
                 WHERE "id" = $1`,
				transactionID, int16(domain.TransactionStatusCanceled), "insufficient funds",
			); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return err
			}
			return ErrInsufficientFunds
		}

		if _, err := tx.ExecContext(ctx,
			`UPDATE "sufirmart"."transaction"
             SET "status" = $2, "processed_at" = NOW()
             WHERE "id" = $1`,
			transactionID, int16(domain.TransactionStatusProcessed),
		); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	}

	if accrual > 0 {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO "sufirmart"."account" ("user_id", "current_balance")
             VALUES ($1, $2)
             ON CONFLICT ("user_id")
             DO UPDATE SET "current_balance" = "sufirmart"."account"."current_balance" + EXCLUDED."current_balance"`,
			userID, accrual,
		); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE "sufirmart"."transaction"
             SET "status" = $2, "accrual" = $3, "processed_at" = NOW()
             WHERE "id" = $1`,
			transactionID, int16(domain.TransactionStatusProcessed), accrual,
		); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return nil
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE "sufirmart"."transaction"
         SET "status" = $2, "processed_at" = NOW()
         WHERE "id" = $1`,
		transactionID, int16(domain.TransactionStatusProcessed),
	); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
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

func (r *AccountRepository) CancelPlannedTransactionsByOrder(ctx context.Context, userID domain.UserID, orderNum string, reason string) error {
	sb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	update := sb.Update(`"sufirmart"."transaction"`).
		Set("status", int16(domain.TransactionStatusCanceled)).
		Set("comment", reason).
		Set("processed_at", squirrel.Expr("NOW()")).
		Where(squirrel.And{
			squirrel.Eq{"user_id": userID.String()},
			squirrel.Eq{"order_num": orderNum},
			squirrel.Eq{"status": int16(domain.TransactionStatusPlanned)},
		})

	_, err := update.RunWith(r.db).ExecContext(ctx)
	return err
}
