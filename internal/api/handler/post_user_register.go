package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"sufirmart/internal/api"
	"sufirmart/internal/auth"
	"sufirmart/internal/user"
)

func NewPostApiUserRegisterHandler(userSvc *user.UserService, authSvc auth.Authentication) http.HandlerFunc {
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

		if err := userSvc.RegisterUser(creds.Login, creds.Password); err != nil {
			if errors.Is(err, user.ErrLoginAlreadyExists) {
				http.Error(w, "login already exists", http.StatusConflict)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		token, err := authSvc.Authenticate(creds.Login, creds.Password)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Authorization", "Bearer "+token)
		w.WriteHeader(http.StatusOK)
	}
}
