package daemon

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HJyup/patchdock/internal/lock"
	"github.com/HJyup/patchdock/internal/runtimedir"
	"github.com/HJyup/patchdock/internal/transport"
)

var ReadTimeout = 5 * time.Second

func RunServer(ctx context.Context, dir runtimedir.Dir) {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	l, err := lock.Acquire(dir.Lock())
	if err != nil {
		if errors.Is(err, lock.ErrHeld) {
			log.Fatalf("daemon already running (lock held on %v)", dir.Lock())
		}
		log.Fatalf("unexpected error while trying to get a lock")
	}
	defer func() {
		log.Print("Removing lock...")
		l.Release()
	}()

	if err := os.Remove(dir.Socket()); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("remove stale socket %s: %v", dir.Socket(), err)
	}

	listener, err := transport.Listen(dir.Socket())
	if err != nil {
		log.Fatalf("failed to create a connection to sock file")
	}
	defer listener.Close()

	service := NewService(dir)
	router := NewRouter(service)
	srv := &http.Server{
		Handler:           router,
		ReadHeaderTimeout: ReadTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("Server is listening on unix sock: %v", dir.Socket())
		errCh <- srv.Serve(listener)
	}()

	select {
	case err := <-errCh:
		log.Printf("server died %v", err)
	case <-ctx.Done():
		log.Println("Graceful shutdown...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		srv.Shutdown(shutCtx)
	}
}
