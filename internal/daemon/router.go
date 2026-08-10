package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/HJyup/patchdock/internal/daemon/api"
)

type service interface {
	Health(ctx context.Context) api.HealthResponse
	Snapshot(ctx context.Context) (<-chan api.Snapshot, <-chan error)
}

type Router struct {
	service service
	mux     *http.ServeMux
}

func NewRouter(service service) http.Handler {
	rt := &Router{service: service, mux: http.NewServeMux()}

	rt.mux.HandleFunc("GET /health", rt.health)
	rt.mux.HandleFunc("GET /run", rt.stream)

	return rt
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rt.mux.ServeHTTP(w, r)
}

func (rt *Router) health(w http.ResponseWriter, r *http.Request) {
	rt.writeJSON(w, http.StatusOK, rt.service.Health(r.Context()))
}

func (rt *Router) stream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	data, errs := rt.service.Snapshot(ctx)
	for {
		select {
		case <-ctx.Done():
			return

		case err, ok := <-errs:
			if !ok {
				return
			}
			if err := writeEvent(w, flusher, api.EventError, api.ErrorEvent{Message: err.Error()}); err != nil {
				log.Printf("stream: %v", err)
			}
			return

		case snap, ok := <-data:
			if !ok {
				return
			}
			if err := writeEvent(w, flusher, api.EventSnapshot, snap); err != nil {
				log.Printf("stream: %v", err)
				return
			}
		}
	}
}

func writeEvent(w http.ResponseWriter, flusher http.Flusher, event string, payload any) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode %s event: %w", event, err)
	}

	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, buf); err != nil {
		return fmt.Errorf("write %s event: %w", event, err)
	}

	flusher.Flush()
	return nil
}

func (rt *Router) writeJSON(w http.ResponseWriter, status int, payload any) {
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
