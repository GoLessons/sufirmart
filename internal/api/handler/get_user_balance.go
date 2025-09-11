package handler

import (
	"encoding/json"
	"net/http"
	"sufirmart/internal/api"
	"sufirmart/internal/auth"
	"sufirmart/internal/repository"
)

func NewGetApiUserBalanceHandler(accounts *repository.AccountRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok || userID == "" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		balance, err := accounts.GetBalance(ctx, userID)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		resp := api.Balance{
			Current:   balance.Current(),
			Withdrawn: balance.Withdrawn(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
