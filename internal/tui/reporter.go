package tui

import (
	"fmt"
	"strings"

	"github.com/HJyup/patchdock/internal/types"
)

const (
	nameWidth = 9
	noteWidth = 8
)

type Reporter struct {
	progress *Progress
}

func NewReporter(progress *Progress) *Reporter {
	return &Reporter{progress: progress}
}

func (r *Reporter) StageStarted(stage types.StageName, _ int) {
	r.progress.Start(stageLabel(stage))
}

func (r *Reporter) StageActivity(activity string) {
	r.progress.Detail(activity)
}

func (r *Reporter) StageNote(note string) {
	r.progress.Note(note)
}

func (r *Reporter) StageFinished(_ types.StageName, _ int, note string, err error) {
	if err == nil && note == string(types.ReviewReject) {
		r.progress.FinishRetry(shortNote(note))
		return
	}

	r.progress.Finish(shortNote(note), err)
}

// stageLabel renders the name column. A retry repeats the executor and reviewer
// rows, which is itself the record of how many attempts a run took, so the
// number is left off rather than printed on every line that never varies
func stageLabel(stage types.StageName) string {
	name := string(stage)
	if name != "" {
		name = strings.ToUpper(name[:1]) + name[1:]
	}

	return fmt.Sprintf("%-*s", nameWidth, name)
}

// shortNote keeps the result column narrow. The unabbreviated status is in
// run.json; this cell only has to tell the statuses apart
func shortNote(note string) string {
	if note == string(types.ExecutionPartialSuccess) {
		return "partial"
	}

	return note
}
