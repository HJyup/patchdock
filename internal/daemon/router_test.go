package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/api"
)

type stubService struct {
	health    api.HealthResponse
	runResp   api.RunResponse
	runErr    error
	cancelErr error

	snapshots chan api.Snapshot
	snapErrs  chan error

	gotRepo   string
	gotPrompt string
	gotRunID  string
}

func newFakeService() *stubService {
	return &stubService{
		health:    api.HealthResponse{Status: "ok", Uptime: "1m0s", PID: 4242},
		runResp:   api.RunResponse{RunID: "run-abc123"},
		snapshots: make(chan api.Snapshot),
		snapErrs:  make(chan error, 1),
	}
}

func (f *stubService) Health(context.Context) api.HealthResponse { return f.health }

func (f *stubService) Run(_ context.Context, repo, prompt string) (api.RunResponse, error) {
	f.gotRepo, f.gotPrompt = repo, prompt
	return f.runResp, f.runErr
}

func (f *stubService) Cancel(_ context.Context, runID string) error {
	f.gotRunID = runID
	return f.cancelErr
}

func (f *stubService) Snapshot(context.Context) (<-chan api.Snapshot, <-chan error) {
	return f.snapshots, f.snapErrs
}

func do(t *testing.T, h http.Handler, method, target string, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, reader))
	return rec
}

func TestRunAcceptsAPrompt(t *testing.T) {
	svc := newFakeService()
	rec := do(t, NewRouter(svc), http.MethodPost, "/run", `{"repo":"/abs/repo","prompt":"fix the bug"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body)
	}

	var got api.RunResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if got.RunID != "run-abc123" {
		t.Errorf("run_id = %q, want run-abc123", got.RunID)
	}

	if svc.gotRepo != "/abs/repo" || svc.gotPrompt != "fix the bug" {
		t.Errorf("service received repo=%q prompt=%q, want the decoded payload", svc.gotRepo, svc.gotPrompt)
	}
}

func TestRunStatusCodes(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		runErr   error
		wantCode int
	}{
		{
			name:     "malformed body",
			body:     `{"repo":`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "not json at all",
			body:     `hello`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "service rejects the payload",
			body:     `{"repo":"relative/path","prompt":"x"}`,
			runErr:   ErrInvalidUserPayload,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "service fails internally",
			body:     `{"repo":"/abs/repo","prompt":"x"}`,
			runErr:   errors.New("queue is gone"),
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newFakeService()
			svc.runErr = tt.runErr

			rec := do(t, NewRouter(svc), http.MethodPost, "/run", tt.body)
			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d: %s", rec.Code, tt.wantCode, rec.Body)
			}
		})
	}
}

func TestCancelWithoutARunIDIsNotRouted(t *testing.T) {
	svc := newFakeService()

	rec := do(t, NewRouter(svc), http.MethodDelete, "/run/", "")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// Snapshot stream

func TestStreamEmitsSnapshotEvents(t *testing.T) {
	svc := newFakeService()
	rt := NewRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/run", nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		rt.ServeHTTP(rec, req)
	}()

	svc.snapshots <- api.Snapshot{
		At:   time.Now(),
		Runs: []api.Run{{ID: "run-1", Status: api.StatusCoding}},
	}
	close(svc.snapshots)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("stream handler never returned after its source closed")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("content-type = %q, want text/event-stream", got)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: "+api.EventSnapshot) {
		t.Errorf("body has no snapshot event:\n%s", body)
	}

	if !strings.Contains(body, `"id":"run-1"`) {
		t.Errorf("body does not carry the run payload:\n%s", body)
	}
}

func TestStreamEmitsErrorEvents(t *testing.T) {
	svc := newFakeService()
	rt := NewRouter(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/run", nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		rt.ServeHTTP(rec, req)
	}()

	svc.snapErrs <- errors.New("follow broker: the broker has been closed")

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("stream handler did not return after reporting an error")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "event: "+api.EventError) {
		t.Errorf("body has no error event:\n%s", body)
	}

	if !strings.Contains(body, "the broker has been closed") {
		t.Errorf("error event does not carry the message:\n%s", body)
	}
}

func TestStreamStopsWhenTheClientDisconnects(t *testing.T) {
	svc := newFakeService()
	rt := NewRouter(svc)

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/run", nil).WithContext(ctx)

	done := make(chan struct{})
	go func() {
		defer close(done)
		rt.ServeHTTP(httptest.NewRecorder(), req)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("stream handler outlived its request context")
	}
}
