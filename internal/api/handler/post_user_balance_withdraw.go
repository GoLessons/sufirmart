package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sufirmart/internal/api"
	"sufirmart/internal/auth"
	"sufirmart/internal/repository"
	"sufirmart/internal/security"
	"sufirmart/internal/tools/db"
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

		errInsufficient := errors.New("insufficient funds")
		_, trErr := db.WrapTransaction(ctx, dbConn, func(txCtx context.Context) (any, error) {
			bal, err := accounts.GetBalance(txCtx, userID)
			if err != nil {
				return nil, err
			}

			if bal.Current() >= float64(req.Sum) {
				if err := accounts.ApproveTransaction(txCtx, txID, 0.0); err != nil {
					return nil, err
				}
				return nil, nil
			}

			if err := accounts.CancelTransaction(txCtx, txID, "insufficient funds"); err != nil {
				return nil, err
			}
			return nil, errInsufficient
		})
		if trErr != nil {
			if errors.Is(trErr, errInsufficient) {
				http.Error(w, http.StatusText(http.StatusPaymentRequired), http.StatusPaymentRequired)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
