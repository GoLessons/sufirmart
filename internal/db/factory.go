package db

import (
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"sufirmart/internal/config"
)

func DBFactory(config *config.AppConfig, logger *zap.Logger) (*sql.DB, error) {
	db, err := sql.Open("pgx", config.DatabaseUri)
	if err != nil {
		return nil, err
	}

	if config.AutoMigrate && config.MigrationDir != "" {
		migrator := NewMigrator(db, logger, config.MigrationDir)
		if err := migrator.Up(); err != nil {
			_ = db.Close()
			return nil, err
		}
	}

	return db, nil
}
