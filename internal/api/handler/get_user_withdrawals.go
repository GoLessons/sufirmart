package handler

import (
	"encoding/json"
	"net/http"
	"sufirmart/internal/api"
	"sufirmart/internal/auth"
	"sufirmart/internal/repository"
)

func NewGetApiUserWithdrawalsHandler(accounts *repository.AccountRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok || userID == "" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		items, err := accounts.ListWithdrawals(ctx, userID)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if len(items) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		resp := make([]api.Withdrawal, 0, len(items))
		for _, it := range items {
			resp = append(resp, api.Withdrawal{
				Order:       it.OrderNum,
				Sum:         float32(it.Sum),
				ProcessedAt: it.ProcessedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
