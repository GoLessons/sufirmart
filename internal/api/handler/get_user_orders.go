package handler

import (
	"encoding/json"
	"net/http"
	"sufirmart/internal/api"
	"sufirmart/internal/auth"
	"sufirmart/internal/domain"
	"sufirmart/internal/repository"
)

func NewGetApiUserOrdersHandler(orders *repository.OrderRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok || userID == "" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		list, err := orders.ListByUser(ctx, userID)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if len(list) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		resp := make([]api.Order, 0, len(list))
		for _, o := range list {
			var st api.OrderStatus
			switch o.Status() {
			case domain.OrderStatusNew:
				st = api.NEW
			case domain.OrderStatusProcessing:
				st = api.PROCESSING
			case domain.OrderStatusInvalid:
				st = api.INVALID
			case domain.OrderStatusProcessed:
				st = api.PROCESSED
			default:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			resp = append(resp, api.Order{
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
}
