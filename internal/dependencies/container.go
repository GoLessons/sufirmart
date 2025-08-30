package dependencies

import (
	"database/sql"
	"go.uber.org/zap"
	"sufirmart/internal/config"
)

type Container struct {
	logger *zap.Logger
	config *config.AppConfig
	db     *sql.DB
}

func NewContainer(logger *zap.Logger, config *config.AppConfig, db *sql.DB) *Container {
	return &Container{
		logger: logger,
		config: config,
		db:     db,
	}
}

func (c *Container) Config() *config.AppConfig {
	return c.config
}

func (c *Container) Db() *sql.DB {
	return c.db
}

func (c *Container) Logger() *zap.Logger {
	return c.logger
}
