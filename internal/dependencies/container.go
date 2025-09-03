package dependencies

import (
	"database/sql"
	"go.uber.org/zap"
	"sufirmart/internal/accrual"
	"sufirmart/internal/config"
	"sufirmart/internal/tools/workerpool"
)

type Container struct {
	logger        *zap.Logger
	config        *config.AppConfig
	db            *sql.DB
	accrualReader accrual.Reader
	workerPool    *workerpool.WorkerPool
}

func NewContainer(logger *zap.Logger, config *config.AppConfig, db *sql.DB, accrualReader accrual.Reader, workerPool *workerpool.WorkerPool) *Container {
	return &Container{
		logger:        logger,
		config:        config,
		db:            db,
		accrualReader: accrualReader,
		workerPool:    workerPool,
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

func (c *Container) AccrualReader() accrual.Reader {
	return c.accrualReader
}

func (c *Container) WorkerPool() *workerpool.WorkerPool {
	return c.workerPool
}
