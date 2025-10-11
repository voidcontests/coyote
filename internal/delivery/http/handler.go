package http

import (
	"net/http"
)

type Handler struct {
	mux *http.ServeMux
}

func NewHandler() *Handler {
	h := &Handler{
		mux: http.NewServeMux(),
	}

	h.initRoutes()
	return h
}

func (h *Handler) initRoutes() {
	h.mux.HandleFunc("/api/healthcheck", h.healthcheck)
}

func (h *Handler) healthcheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}
