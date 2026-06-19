# patchdock

> A typed, sandboxed runtime for running coding agents. You drive it from
> Claude Code over MCP — or from the CLI — and it runs the work in isolated
> Docker containers, routes typed contracts between stages, and hands you a
> complete audit trail of every run.

## What patchdock is

Patchdock is the **engine room** behind your coding agent. Claude Code (or
Codex, or your own agent) is the **cockpit**. Patchdock is what the cockpit *delegates to* when the work needs
to be **isolated, parallel, unattended, or recorded**: it runs each stage of the
work in its own Docker container, enforces typed contracts between stages, and
hands back a structured result plus an audit trail.

It is the runtime that lets you *offload* coding work — *"run this in a sandbox, I'll keep
working"* — and run many such tasks at once, without melting your laptop or
touching your working tree.

Work flows through up to three stages:

    task → planner → executor → checks → reviewer → result (diff + audit)

Each stage can be backed by a **real coding agent** (Claude Code / Codex running
headless) or a **custom TypeScript agent** written against the SDK. Stages talk
only through typed JSON contracts. Tasks run concurrently — one container per
stage per task.

## Two front doors, one engine

Patchdock exposes the same building blocks at two altitudes:

- **MCP server** — Claude Code connects and calls `run_task`, `run_planner`,
  `run_executor`, `run_reviewer` as tools. Execution is **async**: a call
  returns a `run_id` immediately and the work runs in the background, so you
  fire off a sandboxed task and keep working. The result and an audit pointer
  come back when it lands. This is the offload story.
- **CLI** — `patchdock run` drives the **blessed end-to-end pipeline** for
  unattended / batch use, with the runtime owning the loop (gated checks,
  bounded retries).

The MCP primitives and the CLI pipeline are the same stages composed two ways:
**à la carte** when you want control, the **set-menu** pipeline when you want
guarantees. Strict data, flexible flow.

## How a user uses it

1. **`patchdock init`** in any repo. Creates a `.patchdock/` directory with a
   `config.yml`, a `Dockerfile` describing how to replicate the codebase in a
   container, and agent definition files.
2. **Configure.** In `config.yml` the user sets per-container limits (wall-clock
   timeout, token budget), the maximum number of concurrent containers, and the
   deterministic checks to run against the executor's diff. Each stage is
   pointed at an agent backend — a real coding agent or a custom SDK agent.
3. **Drive it.** Either connect Claude Code to the MCP server and offload work,
   or run the full pipeline from the CLI.
4. **The runtime executes.** The planner reads the task and the repo
   (read-only) and produces a `Plan`. The executor receives the Plan, modifies a
   sandboxed git clone of the repo, and produces an `ExecutionResult` (the diff).
   Deterministic checks run against the diff. The reviewer evaluates the result
   against the plan and the checks, and accepts or rejects.
5. **On reject,** the executor retries against the same plan with the reviewer's
   structured feedback folded in. Retries are bounded by config.
6. **Every run produces an audit log** at `.patchdock/logs/<run-id>/`: each
   stage's typed input and output, container logs, token usage, and a manifest.
   Runs are append-only and fully reconstructable.

## Features

### Pluggable agent backends

Each stage is backed by an agent: a real coding agent (Claude Code or Codex run
headless inside the container) or a custom TypeScript agent written against the
SDK. Patchdock has no opinion on how a stage thinks — it provides the typed
input, the sandbox, and the budget, and collects the typed output.

### Typed contracts between stages

Stages communicate only through three typed JSON contracts — `Plan`,
`ExecutionResult`, `Review`. Each is validated on the way out of one stage and
into the next; failures surface at the boundary with the broken field named.
These same contracts are what Claude Code reasons over when it orchestrates —
they are the **orchestration interface**, not just data hygiene.

### MCP front door + async offload

Patchdock runs as an MCP server. Claude Code calls the stages and the pipeline as
tools. Execution is non-blocking: a call returns a `run_id` immediately and the
work runs in the background, so you fire a sandboxed task and keep working.
Results come back as a structured value **+ a short summary + a pointer to the
audit log** — never raw logs dumped into context.

### Concurrent, bounded execution

Many tasks run in parallel, each with its own container per stage, capped by a
configured max-concurrency. Logs from concurrent containers are streamed live
and demultiplexed by task. Run one task, or fan out — best-of-N attempts at one
problem, or a whole batch overnight.

### Docker-isolated execution with hard budgets

The planner sees the repo read-only; the executor works in a sandboxed git clone
it cannot escape; patchdock extracts a clean diff after it exits. Every container
has a configured **wall-clock timeout** and **token budget**, so a run can't burn
your night or your wallet.

### Deterministic checks

Between executor and reviewer, user-configured commands (tests, linters, type
checkers) run in a container against the diff. Failures are surfaced to the
reviewer as blocking issues.

### Reviewer with structured feedback

The reviewer produces a typed `accept`/`reject` and, on reject, a list of
structured `Issues` — each with severity, location, and a suggested fix. The
executor retries against the same plan with the issues folded into its input.
Bounded by config.

### Per-run audit log

Every run writes a complete, append-only record to `.patchdock/logs/<run-id>/`:
manifest summary, per-attempt typed input and output, container logs, token
usage, and the final patch. The audit log is the **source of truth**; the result
Claude Code sees is a structured projection of it.

## Architecture
| Package | Responsibility |
|---|---|
| `cmd/patchdock` | CLI entry, subcommands, MCP server launch, dependency wiring |
| `internal/mcp` | MCP server: `run_task` / `run_planner` / `run_executor` / `run_reviewer`, async run store, handles |
| `internal/types` | the three typed contracts + validation |
| `internal/config` | load + validate `.patchdock/config.yml` (limits, concurrency, checks, agent backends) |
| `internal/docker` | image build, container run, mounts, limits, log streaming |
| `internal/workspace` | per-task git-clone sandbox, mounts, diff extraction |
| `internal/stage` | run a single stage in a container, typed file-based IO exchange |
| `internal/checks` | run deterministic checks against the diff, produce a `CheckReport` |
| `internal/pipeline` | the blessed composition: stage order, retry loop, budgets |
| `internal/scheduler` | bounded-concurrency orchestration of many runs |
| `internal/auditlog` | append-only `.patchdock/logs/<run-id>/` record |
| `sdk/` | TypeScript SDK + agent runtime + base image; agent backends (custom + Claude Code / Codex adapters) |

## Mental model

Patchdock is a **runtime, not an agent**. Claude Code is the cockpit; patchdock
is the engine room. The runtime has no opinion on how the planner thinks or what
model the executor uses — those are the user's decisions, expressed in config and
agent files. Patchdock's job is to route typed values between isolated containers
safely, concurrently, and reproducibly, and to hand the orchestrator — you, or
Claude Code — a complete record of what happened.
