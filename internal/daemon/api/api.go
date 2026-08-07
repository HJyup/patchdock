package api

const (
	ProtocolHeader = "X-Patchdock-Protocol"
)

type HealthResponse struct {
	Status string `json:"status"`
	Uptime string `json:"uptime"`
	PID    int    `json:"pid"`
}
