package api

const (
	ProtocolHeader = "X-Patchdock-Protocol"
)

type HealthResponse struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
	PID    int    `json:"pid"`
}

type SubmitRequest struct {
	Repo   string `json:"repo"`
	Prompt string `json:"prompt"`
}

type SubmitResponse struct {
	RunID string `json:"run_id"`
}
