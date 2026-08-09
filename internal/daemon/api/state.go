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
	ID     string `json:"id"`
	TaskID string `json:"task_id"`

	// Repo is the absolute path the run targets. Runs are grouped by it
	Repo string `json:"repo"`

	// Title is the first line of the task, for display only
	Title  string `json:"title"`
	Status Status `json:"status"`

	// Reason explains a non-obvious status: why a queued run has not started
	// ("repo busy"), or why one was cancelled
	Reason string `json:"reason,omitempty"`

	Attempt  int `json:"attempt,omitempty"`
	MaxTries int `json:"max_tries"`

	QueuedAt   time.Time  `json:"queued_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	// Patch is the diff size of the most recent attempt
	Patch  *auditlog.PatchStat `json:"patch,omitempty"`
	Branch string              `json:"branch,omitempty"`
}

func (r Run) Clone() Run {
	out := r

	if r.StartedAt != nil {
		t := *r.StartedAt
		out.StartedAt = &t
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

// Snapshot is the whole daemon state at one instant
type Snapshot struct {
	At   time.Time `json:"at"`
	Runs []Run     `json:"runs"`
}
