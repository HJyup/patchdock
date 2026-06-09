package contracts

// TaskID identifies a pipeline run end-to-end. Threaded through every contract
// so logs, diffs, and retries can be correlated.
type TaskID string

// StageName names a pipeline stage. Used in logs, in the agent-runtime stage
// dispatch flag, and in feedback routing.
type StageName string

const (
	StagePlanner  StageName = "planner"
	StageExecutor StageName = "executor"
	StageReviewer StageName = "reviewer"
)

// TokenUsage records LLM token consumption for one stage invocation.
// Reported back by the agent runtime; the orchestrator sums per-task and
// enforces the per-task cap from config.yml.
type TokenUsage struct {
	Input  int `json:"input" validate:"gte=0"`
	Output int `json:"output" validate:"gte=0"`
}

// Validate reports every broken invariant at once, each error naming the
// offending field.
func (t *TokenUsage) Validate() error { return validateStruct(t, "token_usage") }
