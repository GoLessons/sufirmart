package logger

import "go.uber.org/zap"

func NewLogger(config zap.Config) (*zap.Logger, error) {
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}
