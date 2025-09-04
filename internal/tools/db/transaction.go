package db

import (
	"context"
	"database/sql"
)

func WrapTransaction[T any](
	ctx context.Context,
	db *sql.DB,
	fn func(ctx context.Context) (T, error),
) (T, error) {
	var zero T

	tx, err := db.BeginTx(ctx, nil)
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
