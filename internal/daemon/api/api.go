package api

const (
	ProtocolHeader = "X-Patchdock-Protocol"
)

// GET /health
type HealthResponse struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
	PID    int    `json:"pid"`
}

// POST /run
type RunRequest struct {
	Repo   string `json:"repo"`
	Prompt string `json:"prompt"`
}

type RunResponse struct {
	RunID string `json:"run_id"`
}

// Returned from SSE events
type SnapshotReponse struct {
	Event string   `json:"event"`
	Data  Snapshot `json:"data"`
}
