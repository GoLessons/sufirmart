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

type BuildURLError struct {
	err error
}

func NewBuildURLError(err error) *BuildURLError {
	return &BuildURLError{err: err}
}

func (e *BuildURLError) Error() string {
	return fmt.Sprintf("build endpoint: %v", e.err)
}

func (e *BuildURLError) Unwrap() error {
	return e.err
}

type RequestError struct {
	err error
}

func NewRequestError(err error) *RequestError {
	return &RequestError{err: err}
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("request accrual: %v", e.err)
}

func (e *RequestError) Unwrap() error {
	return e.err
}

type DecodeError struct {
	err error
}

func NewDecodeError(err error) *DecodeError {
	return &DecodeError{err: err}
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("decode accrual response: %v", e.err)
}

func (e *DecodeError) Unwrap() error {
	return e.err
}

type AccrualServiceError struct {
	StatusCode int
	Status     string
}

func NewAccrualServiceError(code int, status string) *AccrualServiceError {
	return &AccrualServiceError{StatusCode: code, Status: status}
}

func (e *AccrualServiceError) Error() string {
	return fmt.Sprintf("accrual service error: %s", e.Status)
}

type UnexpectedStatusError struct {
	StatusCode int
}

func NewUnexpectedStatusError(code int) *UnexpectedStatusError {
	return &UnexpectedStatusError{StatusCode: code}
}

func (e *UnexpectedStatusError) Error() string {
	return fmt.Sprintf("unexpected status code from accrual service: %d", e.StatusCode)
}
