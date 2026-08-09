package pipeline

import "github.com/HJyup/patchdock/internal/types"

// Reporter observes stage transitions so a caller can show progress
type Reporter interface {
	StageStarted(stage types.StageName, attempt int)
	StageActivity(activity string)
	StageNote(note string)
	StageFinished(stage types.StageName, attempt int, note string, err error)
}

type stubReporter struct{}

func (stubReporter) StageStarted(types.StageName, int)                 {}
func (stubReporter) StageActivity(string)                              {}
func (stubReporter) StageNote(string)                                  {}
func (stubReporter) StageFinished(types.StageName, int, string, error) {}
