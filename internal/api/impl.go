package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"sufirmart/internal/auth"
	"sufirmart/internal/user"
)

var _ ServerInterface = (*MartApi)(nil)

type MartApi struct {
	authSvc auth.Authentication
	userSvc *user.UserService
}

func NewApi(authSvc auth.Authentication, userSvc *user.UserService) MartApi {
	return MartApi{authSvc: authSvc, userSvc: userSvc}
}

func (s MartApi) GetApiUserBalance(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s MartApi) PostApiUserBalanceWithdraw(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s MartApi) PostApiUserLogin(w http.ResponseWriter, r *http.Request) {
	var creds UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if creds.Login == "" || creds.Password == "" {
		http.Error(w, "login and password are required", http.StatusBadRequest)
		return
	}

	token, err := s.authSvc.Authenticate(creds.Login, creds.Password)
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

func (s MartApi) GetApiUserOrders(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s MartApi) PostApiUserOrders(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s MartApi) PostApiUserRegister(w http.ResponseWriter, r *http.Request) {
	var creds UserCredentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if creds.Login == "" || creds.Password == "" {
		http.Error(w, "login and password are required", http.StatusBadRequest)
		return
	}

	if err := s.userSvc.RegisterUser(creds.Login, creds.Password); err != nil {
		if errors.Is(err, user.ErrLoginAlreadyExists) {
			http.Error(w, "login already exists", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Автоматически аутентифицируем после успешной регистрации
	token, err := s.authSvc.Authenticate(creds.Login, creds.Password)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

func (s MartApi) GetApiUserWithdrawals(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
