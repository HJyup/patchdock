package types

import (
	"errors"
	"fmt"
)

// errs collects contract violations as "path: problem" lines, reported in
// field-declaration order so an agent (or a repair prompt) sees every
// problem at once instead of fixing them one round-trip at a time.
type errs struct{ list []error }

func (e *errs) addf(format string, args ...any) {
	e.list = append(e.list, fmt.Errorf(format, args...))
}

func (e *errs) required(path, value string) {
	if value == "" {
		e.addf("%s: empty", path)
	}
}

// join returns nil when no violations were recorded.
func (e *errs) join() error { return errors.Join(e.list...) }
