# patchdock

## Commands

| Command | What it does |
| --- | --- |
| `dock init` | Scaffold `.patchdock/` in the current repo: config, starter agents, Dockerfile. |
| `dock` | Full-screen view of every run across every repo. Read-only — quitting never stops a run. |
| `dock run -p "<prompt>"` | Queue a run in the current repo and attach to it. |
| `dock run` | Open the repo-scoped view with a prompt input, then attach to what you submit. |
| `dock run -d -p "<prompt>"` | Queue and exit, printing the `run_id`. |
| `dock watch` | Same view as `dock`. |
| `dock watch <run-id>` | Open focused on a single run — the way back after detaching. |
| `dock cancel <run-id>` | Cancel a queued or running run. |
| `dock daemon status` | Daemon health: uptime, version, queue depth, running count, Docker reachability. |
| `dock daemon stop` | Drain and stop the daemon. |
| `dock daemon run` | Run the daemon in the foreground, for debugging. |

### `dock run` flags

| Flag | Meaning |
| --- | --- |
| `-p, --prompt` | The task. Without it, `dock run` opens an input. |
| `-d, --detach` | Queue and exit instead of attaching. |
| `--repo <path>` | Target a repo other than the current directory. |

### Behaviour worth knowing

- Every command routes through the daemon's queue. `dock run` with and without `-d`
  differ only in whether the CLI stays attached.
- Any client starts the daemon on demand — except `dock daemon status`, which reports
  that none is running rather than starting one.
- Ctrl-C while attached detaches; the run keeps going. Stop it with `dock cancel`.
- Evidence for a run lands in `.patchdock/logs/<run-id>/`.
- Commands that need a terminal print help instead when stdout is redirected.
