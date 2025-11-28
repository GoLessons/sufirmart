package accrual

import (
	"sufirmart/internal/domain"
	"sync"
)

type StubReader struct {
	mu       sync.RWMutex
	data     map[string]*domain.Accrual
	forceErr map[string]error
}

func NewStubReader(initial ...map[string]*domain.Accrual) *StubReader {
	r := &StubReader{
		data: make(map[string]*domain.Accrual),
	}

	if len(initial) > 0 && initial[0] != nil {
		for k, v := range initial[0] {
			r.data[k] = v
		}
	}

	return r
}

func (r *StubReader) Get(orderNumber string) (*domain.Accrual, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err, ok := r.forceErr[orderNumber]; ok {
		return nil, err
	}

	if acc, ok := r.data[orderNumber]; ok {
		return acc, nil
	}

	return nil, NewErrNotFound(orderNumber)
}

func (r *StubReader) Set(acc *domain.Accrual) {
	if acc == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[acc.OrderNumber().String()] = acc
}

func (r *StubReader) Remove(orderNumber string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, orderNumber)
}

func (r *StubReader) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data = make(map[string]*domain.Accrual)
}

func (r *StubReader) FailWith(orderNumber string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.forceErr[orderNumber] = err
}

func (r *StubReader) RemoveError(orderNumber string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.forceErr, orderNumber)
}
