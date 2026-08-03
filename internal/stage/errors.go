package stage

import (
	"errors"
	"fmt"
)

const (
	reasonNotJSON  = "is not valid JSON"
	reasonContract = "violates the contract"
)

type ErrContainer struct {
	ExitCode int64
}

func (e ErrContainer) Error() string {
	return fmt.Sprintf("container exited with code %d", e.ExitCode)
}

type ErrOutputMissing struct {
	Path string
}

func (e ErrOutputMissing) Error() string {
	return fmt.Sprintf("container exited 0 but wrote no output: %s does not exist", e.Path)
}

type ErrOutput struct {
	Reason string
	Err    error
	Raw    []byte
}

func (e ErrOutput) Error() string {
	return fmt.Sprintf("output %s: %v", e.Reason, e.Err)
}

func (e ErrOutput) Unwrap() error { return e.Err }

// RawOutput returns the bytes behind a rejected output, or nil for any other error
func RawOutput(err error) []byte {
	if e, ok := errors.AsType[ErrOutput](err); ok {
		return e.Raw
	}
	return nil
}
