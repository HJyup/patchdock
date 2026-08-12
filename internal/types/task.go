package types

type Task struct {
	Title string `json:"title,omitempty"`
	// Description is the full task: either a GitHub issue body or a user prompt.
	Description string   `json:"description"`
	Labels      []string `json:"labels,omitempty"`
}

func NewTask(t Task) (Task, error) {
	if err := t.validate(); err != nil {
		return Task{}, err
	}
	return t, nil
}

func (t *Task) validate() error {
	var e errs
	e.required("task.description", t.Description)
	return e.join()
}
