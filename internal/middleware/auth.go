package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sufirmart/internal/auth"
)

func NewAuthMiddleware(authSvc auth.Authentication) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			if !strings.HasPrefix(authorization, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))

			uid, err := authSvc.GetUserID(token)
			if err != nil {
				var ae *auth.AuthError
				if errors.As(err, &ae) {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if uid == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), auth.UserIDContextKey, uid)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
