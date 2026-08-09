package pipeline

import (
	"fmt"
	"time"

	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/stage"
	"github.com/HJyup/patchdock/internal/types"
)

const setupStage = "setup"

type auditRun struct {
	logger  *auditlog.Logger
	rec     auditlog.Record
	failed  types.StageName
	rawKept bool
}

func newAuditRun(logger *auditlog.Logger, task types.Task) *auditRun {
	return &auditRun{
		logger: logger,
		rec: auditlog.Record{
			RunID:     logger.LogID,
			Task:      task,
			StartedAt: time.Now(),
		},
	}
}

func (a *auditRun) Published(branch string) {
	a.rec.Branch = branch
}

func (a *auditRun) Planned(plan types.Plan) {
	a.rec.Plan = plan
}

func (a *auditRun) Patched(stat auditlog.PatchStat) {
	a.rec.Patch = stat
}

func (a *auditRun) Failed(name types.StageName, err error) {
	a.failed = name

	raw := stage.RawOutput(err)
	if len(raw) == 0 {
		return
	}

	if writeErr := a.logger.WriteFailedOutput(raw); writeErr != nil {
		fmt.Fprintf(a.logger, "audit: failed to archive raw output: %v\n", writeErr)
		return
	}
	a.rawKept = true
}

func (a *auditRun) Finish(h *history, out *Outcome, err error) {
	a.rec.Attempts = h.auditAttempts()
	a.rec.Accepted = out.Accepted
	a.rec.Duration = time.Since(a.rec.StartedAt).Round(time.Second).String()

	if err != nil {
		failed := string(a.failed)
		if failed == "" {
			failed = setupStage
		}
		a.rec.Failure = &auditlog.Failure{
			Stage:     failed,
			Message:   err.Error(),
			RawOutput: a.rawKept,
		}
	}

	if writeErr := a.logger.WriteRun(&a.rec); writeErr != nil {
		fmt.Fprintf(a.logger, "audit: failed to write run record: %v\n", writeErr)
	}
}
