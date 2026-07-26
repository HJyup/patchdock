package auditlog

import (
	"strings"
	"time"

	"github.com/HJyup/patchdock/internal/types"
)

type Record struct {
	RunID     string     `json:"run_id"`
	Task      types.Task `json:"task"`
	Plan      types.Plan `json:"plan"`
	Attempts  []Attempt  `json:"attempts"`
	Accepted  bool       `json:"accepted"`
	Failure   *Failure   `json:"failure,omitempty"`
	StartedAt time.Time  `json:"started_at"`
	Duration  string     `json:"duration"`
	Patch     PatchStat  `json:"patch"`
}

type Attempt struct {
	Number    int                   `json:"number"`
	Execution types.ExecutionResult `json:"execution"`
	Review    types.Review          `json:"review"`
}

type Failure struct {
	Stage     string `json:"stage"`
	Message   string `json:"message"`
	RawOutput bool   `json:"raw_output"`
}

type PatchStat struct {
	Files     int `json:"files"`
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
}

func StatPatch(diff string) PatchStat {
	var stat PatchStat
	for line := range strings.SplitSeq(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			stat.Files++
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		case strings.HasPrefix(line, "+"):
			stat.Additions++
		case strings.HasPrefix(line, "-"):
			stat.Deletions++
		}
	}
	return stat
}
