package test

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"sufirmart/internal/tools/repeater"
	"testing"
	"time"
)

func TestRepeatSuccessFirstAttempt(t *testing.T) {
	r := repeater.NewRepeater()
	strategy := repeater.NewFixedDelaysStrategy(func(error) bool { return false }, time.Millisecond*10)

	callCount := 0
	action := func() (any, error) {
		callCount++
		return "success", nil
	}

	result, err := r.Repeat(strategy, action)

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 1, callCount, "Action должен быть вызван только один раз")
}

func TestRepeatSuccessAfterRetry(t *testing.T) {
	r := repeater.NewRepeater()
	strategy := repeater.NewFixedDelaysStrategy(func(error) bool { return true }, time.Millisecond*10, time.Millisecond*20)

	callCount := 0
	action := func() (any, error) {
		callCount++
		if callCount < 2 {
			return nil, errors.New("временная ошибка")
		}
		return "success", nil
	}

	result, err := r.Repeat(strategy, action)

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 2, callCount, "Action должен быть вызван дважды")
}

func TestRepeatFailAllAttempts(t *testing.T) {
	r := repeater.NewRepeater()
	strategy := repeater.NewFixedDelaysStrategy(func(error) bool { return true }, time.Millisecond*10, time.Millisecond*20)

	expectedErr := errors.New("постоянная ошибка")
	callCount := 0
	action := func() (any, error) {
		callCount++
		return nil, expectedErr
	}

	result, err := r.Repeat(strategy, action)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, result)
	assert.Equal(t, 2, callCount, "Action должен быть вызван 3 раза (начальная + 2 повтора)")
}

func TestRepeatWithOnErrorCallback(t *testing.T) {
	callbackCalled := 0
	onError := func(err error) {
		callbackCalled++
	}

	r := repeater.NewRepeater(onError)
	strategy := repeater.NewFixedDelaysStrategy(func(error) bool { return true }, time.Millisecond*10, time.Millisecond*10)

	action := func() (any, error) {
		return nil, errors.New("ошибка")
	}

	_, _ = r.Repeat(strategy, action)

	assert.Equal(t, 2, callbackCalled, "OnError колбэк должен быть вызван дважды")
}

func TestRepeatWithInfiniteAttempts(t *testing.T) {
	r := repeater.NewRepeater()
	strategy := &mockInfiniteStrategy{}

	callCount := 0
	action := func() (any, error) {
		callCount++
		if callCount < 10 {
			return nil, errors.New("временная ошибка")
		}
		return "success", nil
	}

	result, err := r.Repeat(strategy, action)

	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 10, callCount, "Action должен быть вызван 10 раз")
}

func TestFixedDelaysStrategy(t *testing.T) {
	delays := []time.Duration{time.Millisecond * 10, time.Millisecond * 20, time.Millisecond * 30}
	strategy := repeater.NewFixedDelaysStrategy(func(error) bool { return true }, delays...)

	assert.Equal(t, uint(3), strategy.Attempts())
	assert.Equal(t, delays[0], strategy.Delay(1))
	assert.Equal(t, delays[1], strategy.Delay(2))
	assert.Equal(t, delays[2], strategy.Delay(3))
	assert.Equal(t, time.Duration(0), strategy.Delay(4), "Для итерации больше длины массива должен возвращаться 0")
}

type mockInfiniteStrategy struct{}

func (s *mockInfiniteStrategy) Delay(iter uint) time.Duration {
	return time.Millisecond
}

func (s *mockInfiniteStrategy) Attempts() uint {
	return 0
}

func (s *mockInfiniteStrategy) Retriable(err error) bool {
	return true
}
