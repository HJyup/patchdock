# Patchdock SDK

Patchdock SDK is the typed authoring layer for agents that run inside a Patchdock
pipeline. It lets a project define planner, executor, and reviewer agents while the main
Patchdock instance continues to own orchestration, containers, mounts, retries, audit
logs, and runtime validation.

The SDK keeps agent code focused on one contract:

```text
typed input + preassigned context -> agent -> typed output
```

Patchdock supplies the input and context. The agent may use any logic or model internally,
but its output must satisfy the stage contract so the next pipeline stage can consume it.

## Installation

`dock init` is the recommended way to begin: it scaffolds `.patchdock/` with starter
agent files already matched to the runtime configuration. See
[Getting started](../README.md#getting-started).

To add the SDK to an existing TypeScript project instead:

```bash
pnpm add @patchdock/sdk
```

## Define agents and their contracts

Every agent file must default-export one of the three stage definitions:
`definePlanner`, `defineExecutor`, or `defineReviewer`. The examples below use the
built-in Codex adapter.

### `definePlanner`

The planner receives a task and produces the plan that drives the rest of the pipeline.

```typescript
import { codex, definePlanner } from "@patchdock/sdk";

export default definePlanner({
  async run(ctx, input) {
    return codex(ctx, input);
  },
});
```

Planner input and output:

```typescript
interface PlannerInput {
  task: Task;
}

interface PlanData {
  summary: string; // 1-2 sentences, shown in run results
  body: string; // markdown: the full plan
}

type PlannerRun = (ctx: StageContext, input: PlannerInput) => Promise<PlanData>;
```

Patchdock adds the creation timestamp after the planner returns. Nothing in the
contracts carries an ID: the run ID (`ctx.runId`) and the attempt number address
every stage output, so agents never mint or echo identifiers.

Structure inside `body` (approach, ordered steps, acceptance criteria) is a prompt
convention for the executor and reviewer to read, not a schema. Keep the conventional
headings so downstream stages know where to look.

### `defineExecutor`

The executor receives the plan and previous review feedback, then works in the writable
workspace.

```typescript
import { codex, defineExecutor } from "@patchdock/sdk";

export default defineExecutor({
  async run(ctx, input) {
    return codex(ctx, input);
  },
});
```

Executor input and output:

```typescript
interface ExecutorInput {
  plan: Plan;
  reviews: Review[];
}

interface ExecutionResultData {
  status: "success" | "partial_success" | "failed";
  notes?: string; // markdown: what was done, what worked, what didn't
}

type ExecutorRun = (
  ctx: StageContext,
  input: ExecutorInput,
) => Promise<ExecutionResultData>;
```

The executor does not return a patch. It modifies files under `ctx.paths.workspace`, and
the main Patchdock process extracts the authoritative git diff after execution.

### `defineReviewer`

The reviewer receives the plan and execution history, then returns an accept or reject
decision.

```typescript
import { codex, defineReviewer } from "@patchdock/sdk";

export default defineReviewer({
  async run(ctx, input) {
    return codex(ctx, input);
  },
});
```

Reviewer input and output:

```typescript
interface ReviewerInput {
  plan: Plan;
  execution_results: ExecutionResult[];
  previous_reviews: Review[];
}

interface ReviewData {
  decision: "accept" | "reject";
  summary: string;
  feedback?: string; // markdown; required when decision is "reject"
}

type ReviewerRun = (ctx: StageContext, input: ReviewerInput) => Promise<ReviewData>;
```

On reject, `feedback` becomes the executor's context for the next attempt. By
convention, list each issue with a severity and file:line reference so the retry knows
exactly what to fix.

## Customising agent behaviour

Codex does not own the stage definition. The project can inspect or transform the typed
context and input before deciding how to invoke it:

```typescript
import { codex, defineExecutor } from "@patchdock/sdk";

export default defineExecutor({
  async run(ctx, input) {
    ctx.log(`Starting executor attempt ${ctx.attempt}/${ctx.maxAttempts}`);

    if (input.reviews.length > 0) {
      ctx.log("Passing previous review feedback to Codex");
    }

    return codex(ctx, input);
  },
});
```

The built-in model adapter is optional. A definition can instead run any code the project
needs: a different provider, a local model, project-specific tools and services, or
deterministic logic combined with model output. For example, a project can replace the
Codex adapter with its own executor:

```typescript
import {
  defineExecutor,
  type ExecutorInput,
  type ExecutionResultData,
  type StageContext,
} from "@patchdock/sdk";
import { runMyModel } from "./my-model";
import { toExecutorOutput } from "./contracts";

async function run(
  ctx: StageContext,
  input: ExecutorInput,
): Promise<ExecutionResultData> {
  const result = await runMyModel({ ctx, input });
  return toExecutorOutput(result);
}

export default defineExecutor({ run });
```

The strict boundary is the returned output: it must satisfy `PlanData`,
`ExecutionResultData`, or `ReviewData` for the pipeline to continue.

## Supported models

| Model  | Status   | Import           |
| ------ | -------- | ---------------- |
| Codex  | Built in | `@patchdock/sdk` |
| Claude | Planned  | n/a              |

Built-in adapters use the same context and contracts as custom agents, so swapping
between them requires no changes to the surrounding pipeline.

## Into the details

### How an agent is loaded

Each agent filename must match the stage mapping in `.patchdock/config.yml`:

```yaml
stages:
  planner: planner.ts
  executor: executor.ts
  reviewer: reviewer.ts
```

The main Patchdock instance then consumes the file as follows:

```text
write typed input
    -> mount configured agent
    -> import its default definition
    -> validate input
    -> call run(ctx, input)
    -> validate output
    -> stamp the plan's creation timestamp
    -> pass result to the next stage
```

### Context

Patchdock constructs `StageContext` before invoking an agent:

```typescript
type Stage = "planner" | "executor" | "reviewer";

interface StageContext {
  stage: Stage;
  runId: string;
  paths: {
    repo?: string;
    workspace?: string;
  };
  tokenBudget: number | null;
  attempt: number;
  maxAttempts: number;
  log: (entry: string | StageLogEvent) => void;
}

interface StageLogEvent {
  source: string;
  event: string;
  level?: "debug" | "info" | "warn" | "error";
  message?: string;
  [field: string]: unknown;
}
```

- `stage` identifies which definition is running.
- `runId` identifies the current run. It is the only identity Patchdock assigns, and it
  names the run's audit log directory, published branch, and stage containers.
- `paths` contains the conventional mount locations available to the stage.
- `tokenBudget` contains the configured budget or `null` when unlimited.
- `attempt` and `maxAttempts` let retry-aware agents adapt their behaviour.
- `log(entry)` writes a structured event or a plain agent message into the Patchdock audit log.

Use `ctx.log` for progress and diagnostic information:

```typescript
ctx.log(`Running ${ctx.stage} for run ${ctx.runId}`);

ctx.log({
  source: "my-agent",
  event: "verification_completed",
  level: "info",
  command: "pnpm test",
  exit_code: 0,
});
```

The stage audit stream is JSON Lines. Plain strings are wrapped as `agent/message` events. The
Codex adapter records lifecycle, command, file-change, tool-call, error, and token-usage
summaries; it deliberately excludes reasoning, agent prose, command output, tool
arguments/results, and patch bodies. Logs are not contract output: only the object returned
from `run` is passed to the pipeline.

### Runtime toolchains

The scaffolded agent image targets TypeScript and JavaScript repositories: Debian
Bookworm with Node.js 22, npm, `tsx`, Git, ripgrep, `make`, and `curl`, and no C
toolchain, Python, or other language runtime. `PATCHDOCK_TOOLCHAIN_SUMMARY` in the
Dockerfile is injected verbatim into every agent prompt, so a repository that needs a
different toolchain must extend the image and that summary together.

Codex is also instructed to inspect repository manifests and lockfiles, prefer existing
repository scripts, run focused checks after editing, and report missing tools or unrun
checks instead of implying verification succeeded.

### Mounts

Mounts are capabilities assigned by the main Patchdock runtime:

| Stage        | Path         | Access     | Purpose                                                |
| ------------ | ------------ | ---------- | ------------------------------------------------------ |
| Planner      | `/repo`      | Read-only  | Inspect the original repository while producing a plan |
| Executor     | `/workspace` | Read-write | Modify the isolated repository clone                   |
| Reviewer     | `/workspace` | Read-only  | Inspect the executor's resulting workspace             |
| All stages   | `/agents`    | Read-only  | Load configured agent modules                          |
| Runtime only | `/io`        | Read-write | Exchange validated input and output JSON               |

Read locations from `ctx.paths` instead of hardcoding them, and only use the path
assigned to the current stage. Changes made outside `/workspace` are container-local and
disappear with the container, so executor edits must be written under
`ctx.paths.workspace` for Patchdock to extract them.

### Contract and validation rules

Contracts are checked at two boundaries:

1. The TypeScript SDK validates input before the agent runs and validates its returned
   output afterward.
2. The Go host validates the same domain contract again before the result reaches the
   next stage or the audit record.

Validation failures stop the stage. Invalid data is never passed to the next agent.

The main rules agent authors need to respect are:

- Default-export one definition matching the configured stage.
- Return the output type belonging to that definition.
- Use snake-case JSON field names such as `execution_results` and `previous_reviews`.
- Planner output requires a non-empty `summary` and `body`.
- Executor `status` must be `success`, `partial_success`, or `failed`.
- Reviewer `decision` must be `accept` or `reject`; a rejected review must carry
  non-empty `feedback` (accepted reviews may include it for non-blocking notes).
- Do not return identifiers, timestamps, or the executor patch. The run ID and the
  attempt number are the runtime's to assign, and both already reach you on `ctx`.
- Write executor file changes only into the writable workspace.

## Development checks

When changing the SDK or its examples, run:

```bash
cd sdk
pnpm typecheck
pnpm lint
pnpm test
pnpm format:check
```

The files generated by `dock init` should remain aligned with the definitions and
contracts documented here.
