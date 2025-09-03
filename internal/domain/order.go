package domain

import (
	"errors"
	"time"
)

type OrderStatus int16

const (
	OrderStatusNew OrderStatus = iota
	OrderStatusProcessing
	OrderStatusInvalid
	OrderStatusProcessed
)

var ErrStatusCantBeChanged = errors.New("order status can't be changed")

type Order struct {
	userID     UserID
	number     OrderNumber
	status     OrderStatus
	uploadedAt time.Time
	accrual    *float64
}

func NewOrder(
	userID UserID,
	number OrderNumber,
	status OrderStatus,
	uploadedAt time.Time,
	accrual *float64,
) Order {
	return Order{
		userID:     userID,
		number:     number,
		status:     status,
		uploadedAt: uploadedAt,
		accrual:    accrual,
	}
}

func (o *Order) UserID() UserID {
	return o.userID
}

func (o *Order) Number() OrderNumber {
	return o.number
}

func (o *Order) Status() OrderStatus {
	return o.status
}

func (o *Order) UploadedAt() time.Time {
	return o.uploadedAt
}

func (o *Order) Accrual() *float64 {
	return o.accrual
}

func (o *Order) ChangeStatus(newStatus OrderStatus) error {
	// Проверяем, что текущий статус не является терминальным
	if o.status == OrderStatusInvalid || o.status == OrderStatusProcessed {
		return ErrStatusCantBeChanged
	}

	o.status = newStatus
	return nil
}
