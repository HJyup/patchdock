package pipeline

import (
	"github.com/HJyup/patchdock/internal/auditlog"
	"github.com/HJyup/patchdock/internal/types"
)

type history struct {
	Executions []types.ExecutionResult
	Reviews    []types.Review
}

func newHistory() *history {
	return &history{
		Executions: make([]types.ExecutionResult, 0),
		Reviews:    make([]types.Review, 0),
	}
}

func (h *history) AddExecution(execution types.ExecutionResult) {
	h.Executions = append(h.Executions, execution)
}

func (h *history) AddReview(review types.Review) {
	h.Reviews = append(h.Reviews, review)
}

func (h *history) auditAttempts() []auditlog.Attempt {
	attempts := make([]auditlog.Attempt, len(h.Executions))
	for i, execution := range h.Executions {
		attempts[i] = auditlog.Attempt{Number: i + 1, Execution: execution}
		if i < len(h.Reviews) {
			attempts[i].Review = h.Reviews[i]
		}
	}

	return attempts
}
