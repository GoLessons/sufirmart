package domain

type Balance struct {
	userID    UserID
	accrued   float64
	withdrawn float64
}

func NewBalance(userID UserID, accrued, withdrawn float64) Balance {
	return Balance{
		userID:    userID,
		accrued:   accrued,
		withdrawn: withdrawn,
	}
}

func (b Balance) UserID() UserID {
	return b.userID
}

func (b Balance) Accrued() float64 {
	return b.accrued
}

func (b Balance) Withdrawn() float64 {
	return b.withdrawn
}

func (b Balance) Current() float64 {
	return b.accrued - b.withdrawn
}
