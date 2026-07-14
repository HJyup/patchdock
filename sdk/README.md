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

Install the SDK directly in a TypeScript project:

```bash
pnpm i patchdock-sdk
```

Or initialize a repository from the main Patchdock instance:

```bash
patchdock init
```

`patchdock init` creates `.patchdock/config.yml` and starter files for the planner,
executor, and reviewer. This is the recommended way to begin because the generated files
match the Patchdock runtime configuration.

## Define agents and their contracts

Patchdock has three stage definitions:

- `definePlanner`
- `defineExecutor`
- `defineReviewer`

Every configured agent file must default-export the matching definition. The examples
below show the intended built-in Codex adapter. Codex support is currently in development.

### `definePlanner`

The planner receives a task and produces the plan that drives the rest of the pipeline.

```typescript
import { definePlanner } from "patchdock-sdk";
import { codex } from "patchdock-sdk/agents/codex";

export default definePlanner({
  run: (ctx, input) => codex(ctx, input),
});
```

Planner input and output:

```typescript
interface PlannerInput {
  task: Task;
}

interface PlanData {
  approach: string;
  acceptance_criteria: string[];
  steps: Array<{
    id: string;
    description: string;
    rationale?: string;
    files_to_modify?: string[];
  }>;
  context?: string[];
  assumptions?: string[];
}

type PlannerRun = (ctx: StageContext, input: PlannerInput) => Promise<PlanData>;
```

Patchdock adds the plan ID, task ID, and creation timestamp after the planner returns.

### `defineExecutor`

The executor receives the plan and previous review feedback, then works in the writable
workspace.

```typescript
import { defineExecutor } from "patchdock-sdk";
import { codex } from "patchdock-sdk/agents/codex";

export default defineExecutor({
  run: (ctx, input) => codex(ctx, input),
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
  step_results: Array<{
    step_id: string;
    status: "success" | "partial_success" | "failed";
    notes?: string;
  }>;
  errors?: Array<{
    step_id?: string;
    message: string;
  }>;
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
import { defineReviewer } from "patchdock-sdk";
import { codex } from "patchdock-sdk/agents/codex";

export default defineReviewer({
  run: (ctx, input) => codex(ctx, input),
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
  issues?: Array<{
    severity: "blocker" | "major" | "minor";
    message: string;
    step_id?: string;
    file_path?: string;
    line_range?: string;
    suggestion?: string;
  }>;
}

type ReviewerRun = (ctx: StageContext, input: ReviewerInput) => Promise<ReviewData>;
```

Patchdock adds the review ID, task ID, and latest execution ID after the reviewer
returns.

## Customising agent behaviour

The built-in model adapter is optional. A definition can run any code the project needs:

- Read and transform the typed input.
- Inspect the preassigned stage context.
- Read from the stage's mounted repository or workspace.
- Call a different model provider.
- Use a local model.
- Call tools, services, or project-specific functions.
- Combine deterministic logic with model output.

For example, a project can replace the Codex adapter with its own executor:

```typescript
import {
  defineExecutor,
  type ExecutorInput,
  type ExecutionResultData,
  type StageContext,
} from "patchdock-sdk";
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

Patchdock does not prescribe how the agent reasons or produces its answer. The strict
boundary is the returned output: it must satisfy `PlanData`, `ExecutionResultData`, or
`ReviewData` for the pipeline to continue.

## Supported models

Patchdock does not ship a completed built-in model adapter yet.

| Model  | Status         | Planned import                |
| ------ | -------------- | ----------------------------- |
| Codex  | In development | `patchdock-sdk/agents/codex`  |
| Claude | Planned        | `patchdock-sdk/agents/claude` |

Built-in agents will use the same context and contracts as custom agents. Switching from
a custom implementation to a supported model should not require changes to the
surrounding pipeline.

## Into the details

### Define your own agent

An agent file is a TypeScript module with one default export:

```typescript
import { definePlanner } from "patchdock-sdk";
import { runPlanner } from "./planner-implementation";

export default definePlanner({
  run: runPlanner,
});
```

The configured filename must match the stage mapping in `.patchdock/config.yml`:

```yaml
stages:
  planner: planner.ts
  executor: executor.ts
  reviewer: reviewer.ts
```

Agent `run` functions are asynchronous. They receive `ctx` first and the stage input
second. Their returned value is the only typed output passed back to Patchdock.

The main Patchdock instance consumes an agent as follows:

```text
write typed input
    -> mount configured agent
    -> import its default definition
    -> validate input
    -> call run(ctx, input)
    -> validate output
    -> enrich runtime-owned fields
    -> pass result to the next stage
```

### Context

Patchdock constructs `StageContext` before invoking an agent:

```typescript
type Stage = "planner" | "executor" | "reviewer";

interface StageContext {
  stage: Stage;
  taskId: string;
  paths: {
    repo?: string;
    workspace?: string;
  };
  tokenBudget: number | null;
  attempt: number;
  maxAttempts: number;
  log: (message: string) => void;
}
```

- `stage` identifies which definition is running.
- `taskId` identifies the current Patchdock task.
- `paths` contains the conventional mount locations available to the stage.
- `tokenBudget` contains the configured budget or `null` when unlimited.
- `attempt` and `maxAttempts` let retry-aware agents adapt their behaviour.
- `log(message)` writes a stage-prefixed message into the Patchdock audit log.

Use `ctx.log` for progress and diagnostic information:

```typescript
ctx.log(`Running ${ctx.stage} for task ${ctx.taskId}`);
```

Logs are not contract output. Patchdock captures stdout and stderr for the audit trail,
but only the object returned from `run` is passed to the pipeline.

### Mounts

Mounts are capabilities assigned by the main Patchdock runtime:

| Stage        | Path         | Access     | Purpose                                                |
| ------------ | ------------ | ---------- | ------------------------------------------------------ |
| Planner      | `/repo`      | Read-only  | Inspect the original repository while producing a plan |
| Executor     | `/workspace` | Read-write | Modify the isolated repository clone                   |
| Reviewer     | `/workspace` | Read-only  | Inspect the executor's resulting workspace             |
| All stages   | `/agents`    | Read-only  | Load configured agent modules                          |
| Runtime only | `/io`        | Read-write | Exchange validated input and output JSON               |

Use the context paths instead of hardcoding mount locations:

```typescript
const repoPath = ctx.paths.repo;
const workspacePath = ctx.paths.workspace;
```

Only use the path assigned to the current stage. A conventional path value does not mean
that the corresponding directory is mounted for every stage.

Changes made outside `/workspace` are container-local and disappear when the container
is removed. Executor changes must be written under `ctx.paths.workspace` so Patchdock
can extract them.

### Contract and validation rules

Contracts are checked at two boundaries:

1. The TypeScript SDK validates input before the agent runs and validates its returned
   output afterward.
2. The Go host enriches the result with runtime-owned fields and validates the complete
   domain contract again.

Validation failures stop the stage. Invalid data is never passed to the next agent.

The main rules agent authors need to respect are:

- Default-export one definition matching the configured stage.
- Return the output type belonging to that definition.
- Use snake-case JSON field names such as `acceptance_criteria` and `step_results`.
- Planner output requires a non-empty approach, acceptance criteria, and steps.
- Executor status and step-result status must be `success`, `partial_success`, or
  `failed`.
- Reviewer decision must be `accept` or `reject`.
- A rejected review must contain actionable issues; an accepted review must not contain
  issues.
- Do not return runtime-owned IDs, timestamps, stage relationships, or the executor patch.
- Write executor file changes only into the writable workspace.

Inside those boundaries, custom agents are free to parse input, call models, use tools,
and organize their behaviour however the project requires.

## Development checks

When changing the SDK or its examples, run:

```bash
cd sdk
pnpm typecheck
pnpm lint
pnpm test
pnpm format:check
```

The files generated by `patchdock init` should remain aligned with the definitions and
contracts documented here.
