package accrual

import (
	"fmt"
	"sufirmart/internal/domain"
)

func AccrualStatusFromString(status string) (domain.AccrualStatus, error) {
	switch status {
	case "REGISTERED":
		return domain.AccrualStatusRegistered, nil
	case "PROCESSING":
		return domain.AccrualStatusProcessing, nil
	case "INVALID":
		return domain.AccrualStatusInvalid, nil
	case "PROCESSED":
		return domain.AccrualStatusProcessed, nil
	default:
		return domain.AccrualStatusUnspecified, fmt.Errorf("unknown accrual status: %s", status)
	}
}
