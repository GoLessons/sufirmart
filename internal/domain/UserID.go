package domain

import (
	"fmt"
	"regexp"
)

type UserID string

func NewUserID(s string) (UserID, error) {
	uuidV7Regexp := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidV7Regexp.MatchString(s) {
		return "", fmt.Errorf("invalid UUIDv7 format: %s", s)
	}

	return UserID(s), nil
}

func (id UserID) String() string {
	return string(id)
}
