package accrual

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
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
	endpoint := fmt.Sprintf("%s/api/orders/%s", r.baseURL, url.PathEscape(orderNumber))

	resp, err := r.client.R().
		SetHeader("Accept", "application/json").
		Get(endpoint)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode() {
	case 200:
		var dto struct {
			Order   string   `json:"order"`
			Status  string   `json:"status"`
			Accrual *float64 `json:"accrual,omitempty"`
		}
		if err := json.Unmarshal(resp.Body(), &dto); err != nil {
			return nil, err
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

	case 204:
		return nil, NewErrNotFound(orderNumber)

	case 429:
		retryAfterHeader := resp.Header().Get("Retry-After")
		var dur time.Duration
		if retryAfterHeader != "" {
			if secs, err := strconv.Atoi(retryAfterHeader); err == nil {
				dur = time.Duration(secs) * time.Second
			}
		}
		return nil, NewTooManyRequestsError(dur)

	default:
		if resp.StatusCode() >= 500 {
			return nil, fmt.Errorf("accrual service error: %s", resp.Status())
		}
		return nil, fmt.Errorf("unexpected status code from accrual service: %d", resp.StatusCode())
	}
}
