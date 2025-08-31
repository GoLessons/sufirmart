package auth

import (
	"context"
	"sufirmart/internal/domain"
)

const UserIDContextKey string = "userID"

func UserIDFromContext(ctx context.Context) (domain.UserID, bool) {
	v := ctx.Value(UserIDContextKey)
	if v == nil {
		return "", false
	}

	id, ok := v.(domain.UserID)
	return id, ok
}
