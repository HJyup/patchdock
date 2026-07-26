package pipeline

import "github.com/HJyup/patchdock/internal/types"

// Reporter observes stage transitions so a caller can show progress. The
// pipeline renders nothing itself and does not care whether anyone is watching
type Reporter interface {
	StageStarted(stage types.StageName, attempt int)

	// StageActivity reports what the running stage just did. It fires often and
	// carries no history: each call replaces the last
	StageActivity(activity string)

	// StageFinished closes the stage. note carries a short result worth
	// surfacing, such as the reviewer's decision, and is empty otherwise
	StageFinished(stage types.StageName, attempt int, note string, err error)
}

type emptyReporter struct{}

func (emptyReporter) StageStarted(types.StageName, int)                 {}
func (emptyReporter) StageActivity(string)                              {}
func (emptyReporter) StageFinished(types.StageName, int, string, error) {}
