package tui

import (
	"fmt"
	"strings"

	"github.com/HJyup/patchdock/internal/types"
)

const (
	nameWidth = 9
)

type Reporter struct {
	progress *Progress
}

func NewReporter(progress *Progress) *Reporter {
	return &Reporter{progress: progress}
}

func (r *Reporter) StageChange(stage types.StageName, attempt int) {
	r.progress.Start(stageLabel(stage, attempt))
}

func (r *Reporter) StageActivity(activity string) {
	r.progress.Detail(activity)
}

func (r *Reporter) StageNote(note string) {
	r.progress.Note(note)
}

// stageLabel renders the name column. The attempt is printed from the second
// onward: a snapshot has room for one stage and one number, so the count cannot
// be left to the reader to infer from repeated rows
func stageLabel(stage types.StageName, attempt int) string {
	name := string(stage)
	if name != "" {
		name = strings.ToUpper(name[:1]) + name[1:]
	}

	label := fmt.Sprintf("%-*s", nameWidth, name)
	if attempt > 1 {
		label += fmt.Sprintf(" (attempt %d)", attempt)
	}

	return label
}
