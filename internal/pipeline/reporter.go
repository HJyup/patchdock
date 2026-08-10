package pipeline

import "github.com/HJyup/patchdock/internal/types"

type Reporter interface {
	StageChange(stage types.StageName, attempt int)
	StageActivity(activity string)
	StageSummary(summary string)
}
