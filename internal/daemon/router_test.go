package daemon

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/daemon/queue"
)

type fakeRouterService struct {
	cancelRunID string
	cancelErr   error
}

func (f *fakeRouterService) Health(context.Context) api.HealthResponse {
	return api.HealthResponse{Status: "ok"}
}

func (f *fakeRouterService) Snapshot(context.Context) (<-chan api.Snapshot, <-chan error) {
	data := make(chan api.Snapshot)
	errs := make(chan error)
	close(data)
	close(errs)
	return data, errs
}

func (f *fakeRouterService) Run(context.Context, string, string) (api.RunResponse, error) {
	return api.RunResponse{}, errors.New("not implemented")
}

func (f *fakeRouterService) Cancel(_ context.Context, runID string) error {
	f.cancelRunID = runID
	return f.cancelErr
}

func TestRouterCancel(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "success", wantStatus: http.StatusNoContent},
		{name: "missing", err: queue.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "finished", err: queue.ErrFinished, wantStatus: http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeRouterService{cancelErr: tt.err}
			req := httptest.NewRequest(http.MethodDelete, "/run/run-123", nil)
			rec := httptest.NewRecorder()

			NewRouter(svc).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if svc.cancelRunID != "run-123" {
				t.Fatalf("runID = %q, want %q", svc.cancelRunID, "run-123")
			}
		})
	}
}

func TestRouterCancelMissingPathValue(t *testing.T) {
	svc := &fakeRouterService{}
	req := httptest.NewRequest(http.MethodDelete, "/run/", nil)
	rec := httptest.NewRecorder()

	NewRouter(svc).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if svc.cancelRunID != "" {
		t.Fatalf("Cancel called with runID %q", svc.cancelRunID)
	}
}
