package repeater

import "time"

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

func (r *Repeater) Repeat(strategy Strategy, action Action) (result any, err error) {
	attempts := strategy.Attempts()
	infinity := attempts == 0
	for attempt := uint(1); infinity || attempt <= attempts; attempt++ {
		result, err = action()
		if err != nil {
			for _, onError := range r.onError {
				onError(err)
			}

			if !strategy.Retriable(err) {
				return nil, err
			}

			if strategy.Delay(attempt) > 0 {
				time.Sleep(strategy.Delay(attempt))
			}
		} else {
			return result, nil
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
	return &fixedDelaysStrategy{delays, uint(len(delays)), isRetriable}
}

func (s *fixedDelaysStrategy) Delay(iter uint) time.Duration {
	if len(s.delays) >= int(iter) {
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
