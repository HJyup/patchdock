package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/client"
	"github.com/HJyup/patchdock/internal/runtimedir"
)

var retryTimeout = time.Second

// Connect returns a client for the daemon, starting the daemon if it is not
// already running or listening.
func Connect(ctx context.Context, dir *runtimedir.Dir) (*client.Client, error) {
	c := client.New(dir.Socket())

	if _, err := c.Health(ctx); err != nil {
		if !errors.Is(err, client.ErrNoDaemon) && !errors.Is(err, client.ErrNotListening) {
			return nil, err
		}

		if err := spawn(dir); err != nil {
			return nil, err
		}

		return c, wait(ctx, c)
	}

	return c, nil
}

func wait(ctx context.Context, c *client.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	for {
		_, err := c.Health(ctx)

		if err == nil {
			return nil
		}

		log.Printf("retrying health check: %v", err)

		select {
		case <-ctx.Done():
			return fmt.Errorf("failed to connect to daemon: %w", ctx.Err())
		case <-time.After(retryTimeout):
		}
	}
}

func spawn(dir *runtimedir.Dir) error {
	self, err := os.Executable()
	if err != nil {
		return err
	}

	logf, err := os.OpenFile(dir.Log(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer logf.Close()

	cmd := exec.Command(self, "daemon", "run")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Dir = dir.Root()
	cmd.Stdin = nil
	cmd.Stdout, cmd.Stderr = logf, logf

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Process.Release()
}
