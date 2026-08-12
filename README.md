# Patchdock

Patchdock turns a prompt into a reviewed patch. Describe a task, and agents
work through it inside an isolated Docker container; the result lands as a
commit on a `patchdock/…` branch in your repository, ready to review and merge.

<img width="1501" height="644" alt="example screen" src="https://github.com/user-attachments/assets/1d36e6ea-a430-4f52-bd9d-b52570dac3a3" />

## Getting started

> [!NOTE]
> Patchdock is not yet distributed through Homebrew or npm. Distribution will
> be added after critical work is complete, including Docker runtime
> hardening, cancellation support, and daemon lifecycle improvements.

Building Patchdock requires Go; running it requires a Docker Engine. Install
the `dock` binary from source:

```bash
git clone https://github.com/HJyup/patchdock.git
cd patchdock
go install ./cmd/dock
```

Make sure the Go binary directory is on your `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

Then initialise the repository you want agents to work on:

```bash
cd your-repo
dock init
```

This creates a `.patchdock/` directory with the configuration and agent
definitions Patchdock uses during execution:

```text
.patchdock/
├── config.yml
├── Dockerfile
├── planner.ts
├── executor.ts
└── reviewer.ts
```

Adapt the Dockerfile to your repository's toolchain, then open Patchdock and
submit your first task:

```bash
dock
```

## How it works

The `dock` CLI talks to a local daemon over a unix socket. The daemon queues
runs, drives Docker execution, and streams live state back to the clients.

Every task moves through three agents: the planner produces a plan, the
executor applies it to the repository, and the reviewer judges the changes.
A rejected review sends the task back to the executor until the configured
retry limit is reached; an accepted review is published as a branch and
commit.

```mermaid
stateDiagram-v2
    [*] --> Planning

    state "Planner agent" as Planning
    state "Executor agent" as Executing
    state "Reviewer agent" as Reviewing
    state "Publish branch" as Publishing
    state "Run succeeded" as Succeeded
    state "Run rejected" as Rejected
    state "Run failed" as Failed

    Planning --> Executing: Valid plan produced
    Planning --> Failed: Error

    Executing --> Reviewing: Changes and execution result produced
    Executing --> Failed: Error

    Reviewing --> Publishing: Review accepted
    Reviewing --> Executing: Changes requested and attempts remain
    Reviewing --> Rejected: Retry limit reached
    Reviewing --> Failed: Error

    Publishing --> Succeeded: Branch and commit created
    Publishing --> Failed: Error

    Succeeded --> [*]
    Rejected --> [*]
    Failed --> [*]
```

## Configuration

### Dockerfile

Each repository owns a `.patchdock/Dockerfile`. It defines the isolated
environment in which the planner, executor, and reviewer operate.

Add everything the agents need to build, test, and inspect the repository:
language runtimes, package managers, compilers, system libraries, project tools,
and prewarmed dependencies. These additions become part of the agent image and
are available during every stage.

### config.yml

The repository's `.patchdock/config.yml` defines the execution guardrails for
its agents. For example:

```yaml
container:
  timeout: 10m
  token_budget: 100000

retries:
  max: 2
```

The container timeout is a hard wall-clock limit for each stage. The token
budget is passed to the agent as an advisory budget, while the retry limit
controls how many executor and reviewer rounds may run. The configuration also
selects stage files, declares read-only credential mounts, names the reusable
agent image, and controls the Git branch prefix.

## Patchdock Agent SDK

The TypeScript files in `.patchdock/` use `@patchdock/sdk` to define typed
planner, executor, and reviewer contracts. Each stage file decides how its
agent is driven inside the container.

### Supported agents

The table below lists the coding agents that can run inside the container:

| Agent | Status |
| --- | --- |
| Codex | Supported |
| Claude | Coming soon |

For agent contracts, custom implementations, runtime context, and configuration
details, read the [Patchdock Agent SDK documentation](./sdk/README.md).

## CLI reference

### Initialise a repository

```bash
dock init
```

Creates the repository's `.patchdock/` directory. Use `dock init --force` to
regenerate it; this overwrites the existing configuration and agent files.

### Open Patchdock

```bash
dock
```

Opens the interactive terminal interface on the task input. Submit tasks there
and switch to the live dashboard to follow runs across repositories.

### Watch runs

```bash
dock watch
```

Opens the terminal interface directly on the live dashboard.

### Submit a detached task

```console
dock "Update the API error handling"
run-4e6b30262e44
```

Passing an inline prompt queues the task, starts the daemon on demand if
necessary, prints the run ID, and exits without opening the terminal interface.

Use `--repo` to target another repository:

```bash
dock --repo ../another-project "Add request validation"
```

### Control the daemon

The daemon owns the run queue, Docker execution, and live state. Clients start
it automatically when needed, while these commands let you inspect or control
it directly:

| Command | Description |
| --- | --- |
| `dock daemon status` | Show daemon health, uptime, process ID, socket, and log path. |
| `dock daemon run` | Run the daemon in the foreground for debugging. |
| `dock daemon stop` | Signal the running daemon to stop and wait for it to exit. |

## References

- [Architecture](./ARCHITECTURE.md): how Patchdock works in detail, covering
  the daemon, the live state feed, and the anatomy of a pipeline run.
- [Patchdock Agent SDK](./sdk/README.md): agent contracts, custom
  implementations, runtime context, and configuration details.
