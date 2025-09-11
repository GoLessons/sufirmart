package domain

type AccrualStatus int16

const (
	AccrualStatusUnspecified AccrualStatus = iota
	AccrualStatusRegistered
	AccrualStatusProcessing
	AccrualStatusInvalid
	AccrualStatusProcessed
)

type Accrual struct {
	orderNum OrderNumber
	status   AccrualStatus
	value    float64
}

func NewAccural(orderNum OrderNumber, status AccrualStatus, value float64) *Accrual {
	return &Accrual{
		orderNum: orderNum,
		status:   status,
		value:    value,
	}
}

func (a *Accrual) OrderNumber() OrderNumber {
	return a.orderNum
}

func (a *Accrual) Value() float64 {
	return a.value
}

func (a *Accrual) Status() AccrualStatus {
	return a.status
}
