package domain

import "time"

type OrderStatus int16

const (
	OrderStatusNew OrderStatus = iota
	OrderStatusProcessing
	OrderStatusInvalid
	OrderStatusProcessed
)

type Order struct {
	UserID     UserID
	Number     OrderNumber
	Status     OrderStatus
	UploadedAt time.Time
	Accrual    *float32
}

func NewOrder(
	userID UserID,
	number OrderNumber,
	status OrderStatus,
	uploadedAt time.Time,
	accrual *float32,
) Order {
	return Order{
		UserID:     userID,
		Number:     number,
		Status:     status,
		UploadedAt: uploadedAt,
		Accrual:    accrual,
	}
}
