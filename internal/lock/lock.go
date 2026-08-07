package lock

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

var (
	ErrHeld    = errors.New("the daemon process have already acquired this file")
	ErrNotHeld = errors.New("lock is not held")
)

type Lock struct {
	file *os.File
}

// File lock to ensure that only one OS process runs
// The holder's PID is written into the file so that other processes can find
// the daemon without asking it.
func Acquire(path string) (*Lock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
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

	if err := writePID(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("record pid in %s: %w", path, err)
	}

	return &Lock{file: f}, nil
}

func writePID(f *os.File) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.WriteAt([]byte(strconv.Itoa(os.Getpid())), 0); err != nil {
		return err
	}
	return f.Sync()
}

// Owner returns the PID of the process currently holding the lock, or ErrNotHeld if it is free
func Owner(path string) (int, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return 0, ErrNotHeld
		}
		return 0, err
	}
	defer f.Close()

	err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == nil {
		unix.Flock(int(f.Fd()), unix.LOCK_UN)
		return 0, ErrNotHeld
	}

	if !errors.Is(err, unix.EWOULDBLOCK) {
		return 0, fmt.Errorf("probe lock %s: %w", path, err)
	}

	raw, err := io.ReadAll(f)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		return 0, fmt.Errorf("lock %s is held but records no usable pid", path)
	}

	return pid, nil
}

func (l *Lock) Release() error {
	return l.file.Close()
}
