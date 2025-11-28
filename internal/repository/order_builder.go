package repository

import (
	"errors"
	"sufirmart/internal/domain"
	"time"
)

func buildOrder(userID string, orderNumber string, status int16, uploadedAt time.Time, accrual *float64) (*domain.Order, error) {
	uid, err := domain.NewUserID(userID)
	if err != nil {
		return nil, err
	}

	orderStatus := domain.OrderStatus(status)

	if orderStatus == domain.OrderStatusProcessed && accrual == nil {
		return nil, errors.New("accrual must be set for processed orders")
	}

	order := domain.NewOrder(uid, domain.OrderNumber(orderNumber), domain.OrderStatus(status), uploadedAt, accrual)
	return &order, nil
}
