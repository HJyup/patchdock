package pipeline

import "github.com/HJyup/patchdock/internal/types"

// Reporter observes stage transitions so a caller can show progress
type Reporter interface {
	StageStarted(stage types.StageName, attempt int)
	StageActivity(activity string)
	StageFinished(stage types.StageName, attempt int, note string, err error)
}

type emptyReporter struct{}

func (emptyReporter) StageStarted(types.StageName, int)                 {}
func (emptyReporter) StageActivity(string)                              {}
func (emptyReporter) StageFinished(types.StageName, int, string, error) {}
