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
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if creds.Login == "" || creds.Password == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if err := userSvc.RegisterUser(creds.Login, creds.Password); err != nil {
			if errors.Is(err, user.ErrLoginAlreadyExists) {
				http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		token, err := authSvc.Authenticate(creds.Login, creds.Password)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Authorization", "Bearer "+token)
		w.WriteHeader(http.StatusOK)
	}
}
