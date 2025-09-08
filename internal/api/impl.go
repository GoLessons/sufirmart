package api

import (
	"net/http"
)

var _ ServerInterface = (*MartApi)(nil)

type MartApi struct {
	apiHandlers map[string]http.HandlerFunc
}

func NewApi(handlers map[string]http.HandlerFunc) MartApi {
	return MartApi{apiHandlers: handlers}
}

func (s MartApi) GetApiUserBalance(w http.ResponseWriter, r *http.Request) {
	h, ok := s.apiHandlers["GetApiUserBalance"]
	if !ok || h == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h(w, r)
}

func (s MartApi) PostApiUserBalanceWithdraw(w http.ResponseWriter, r *http.Request) {
	h, ok := s.apiHandlers["PostApiUserBalanceWithdraw"]
	if !ok || h == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h(w, r)
}

func (s MartApi) PostApiUserLogin(w http.ResponseWriter, r *http.Request) {
	h, ok := s.apiHandlers["PostApiUserLogin"]
	if !ok || h == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h(w, r)
}

func (s MartApi) GetApiUserOrders(w http.ResponseWriter, r *http.Request) {
	h, ok := s.apiHandlers["GetApiUserOrders"]
	if !ok || h == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h(w, r)
}

func (s MartApi) PostApiUserOrders(w http.ResponseWriter, r *http.Request) {
	h, ok := s.apiHandlers["PostApiUserOrders"]
	if !ok || h == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h(w, r)
}

func (s MartApi) PostApiUserRegister(w http.ResponseWriter, r *http.Request) {
	h, ok := s.apiHandlers["PostApiUserRegister"]
	if !ok || h == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h(w, r)
}

func (s MartApi) GetApiUserWithdrawals(w http.ResponseWriter, r *http.Request) {
	h, ok := s.apiHandlers["GetApiUserWithdrawals"]
	if !ok || h == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h(w, r)
}
