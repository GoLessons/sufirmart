package dependencies

import "go.uber.org/zap"

type Container struct {
	logger *zap.Logger
}

func New(logger *zap.Logger) *Container {
	return &Container{logger: logger}
}

func (c *Container) Logger() *zap.Logger {
	return c.logger
}
