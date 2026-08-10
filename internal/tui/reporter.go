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

func (r *Reporter) StageSummary(summary string) {
	r.progress.Note(summary)
}

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
