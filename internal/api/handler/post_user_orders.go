package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sufirmart/internal/auth"
	"sufirmart/internal/domain"
	"sufirmart/internal/order"
	"sufirmart/internal/repository"
	"sufirmart/internal/security"
	"time"
)

func NewPostApiUserOrdersHandler(orders *repository.OrderRepository, processor *order.Processor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "text/plain") {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		rawOrderNum := strings.TrimSpace(string(body))
		if rawOrderNum == "" {
			http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
			return
		}

		if !security.OrderNumValidation(rawOrderNum) {
			http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
			return
		}

		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok || userID == "" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		orderNum, err := domain.NewOrderNumber(rawOrderNum)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
			return
		}

		ctx := r.Context()
		existing, err := orders.GetByNumber(ctx, orderNum, false)
		if err != nil {
			if !errors.Is(err, repository.ErrOrderNotFound) {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		} else if existing != nil {
			if existing.UserID() == userID {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}

		newOrder := domain.NewOrder(userID, orderNum, domain.OrderStatusNew, time.Now(), nil)
		err = orders.Save(ctx, &newOrder)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if processor != nil {
			bgCtx := context.WithoutCancel(r.Context())
			processor.Process(bgCtx, orderNum)
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
