# patchdock

> A typed agent-pipeline runtime. You write the agents; patchdock runs
> them in Docker, routes typed contracts between stages, and gives you
> an audit trail of every run.

## What patchdock does

Patchdock drives a fixed pipeline against a code repository:

    task → planner → executor → checks → reviewer → patch

The user writes three TypeScript files defining what each stage does;
patchdock provides the runtime that executes them. Each stage runs in
its own Docker container. Contracts flow between stages as typed JSON.
Tasks can run concurrently — many issues in flight at once, one
container per stage per task.

## How a user uses it

1. **`patchdock init`** in any repo. Creates a `.patchdock/` directory
   with `config.yml`, a `Dockerfile` describing how to replicate the
   codebase in a container, and three TypeScript files
   (`planner.ts`, `executioner.ts`, `reviewer.ts`).
2. **Edit the agent files.** Each one exports a default function with a
   typed signature provided by the patchdock SDK. Prompts, models,
   tools, and logic are user-defined.
3. **`patchdock`** opens a terminal UI. The user picks a GitHub issue
   (or enters a prompt) and watches the pipeline execute. Multiple
   tasks can run in parallel.
4. **The runtime executes:** the planner reads the task and the repo
   (read-only) and produces a `Plan`. The executor receives the Plan,
   modifies a sandboxed copy of the repo, and produces an
   `ExecutionResult` with the resulting patch. Deterministic checks
   (npm test, go vet, etc.) run against the patch. The reviewer
   evaluates the execution against the plan and the check results, and
   either accepts or rejects.
5. **On accept,** patchdock applies the diff and opens a pull request
   on GitHub. On reject, the executor runs again against the same plan
   with the reviewer's feedback as additional input. Retries are
   bounded by configuration.
6. **Every run produces an audit log** at `.patchdock/logs/<run-id>/`:
   each stage's input, output, container log, token usage, and a
   manifest summary. Runs are fully reconstructable from disk.

## Features

### Configurable agents

Each stage's behavior is defined by a user-owned TypeScript file. The
SDK exposes typed inputs and outputs (`Plan`, `ExecutionResult`,
`ReviewFeedback`), prompt helpers, model selection, and tool wiring.
The user controls prompts, the model used per stage, which tools are
available, and the agent's logic.

### Typed contracts between stages

Stages communicate only through three typed JSON contracts —
`Plan`, `ExecutionResult`, `ReviewFeedback`. Every contract is
validated on the way out of one stage and on the way into the next.
Failures surface at the boundary with the broken field named.

### Concurrent task execution

Multiple tasks run in parallel, each with its own Docker container per
stage. Logs from concurrent containers are streamed live, demultiplexed
by task. The user can launch one task or fan out across every open
issue in a repository.

### Docker-isolated execution

The planner sees the target repository read-only. The executor works in
a sandboxed overlay of the repository — modifications cannot escape
the workspace. The orchestrator extracts a clean diff after the
executor exits. Each container has configurable memory, CPU, and
wall-clock limits.

### Deterministic checks

Between executor and reviewer, user-configured commands (test suites,
linters, type checkers) run inside a container against the executor's
diff. Failures are surfaced to the reviewer as blocking issues.

### Reviewer with structured feedback

The reviewer produces a typed decision (`accept` or `reject`) and, on
reject, a list of structured `Issues` — each with severity, location
(file and line range), and a suggested fix. The executor retries
against the same plan with the issues folded into its input. The loop
is bounded by configured retry limits.

### Per-run audit log

Every run writes a complete record to `.patchdock/logs/<run-id>/`:
manifest summary, per-attempt input and output JSON, container logs,
token usage per stage, and the final patch. Runs are append-only and
never mutated — six months later the chain of decisions is fully
reconstructable.

### Terminal UI

A Bubble Tea–based terminal interface presents an issue picker or
prompt entry, a multi-pane view of concurrent tasks, live log tails,
plan and diff inspection, and accept/reject gates.

### GitHub integration

Issues fetched from GitHub serve as task input. Accepted runs open a
pull request against the repository. Authentication via a personal
access token.

## Architecture

Data flows left to right; everything depends on `contracts`, nothing
depends on `tui`:

    cmd/patchdock ──► tui ──► pipeline (orchestrator)
                                  │
            ┌──────────┬──────────┼──────────┬─────────┐
            ▼          ▼          ▼          ▼         ▼
        workspace   agentio    checks    auditlog   github
            │          │          │
            └──────────┴── docker ┘
                           │
                      contracts  (shared by all)

| Package | Responsibility |
|---|---|
| `cmd/patchdock` | CLI entry, subcommands, dependency wiring |
| `internal/contracts` | the three typed contracts + validation |
| `internal/docker` | image build, container run, mounts, limits, log streaming |
| `internal/config` | load + validate `.patchdock/config.yml` |
| `internal/workspace` | per-task repo copies, mounts, diff extraction |
| `internal/agentio` | file-based contract exchange with agent containers |
| `internal/pipeline` | the orchestrator: stage order, retry loop, token caps |
| `internal/checks` | run deterministic checks, produce a `CheckReport` |
| `internal/auditlog` | append-only `.patchdock/logs/<run-id>/` record |
| `internal/github` | issues in, pull requests out |
| `internal/scaffold` | `patchdock init` templates (`go:embed`) |
| `internal/tui` | Bubble Tea UI, pure consumer of pipeline events |
| `sdk/` | TypeScript SDK + agent runtime + base Docker image |

## Mental model

Patchdock is a **pipeline runtime**, not an autonomous coding agent.
The runtime has no opinion on how the planner thinks, what model the
executor uses, or what the reviewer optimizes for. Those are the user's
decisions, expressed in the user's TypeScript files. Patchdock's job
is to route typed values between isolated processes safely,
concurrently, and reproducibly — and to give the user a complete
record of what happened.
