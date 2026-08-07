package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/HJyup/patchdock/internal/daemon"
	"github.com/HJyup/patchdock/internal/daemon/api"
	"github.com/HJyup/patchdock/internal/daemon/client"
	"github.com/HJyup/patchdock/internal/lock"
	"github.com/HJyup/patchdock/internal/runtimedir"
	"github.com/spf13/cobra"
)

var (
	stopTimeout      = daemon.ShutdownTimeout + 5*time.Second
	stopPollInterval = 100 * time.Millisecond
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Inspect and control the local daemon",
	Long: `The daemon owns the queue, the Docker client and the pipeline. Clients
			start it on demand, so these commands exist for inspecting it and for
			running it in the foreground while debugging.`,
}

var daemonRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the daemon in the foreground",
	Long: `Serves the control API on the unix socket in ~/.patchdock and stays in
			the foreground: logs go to this terminal and Ctrl-C drains and stops it.
			Every client starts the daemon on its own, so this is a debug entry point
			rather than the normal way to bring it up.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := runtimedir.Default()
		if err != nil {
			return err
		}

		return daemon.RunServer(cmd.Context(), dir)
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report daemon health",
	Long: `Prints uptime, version, queue depth, running count and Docker
			reachability. Exits non-zero when no daemon is running.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := runtimedir.Default()
		if err != nil {
			return err
		}

		health, err := client.New(dir.Socket()).Health(cmd.Context())
		if err != nil {
			if errors.Is(err, client.ErrNoDaemon) || errors.Is(err, client.ErrNotListening) {
				return errNoDaemon
			}
			return err
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
		fmt.Fprintf(w, "status\t%s\n", health.Status)
		fmt.Fprintf(w, "uptime\t%s\n", health.Uptime)
		fmt.Fprintf(w, "pid\t%d\n", health.PID)
		fmt.Fprintf(w, "socket\t%s\n", dir.Socket())
		fmt.Fprintf(w, "logs\t%s\n", dir.Log())
		return w.Flush()
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Drain and stop the daemon",
	Long: `Stops accepting new work, lets the running stage finish, removes any
			containers it owns and exits.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := runtimedir.Default()
		if err != nil {
			return err
		}

		pid, err := lock.Owner(dir.Lock())
		if err != nil {
			if errors.Is(err, lock.ErrNotHeld) {
				return errNoDaemon
			}
			return err
		}

		proc, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("find daemon process %d: %w", pid, err)
		}

		if err := proc.Signal(syscall.SIGTERM); err != nil {
			if errors.Is(err, os.ErrProcessDone) {
				return errNoDaemon
			}
			return fmt.Errorf("signal daemon %d: %w", pid, err)
		}

		if err := awaitExit(cmd.Context(), dir.Lock(), pid); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "stopped daemon (pid %d)\n", pid)
		return nil
	},
}

var daemonQueueCmd = &cobra.Command{
	Use:   "queue <text>",
	Short: "Push a string onto the queue",
	Long: `Sends {"data": "<text>"} to the daemon, which logs it and nothing else.
			A scratch command for exercising the queue and watching it land in
			"daemon dev-view"; it goes away once runs are the thing being queued.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := runtimedir.Default()
		if err != nil {
			return err
		}

		c := client.New(dir.Socket())
		if err := c.Queue(cmd.Context(), api.QueueRequest{Data: args[0]}); err != nil {
			if errors.Is(err, client.ErrNoDaemon) || errors.Is(err, client.ErrNotListening) {
				return errNoDaemon
			}
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "queued")
		return nil
	},
}

// awaitExit blocks until pid no longer holds the lock on path
func awaitExit(ctx context.Context, path string, pid int) error {
	ctx, cancel := context.WithTimeout(ctx, stopTimeout)
	defer cancel()

	ticker := time.NewTicker(stopPollInterval)
	defer ticker.Stop()

	for {
		owner, err := lock.Owner(path)
		switch {
		case errors.Is(err, lock.ErrNotHeld):
			return nil
		case err != nil:
			// Held but unreadable, most likely a replacement daemon that has not written its pid yet
			return nil
		case owner != pid:
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"daemon %d did not exit within %s; check %s or kill -9 %d",
				pid, stopTimeout, path, pid,
			)
		case <-ticker.C:
		}
	}
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(
		daemonRunCmd,
		daemonStatusCmd,
		daemonStopCmd,
		daemonQueueCmd,
	)
}
