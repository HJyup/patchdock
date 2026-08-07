package lock

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

var ErrHeld = errors.New("the daemon process have already acquired this file")

type Lock struct {
	file *os.File
}

// File lock to ensure that only one OS process runs
// Aka distributed lock
func Acquire(path string) (*Lock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0o600)
	if err != nil {
		return nil, err
	}

	err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		f.Close()
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, ErrHeld
		}
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}

	return &Lock{file: f}, nil
}

func (l *Lock) Release() error {
	return l.file.Close()
}
