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
}

type Router struct {
	service service
	mux     *http.ServeMux
}

func NewRouter(service service) http.Handler {
	rt := &Router{service: service, mux: http.NewServeMux()}

	rt.mux.HandleFunc("GET /health", rt.health)

	return rt
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt.mux.ServeHTTP(w, r)
}

func (rt *Router) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, rt.service.Health(r.Context()))
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
