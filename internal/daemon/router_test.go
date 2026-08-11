package daemon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HJyup/patchdock/internal/daemon/api"
)

type stubService struct{}

func (stubService) Health(context.Context) api.HealthResponse {
	return api.HealthResponse{}
}

func (stubService) Snapshot(context.Context) (<-chan api.Snapshot, <-chan error) {
	return nil, nil
}

func (stubService) Run(context.Context, string, string) (api.RunResponse, error) {
	return api.RunResponse{}, fmt.Errorf("%w: repo path is not absolute", ErrInvalidUserPayload)
}

func (stubService) Cancel(context.Context, string) error {
	return nil
}

func TestRunInvalidUserPayloadReturnsBadRequestOnly(t *testing.T) {
	router := NewRouter(stubService{})
	req := httptest.NewRequest(http.MethodPost, "/run", strings.NewReader(`{"repo":"relative","prompt":"test"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "invalid user payload: repo path is not absolute") {
		t.Fatalf("body = %q, want invalid user payload message", body)
	}
	if strings.Contains(body, "failed to submit work") {
		t.Fatalf("body = %q, want no generic failure message", body)
	}
}
