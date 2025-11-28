package domain

type TransactionStatus int8

const (
	TransactionStatusPlanned   TransactionStatus = 0
	TransactionStatusProcessed TransactionStatus = 1
	TransactionStatusCanceled  TransactionStatus = -1
)
