package auth

import (
	"context"
	"sufirmart/internal/domain"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

func UserIDFromContext(ctx context.Context) (domain.UserID, bool) {
	v := ctx.Value(UserIDContextKey)
	if v == nil {
		return "", false
	}

	id, ok := v.(domain.UserID)
	return id, ok
}
