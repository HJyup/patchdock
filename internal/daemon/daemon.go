package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HJyup/patchdock/internal/daemon/broker"
	"github.com/HJyup/patchdock/internal/daemon/config"
	"github.com/HJyup/patchdock/internal/daemon/queue"
	"github.com/HJyup/patchdock/internal/docker"
	"github.com/HJyup/patchdock/internal/lock"
	"github.com/HJyup/patchdock/internal/runtimedir"
	"github.com/HJyup/patchdock/internal/transport"
)

var (
	ReadTimeout     = 5 * time.Second
	ShutdownTimeout = 10 * time.Second
)

func RunServer(ctx context.Context, dir runtimedir.Dir) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	l, err := lock.Acquire(dir.Lock())
	if err != nil {
		if errors.Is(err, lock.ErrHeld) {
			return fmt.Errorf("daemon already running (lock held on %s)", dir.Lock())
		}
		return fmt.Errorf("acquire lock %s: %w", dir.Lock(), err)
	}
	defer func() {
		log.Print("releasing lock")
		l.Release()
	}()

	if err := os.Remove(dir.Socket()); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("remove stale socket %s: %v", dir.Socket(), err)
	}

	listener, err := transport.Listen(dir.Socket())
	if err != nil {
		return err
	}
	defer listener.Close()

	cfg, err := config.Load(dir.Config())
	if err != nil {
		return err
	}

	// Constructing the client does not dial the daemon — connectivity failures
	// still surface per run, where they name the stage that hit them.
	cli, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("connect to docker: %w", err)
	}
	defer cli.Close()

	q := queue.New(ctx, pipelineRunner(cli), cfg)
	ch := q.Snaps()

	b := broker.New(ch)

	go q.Run()
	go b.Run(ctx)

	service := NewService(q, dir, b)
	router := NewRouter(service)
	srv := &http.Server{
		Handler:           router,
		ReadHeaderTimeout: ReadTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", dir.Socket())
		errCh <- srv.Serve(listener)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)

	case <-ctx.Done():
		log.Println("shutting down")
		shutCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
		return nil
	}
}
