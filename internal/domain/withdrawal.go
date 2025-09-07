package domain

import "time"

type Withdrawal struct {
	OrderNum    string
	Sum         float64
	ProcessedAt time.Time
}
