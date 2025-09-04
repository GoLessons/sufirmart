package accrual

import (
	"fmt"
	"sufirmart/internal/domain"
	"time"
)

type Reader interface {
	Get(orderNumber string) (*domain.Accrual, error)
}

type ErrNotFound struct {
	orderNum string
}

type OrderInProcessing struct {
	orderNum string
}

func NewErrNotFound(orderNum string) *ErrNotFound {
	return &ErrNotFound{orderNum: orderNum}
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("order %s not found", e.orderNum)
}

type TooManyRequestsError struct {
	RetryAfter time.Duration
}

func NewTooManyRequestsError(RetryAfter time.Duration) *TooManyRequestsError {
	return &TooManyRequestsError{RetryAfter: RetryAfter}
}

func (e *TooManyRequestsError) Error() string {
	return "too many requests"
}

func NewOrderInProcessing(orderNum string) *OrderInProcessing {
	return &OrderInProcessing{orderNum: orderNum}
}

func (e *OrderInProcessing) Error() string {
	return fmt.Sprintf("order %s in processing", e.orderNum)
}
