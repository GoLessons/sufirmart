package db

import (
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"sufirmart/internal/config"
)

func DBFactory(config *config.AppConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", config.DatabaseUri)
	if err != nil {
		return nil, err
	}

	return db, nil
}
