package stage

import (
	"fmt"
	"strings"
)

// ErrContainer reports an agent container that exited non-zero.
type ErrContainer struct {
	ExitCode   int64
	StderrTail []string
}

func (e ErrContainer) Error() string {
	if len(e.StderrTail) == 0 {
		return fmt.Sprintf("container exited with code %d", e.ExitCode)
	}
	return fmt.Sprintf("container exited with code %d, stderr tail:\n%s",
		e.ExitCode, strings.Join(e.StderrTail, "\n"))
}

// ErrOutputMissing reports a container that exited 0 without writing output.json
type ErrOutputMissing struct {
	Path string
}

func (e ErrOutputMissing) Error() string {
	return fmt.Sprintf("container exited 0 but wrote no output: %s does not exist", e.Path)
}

// ErrOutputNotJSON reports an output.json whose bytes could not be parsed.
type ErrOutputNotJSON struct {
	Err error
}

func (e ErrOutputNotJSON) Error() string {
	return fmt.Sprintf("output is not valid JSON: %v", e.Err)
}

func (e ErrOutputNotJSON) Unwrap() error { return e.Err }

// ErrContractInvalid reports an output that parsed as JSON but failed contract validation.
type ErrContractInvalid struct {
	Err error
}

func (e ErrContractInvalid) Error() string {
	return fmt.Sprintf("output violates the contract: %v", e.Err)
}

func (e ErrContractInvalid) Unwrap() error { return e.Err }
