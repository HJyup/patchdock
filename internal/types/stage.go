package types

// StageName names a pipeline stage. Used in logs, in the agent-runtime stage
// dispatch flag, and in feedback routing.
type StageName string

const (
	StagePlanner  StageName = "planner"
	StageExecutor StageName = "executor"
	StageReviewer StageName = "reviewer"
)
