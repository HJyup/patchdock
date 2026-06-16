package pipeline

import "github.com/HJyup/patchdock/internal/types"

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
