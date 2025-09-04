package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sufirmart/internal/auth"
	"sufirmart/internal/domain"
	"sufirmart/internal/order"
	"sufirmart/internal/repository"
	"sufirmart/internal/security"
	"sufirmart/internal/user"
	"time"
)

var _ ServerInterface = (*MartApi)(nil)

type MartApi struct {
	authSvc     auth.Authentication
	userSvc     *user.UserService
	ordersRep   *repository.OrderRepository
	accountsRep *repository.AccountRepository
	processor   *order.Processor
}

func NewApi(authSvc auth.Authentication, userSvc *user.UserService, ordersRep *repository.OrderRepository, accountsRep *repository.AccountRepository, processor *order.Processor) MartApi {
	return MartApi{authSvc: authSvc, userSvc: userSvc, ordersRep: ordersRep, accountsRep: accountsRep, processor: processor}
}

func (s MartApi) GetApiUserBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	balance, err := s.accountsRep.GetBalance(ctx, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := Balance{
		Current:   balance.Current(),
		Withdrawn: balance.Withdrawn(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
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
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	list, err := s.ordersRep.ListByUser(ctx, userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if len(list) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	resp := make([]Order, 0, len(list))
	for _, o := range list {
		var st OrderStatus
		switch o.Status() {
		case domain.OrderStatusNew:
			st = NEW
		case domain.OrderStatusProcessing:
			st = PROCESSING
		case domain.OrderStatusInvalid:
			st = INVALID
		case domain.OrderStatusProcessed:
			st = PROCESSED
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp = append(resp, Order{
			Number:     o.Number().String(),
			Status:     st,
			UploadedAt: o.UploadedAt(),
			Accrual:    o.Accrual(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
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
	existing, err := s.ordersRep.GetByNumber(ctx, orderNum, false)
	if err != nil {
		if !errors.Is(err, repository.ErrOrderNotFound) {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	} else if existing != nil {
		if existing.UserID() == userID {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "order belongs to another user", http.StatusConflict)
		return
	}

	newOrder := domain.NewOrder(userID, orderNum, domain.OrderStatusNew, time.Now(), nil)
	err = s.ordersRep.Save(ctx, &newOrder)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if s.processor != nil {
		s.processor.Process(context.TODO(), orderNum)
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
