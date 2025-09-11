package db

import (
	"context"
	"database/sql"
)

func WrapTransaction[T any](
	ctx context.Context,
	db *sql.DB,
	opts *sql.TxOptions,
	fn func(ctx context.Context) (T, error),
) (T, error) {
	var zero T

	if opts == nil {
		opts = &sql.TxOptions{Isolation: sql.LevelDefault}
	}

	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return zero, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	result, err := fn(ctx)
	if err != nil {
		return zero, err
	}

	err = tx.Commit()
	if err != nil {
		return zero, err
	}
	tx = nil

	return result, nil
}
