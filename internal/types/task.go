package types

import (
	"github.com/HJyup/patchdock/internal/id"
)

// Task is an issue/prompt which passed as a first context to the planner
type Task struct {
	ID    string `json:"id" validate:"required"`
	Title string `json:"title,omitempty"`
	// Description is the full task: either a GitHub issue body or a user prompt.
	Description string   `json:"description" validate:"required"`
	Labels      []string `json:"labels,omitempty"`
}

func (t *Task) validate() error { return validateStruct(t, "task") }

func NewTask(t Task) (Task, error) {
	if t.ID == "" {
		t.ID = id.New("task")
	}
	if err := t.validate(); err != nil {
		return Task{}, err
	}
	return t, nil
}
