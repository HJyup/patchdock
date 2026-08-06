package cli

import (
	"github.com/spf13/cobra"
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
		// TODO: resolve the runtime dir, point the standard logger at a live
		// tui view, and hand off to daemon.RunServer.
		return errNotImplemented
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report daemon health",
	Long: `Prints uptime, version, queue depth, running count and Docker
			reachability. Exits non-zero when no daemon is running.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: client.Health, then format it.
		//
		// This is the one command that must NOT go through the connect-or-start
		// ladder: starting a daemon in order to report whether one is running
		// makes the "not running" state unobservable.
		return errNotImplemented
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Drain and stop the daemon",
	Long: `Stops accepting new work, lets the running stage finish, removes any
			containers it owns and exits.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: read the PID from the lock file, send SIGTERM, wait for the
		// socket to disappear with a short timeout.
		//
		// Deliberately a signal rather than an HTTP route: it still works when
		// the daemon is wedged inside a handler and cannot answer requests.
		return errNotImplemented
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonRunCmd, daemonStatusCmd, daemonStopCmd)
}
