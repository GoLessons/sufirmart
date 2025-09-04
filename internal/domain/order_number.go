package domain

import (
	"fmt"
	"sufirmart/internal/security"
)

type OrderNumber string

func NewOrderNumber(s string) (OrderNumber, error) {
	if !security.OrderNumValidation(s) {
		return "", fmt.Errorf("invalid order number: %s", s)
	}
	return OrderNumber(s), nil
}

func (n OrderNumber) String() string {
	return string(n)
}
