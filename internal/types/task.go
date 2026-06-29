package types

import (
	"github.com/HJyup/patchdock/internal/id"
)

// Task is an issue/prompt which passed as a first context to the planner
type Task struct {
	ID string `json:"id" validate:"required"`
	// Taken from the GitHub
	Title string `json:"title,omitempty"`
	// Full description of the task we can try to achieve:
	// either GitHub issue description, or just prompt from the user
	Description string `json:"description" validate:"required"`
	// Taken from the GitHub
	Labels []string `json:"labels,omitempty"`
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
