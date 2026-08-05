package client

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"syscall"
)

var (
	ErrNoDaemon     = errors.New("no daemon running")
	ErrNotListening = errors.New("stale socket: no daemon listening")
	ErrPermission   = errors.New("permission denied on daemon socket")
)

type ClientError struct {
	Code int
	Body string
}

func (e *ClientError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("daemon returned %s", http.StatusText(e.Code))
	}
	return fmt.Sprintf("daemon returned %d: %s", e.Code, e.Body)
}

func DeamonError(err error) error {
	switch {
	case errors.Is(err, syscall.ENOENT), errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%w: %v", ErrNoDaemon, err)
	case errors.Is(err, syscall.ECONNREFUSED):
		return fmt.Errorf("%w: %v", ErrNotListening, err)
	case errors.Is(err, syscall.EACCES), errors.Is(err, syscall.EPERM):
		return fmt.Errorf("%w: %v", ErrPermission, err)
	default:
		return err
	}
}
