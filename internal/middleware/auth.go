package middleware

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"sufirmart/internal/auth"
)

func NewAuthMiddleware(authSvc auth.Authentication, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			if !strings.HasPrefix(authorization, "Bearer ") {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))

			uid, err := authSvc.GetUserID(token)
			if err != nil {
				var ae *auth.AuthError
				if errors.As(err, &ae) {
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				logger.Error("auth middleware internal error", zap.Error(err), zap.String("method", r.Method), zap.String("path", r.URL.Path))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			if uid == "" {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), auth.UserIDContextKey, uid)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
