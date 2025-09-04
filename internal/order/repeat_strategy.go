package order

import (
	"errors"
	"sufirmart/internal/accrual"
	"time"
)

type AdaptiveStrategy struct {
	baseDelay   time.Duration
	maxDelay    time.Duration
	attempts    uint
	isRetriable func(error) bool
	lastDelay   time.Duration
}

func NewAdaptiveStrategy(baseDelay time.Duration, attempts uint) *AdaptiveStrategy {
	return &AdaptiveStrategy{
		baseDelay:   baseDelay,
		maxDelay:    30 * time.Second,
		attempts:    attempts,
		isRetriable: isRetriableError,
	}
}

func (s *AdaptiveStrategy) Attempts() uint {
	return s.attempts
}

func (s *AdaptiveStrategy) Retriable(err error) bool {
	if !s.isRetriable(err) {
		return false
	}

	var tooMany *accrual.TooManyRequestsError
	if errors.As(err, &tooMany) {
		delay := tooMany.RetryAfter
		if delay <= 0 {
			delay = s.baseDelay
		}

		if delay > s.maxDelay {
			delay = s.maxDelay
		}

		s.lastDelay = delay

		return true
	}

	// сбрасываем lastDelay
	s.lastDelay = 0
	return true
}

func (s *AdaptiveStrategy) Delay(iter uint) time.Duration {
	if s.lastDelay > 0 {
		return s.lastDelay
	}

	return s.baseDelay
}

var ErrStillProcessing = errors.New("accrual still processing")

func isRetriableError(err error) bool {
	var notFound *accrual.ErrNotFound
	var processing *accrual.OrderInProcessing
	var tooMany *accrual.TooManyRequestsError

	return errors.As(err, &notFound) ||
		errors.As(err, &processing) ||
		errors.As(err, &tooMany) ||
		errors.Is(err, ErrStillProcessing)
}
