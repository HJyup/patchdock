package daemon

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/HJyup/patchdock/internal/daemon/api"
)

type service interface {
	Health(ctx context.Context) api.HealthResponse
	Queue(ctx context.Context, action any)
}

type Router struct {
	service service
	mux     *http.ServeMux
}

func NewRouter(service service) http.Handler {
	rt := &Router{service: service, mux: http.NewServeMux()}

	rt.mux.HandleFunc("GET /health", rt.health)
	rt.mux.HandleFunc("POST /queue", rt.queue)

	return rt
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt.mux.ServeHTTP(w, r)
}

func (rt *Router) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, rt.service.Health(r.Context()))
}

func (rt *Router) queue(w http.ResponseWriter, r *http.Request) {
	var req api.QueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, nil)
		return
	}

	rt.service.Queue(r.Context(), req)
	writeJSON(w, http.StatusOK, nil)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	buf, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(buf); err != nil {
		log.Printf("write response: %v", err)
	}
}
