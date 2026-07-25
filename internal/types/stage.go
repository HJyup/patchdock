package types

// StageName is the shared vocabulary between Go and the SDK: it keys the stage
// config, tags log events, and reaches the container as PATCHDOCK_STAGE
type StageName string

const (
	StagePlanner  StageName = "planner"
	StageExecutor StageName = "executor"
	StageReviewer StageName = "reviewer"
)
