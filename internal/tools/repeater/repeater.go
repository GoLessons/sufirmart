package repeater

import (
	"context"
	"time"
)

type Repeater struct {
	onError []OnError
}

func NewRepeater(onError ...OnError) *Repeater {
	return &Repeater{onError: onError}
}

type Strategy interface {
	Delay(iter uint) time.Duration
	Attempts() uint
	Retriable(err error) bool
}

type Action func() (any, error)

type OnError func(err error)

func (r *Repeater) Repeat(ctx context.Context, strategy Strategy, action Action) (result any, err error) {
	attempts := strategy.Attempts()
	infinity := attempts == 0

	for attempt := uint(1); infinity || attempt <= attempts; attempt++ {
		result, err = action()
		if err == nil {
			return result, nil
		}

		// вызов обработчиков ошибок
		for _, onError := range r.onError {
			onError(err)
		}

		// если ошибка не ретраибельна — выходим сразу
		if !strategy.Retriable(err) {
			return nil, err
		}

		// задержка перед следующим повтором
		if d := strategy.Delay(attempt); d > 0 {
			timer := time.NewTimer(d)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
				// продолжаем
			}
		}
	}

	return result, err
}

type fixedDelaysStrategy struct {
	delays      []time.Duration
	attempts    uint
	isRetriable func(error) bool
}

func NewFixedDelaysStrategy(isRetriable func(error) bool, delays ...time.Duration) *fixedDelaysStrategy {
	return &fixedDelaysStrategy{
		delays:      delays,
		attempts:    uint(len(delays)),
		isRetriable: isRetriable,
	}
}

func (s *fixedDelaysStrategy) Delay(iter uint) time.Duration {
	if int(iter) <= len(s.delays) {
		return s.delays[iter-1]
	}
	return 0
}

func (s *fixedDelaysStrategy) Retriable(err error) bool {
	return s.isRetriable(err)
}

func (s *fixedDelaysStrategy) Attempts() uint {
	return s.attempts
}
