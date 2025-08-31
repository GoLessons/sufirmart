package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sufirmart/internal/auth"
	"sufirmart/internal/domain"
	"sufirmart/internal/repository"
	"sufirmart/internal/security"
	"sufirmart/internal/user"
)

var _ ServerInterface = (*MartApi)(nil)

type MartApi struct {
	authSvc   auth.Authentication
	userSvc   *user.UserService
	ordersRep *repository.Repository
}

func NewApi(authSvc auth.Authentication, userSvc *user.UserService, ordersRep *repository.Repository) MartApi {
	return MartApi{authSvc: authSvc, userSvc: userSvc, ordersRep: ordersRep}
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
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	rawOrderNum := strings.TrimSpace(string(body))
	if rawOrderNum == "" {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
	}

	if !security.OrderNumValidation(rawOrderNum) {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
	}

	// Получаем UserID из контекста (проставляется в auth-мидлвари)
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	orderNum, err := domain.NewOrderNumber(rawOrderNum)
	if err != nil {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
	}

	ctx := r.Context()
	existing, err := s.ordersRep.GetByNumber(ctx, orderNum)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		if existing.UserID == userID {
			// Уже загружен этим пользователем
			w.WriteHeader(http.StatusOK)
			return
		}

		// Уже загружен другим пользователем
		http.Error(w, "order belongs to another user", http.StatusConflict)
		return
	}

	if err := s.ordersRep.InsertNew(ctx, userID, orderNum); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
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
