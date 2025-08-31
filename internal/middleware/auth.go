package middleware

import (
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
			ok, err := authSvc.IsAuthenticated(token)
			if err != nil || !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
