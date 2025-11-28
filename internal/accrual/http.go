package accrual

import (
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"net/http"
	"net/url"
	"strconv"
	"sufirmart/internal/domain"
	"time"
)

type HttpReader struct {
	client  *resty.Client
	baseURL string
}

func NewHttpReader(baseURL string) *HttpReader {
	c := resty.New()
	return &HttpReader{
		client:  c,
		baseURL: baseURL,
	}
}

func (r *HttpReader) Get(orderNumber string) (*domain.Accrual, error) {
	endpoint, err := url.JoinPath(r.baseURL, "api", "orders", orderNumber)
	if err != nil {
		return nil, NewBuildURLError(err)
	}

	resp, err := r.client.R().
		SetHeader("Accept", "application/json").
		Get(endpoint)
	if err != nil {
		return nil, NewRequestError(err)
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		var dto struct {
			Order   string   `json:"order"`
			Status  string   `json:"status"`
			Accrual *float64 `json:"accrual,omitempty"`
		}
		if err := json.Unmarshal(resp.Body(), &dto); err != nil {
			return nil, NewDecodeError(err)
		}

		orderNum, err := domain.NewOrderNumber(dto.Order)
		if err != nil {
			return nil, err
		}

		status, err := AccrualStatusFromString(dto.Status)
		if err != nil {
			return nil, err
		}

		value := 0.0
		if dto.Accrual != nil {
			value = *dto.Accrual
		}

		return domain.NewAccural(orderNum, status, value), nil

	case http.StatusNoContent:
		return nil, NewErrNotFound(orderNumber)

	case http.StatusTooManyRequests:
		retryAfterHeader := resp.Header().Get("Retry-After")
		var dur time.Duration
		if retryAfterHeader != "" {
			if secs, err := strconv.Atoi(retryAfterHeader); err == nil {
				dur = time.Duration(secs) * time.Second
			}
		}
		return nil, NewTooManyRequestsError(dur)

	default:
		if resp.StatusCode() >= http.StatusInternalServerError {
			return nil, NewAccrualServiceError(resp.StatusCode(), resp.Status())
		}
		return nil, NewUnexpectedStatusError(resp.StatusCode())
	}
}
