package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"sufirmart/internal/api"
	"sufirmart/internal/auth"
)

func NewPostApiUserLoginHandler(authSvc auth.Authentication) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds api.UserCredentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if creds.Login == "" || creds.Password == "" {
			http.Error(w, "login and password are required", http.StatusBadRequest)
			return
		}

		token, err := authSvc.Authenticate(creds.Login, creds.Password)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				http.Error(w, "invalid credentials", http.StatusUnauthorized)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Authorization", "Bearer "+token)
		w.WriteHeader(http.StatusOK)
	}
}
