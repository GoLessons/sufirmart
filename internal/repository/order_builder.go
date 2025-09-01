package repository

import (
	"sufirmart/internal/domain"
	"time"
)

func buildOrder(userID string, orderNumber string, status int16, uploadedAt time.Time, accrual *float32) (*domain.Order, error) {
	uid, err := domain.NewUserID(userID)
	if err != nil {
		return nil, err
	}

	num := domain.OrderNumber(orderNumber)
	if err != nil {
		return nil, err
	}

	order := domain.NewOrder(uid, num, domain.OrderStatus(status), uploadedAt, accrual)
	return &order, nil
}
