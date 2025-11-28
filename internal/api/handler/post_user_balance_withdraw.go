package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sufirmart/internal/api"
	"sufirmart/internal/auth"
	"sufirmart/internal/repository"
	"sufirmart/internal/security"
)

func NewPostApiUserBalanceWithdrawHandler(accounts *repository.AccountRepository, dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok || userID == "" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		var req api.WithdrawRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if !security.OrderNumValidation(req.Order) {
			http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
			return
		}
		if req.Sum <= 0 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		txID, err := accounts.RegisterTransaction(ctx, userID, req.Order, -float64(req.Sum), "User withdraw")
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = accounts.ApproveTransaction(ctx, txID, 0.0)
		if err != nil {
			if errors.Is(err, repository.ErrInsufficientFunds) {
				http.Error(w, http.StatusText(http.StatusPaymentRequired), http.StatusPaymentRequired)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
