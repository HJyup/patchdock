package tui

import (
	"fmt"
	"strings"

	"github.com/HJyup/patchdock/internal/types"
)

const (
	nameWidth    = 9
	attemptWidth = 2
	noteWidth    = 8
)

type Reporter struct {
	progress *Progress
}

func NewReporter(progress *Progress) *Reporter {
	return &Reporter{progress: progress}
}

func (r *Reporter) StageStarted(stage types.StageName, attempt int) {
	r.progress.Start(stageLabel(stage, attempt))
}

func (r *Reporter) StageActivity(activity string) {
	r.progress.Detail(activity)
}

func (r *Reporter) StageFinished(_ types.StageName, _ int, note string, err error) {
	if err == nil && note == string(types.ReviewReject) {
		r.progress.FinishRetry(shortNote(note))
		return
	}

	r.progress.Finish(shortNote(note), err)
}

// stageLabel renders the name and attempt columns. The planner runs once per
// task and passes attempt 0, leaving its attempt cell blank
func stageLabel(stage types.StageName, attempt int) string {
	name := string(stage)
	if name != "" {
		name = strings.ToUpper(name[:1]) + name[1:]
	}

	number := ""
	if attempt > 0 {
		number = fmt.Sprintf("%d", attempt)
	}

	return fmt.Sprintf("%-*s %-*s", nameWidth, name, attemptWidth, number)
}

// shortNote keeps the result column narrow. The unabbreviated status is in
// run.json; this cell only has to tell the statuses apart
func shortNote(note string) string {
	if note == string(types.ExecutionPartialSuccess) {
		return "partial"
	}

	return note
}
