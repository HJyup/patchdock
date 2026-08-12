package api

import (
	"time"

	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/types"
)

// Status is what a run is doing right now
type Status string

const (
	// StatusQueued accepted, waiting for a slot
	StatusQueued Status = "queued"

	// StatusStarted admitted, but the pipeline has not reported a stage yet
	StatusStarted Status = "started"

	StatusPlanning   Status = "planning"
	StatusCoding     Status = "coding"
	StatusReviewing  Status = "reviewing"
	StatusPublishing Status = "publishing"

	// StatusSucceeded the reviewer accepted an attempt and the branch is pushed
	StatusSucceeded Status = "succeeded"

	// StatusRejected every attempt was rejected. Not an error: the pipeline ran
	// to completion and the reviewer said no
	StatusRejected Status = "rejected"

	// StatusFailed a stage errored, or the run never got off the ground
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// StatusForStage maps a pipeline stage onto the status a watcher should see
func StatusForStage(stage types.StageName) Status {
	switch stage {
	case types.StagePlanner:
		return StatusPlanning
	case types.StageExecutor:
		return StatusCoding
	case types.StageReviewer:
		return StatusReviewing
	default:
		return StatusStarted
	}
}

// Run is the daemon's record of one queued or running task
type Run struct {
	ID       string `json:"id"`
	Repo     string `json:"repo"`
	Title    string `json:"title"`
	Status   Status `json:"status"`
	Activity string `json:"activity,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Attempt  int    `json:"attempt,omitempty"`

	QueuedAt  time.Time  `json:"queued_at"`
	StartedAt *time.Time `json:"started_at,omitempty"`

	StageStartedAt *time.Time          `json:"stage_started_at,omitempty"`
	FinishedAt     *time.Time          `json:"finished_at,omitempty"`
	Patch          *auditlog.PatchStat `json:"patch,omitempty"`
	Branch         string              `json:"branch,omitempty"`
}

func (r Run) Clone() Run {
	out := r

	if r.StartedAt != nil {
		t := *r.StartedAt
		out.StartedAt = &t
	}
	if r.StageStartedAt != nil {
		t := *r.StageStartedAt
		out.StageStartedAt = &t
	}
	if r.FinishedAt != nil {
		t := *r.FinishedAt
		out.FinishedAt = &t
	}
	if r.Patch != nil {
		p := *r.Patch
		out.Patch = &p
	}

	return out
}

type Snapshot struct {
	At   time.Time `json:"at"`
	Runs []Run     `json:"runs"`
}

func IsFinilised(s Status) bool {
	switch s {
	case StatusSucceeded, StatusRejected, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}
