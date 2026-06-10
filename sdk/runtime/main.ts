// Any failure: readable error on stderr, exit 1. The orchestrator treats
// "exit 0 + parseable, contract-valid output.json" as the only success.

import { existsSync } from "node:fs";
import { readFile, rename, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { pathToFileURL } from "node:url";
import { z } from "zod";

import {
  ExecutionResultSchema,
  PlanSchema,
  ReviewFeedbackSchema,
  ZERO_TOKENS,
  type ExecutorInput,
  type ReviewerInput,
} from "../src/index.ts";
import { isStageDefinition } from "../src/define.ts";
import { makeLogger, type StageName } from "../src/context.ts";
import { newId } from "../src/id.ts";

const STAGES: Record<string, { agentFile: string; prefix: StageName }> = {
  planner: { agentFile: "planner.ts", prefix: "planner" },
  executor: { agentFile: "executioner.ts", prefix: "executor" },
  reviewer: { agentFile: "reviewer.ts", prefix: "reviewer" },
};

function fail(msg: string): never {
  console.error(`patchdock-runtime: ${msg}`);
  process.exit(1);
}

function env(name: string, fallback: string): string {
  const v = process.env[name];
  return v !== undefined && v !== "" ? v : fallback;
}

async function waitForFile(path: string, timeoutMs: number): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (!existsSync(path)) {
    if (Date.now() > deadline) {
      fail(`timed out after ${timeoutMs}ms waiting for ${path}`);
    }
    await new Promise((r) => setTimeout(r, 100));
  }
}

function formatZodError(label: string, err: z.ZodError): string {
  const lines = err.issues.map(
    (i) => `  ${label}.${i.path.join(".") || "(root)"}: ${i.message}`,
  );
  return lines.join("\n");
}

async function writeAtomically(path: string, data: string): Promise<void> {
  // The temp file must live in the destination directory: /io is a bind
  // mount, and rename(2) cannot cross filesystems (EXDEV).
  const tmp = `${path}.tmp-${process.pid}`;
  await writeFile(tmp, data, "utf8");
  await rename(tmp, path);
}

async function main(): Promise<void> {
  const stage = env("PATCHDOCK_STAGE", "") as StageName;
  const stageInfo = STAGES[stage];
  if (!stageInfo) {
    fail(
      `PATCHDOCK_STAGE must be one of ${Object.keys(STAGES).join(", ")}; got ${JSON.stringify(stage)}`,
    );
  }

  const ioDir = env("PATCHDOCK_IO", "/io");
  const agentsDir = env("PATCHDOCK_AGENTS", "/agents");
  const agentFile = env(
    "PATCHDOCK_AGENT_FILE",
    join(agentsDir, stageInfo.agentFile),
  );
  const inputTimeoutMs = Number(env("PATCHDOCK_INPUT_TIMEOUT_MS", "60000"));

  const inputPath = join(ioDir, "input.json");
  await waitForFile(inputPath, inputTimeoutMs);

  let rawInput: unknown;
  try {
    rawInput = JSON.parse(await readFile(inputPath, "utf8"));
  } catch (e) {
    fail(`input.json is not valid JSON: ${(e as Error).message}`);
  }

  if (!existsSync(agentFile)) {
    fail(`agent file not found: ${agentFile}`);
  }
  const mod = (await import(pathToFileURL(agentFile).href)) as {
    default?: unknown;
  };
  const def = mod.default;
  if (!isStageDefinition(def)) {
    fail(
      `${agentFile} must default-export definePlanner/defineExecutor/defineReviewer(...)`,
    );
  }
  if (def.stage !== stage) {
    fail(
      `stage mismatch: runtime is running "${stage}" but ${agentFile} defines a "${def.stage}"`,
    );
  }

  const inputParse = def.inputSchema.safeParse(rawInput);
  if (!inputParse.success) {
    fail(
      `input.json failed validation:\n${formatZodError("input", inputParse.error)}`,
    );
  }
  const input = inputParse.data;
  const taskId = (input as { task: { id: string } }).task.id;

  const ctx = {
    taskId,
    stage,
    log: makeLogger(stage),
    paths: {
      io: ioDir,
      repo: env("PATCHDOCK_REPO", "/repo"),
      workspace: env("PATCHDOCK_WORKSPACE", "/workspace"),
    },
  };

  let draftRaw: unknown;
  try {
    draftRaw = await def.run(input, ctx);
  } catch (e) {
    const err = e as Error;
    fail(`agent threw: ${err.stack ?? err.message}`);
  }

  const draftParse = def.outputSchema.safeParse(draftRaw);
  if (!draftParse.success) {
    fail(
      `agent returned an invalid ${stage} result:\n${formatZodError("output", draftParse.error)}`,
    );
  }

  const draft = draftParse.data as Record<string, unknown>;
  const completed: Record<string, unknown> = {
    ...draft,
    id: draft.id ?? newId(stageInfo.prefix),
    task_id: taskId,
    tokens_used: draft.tokens_used ?? ZERO_TOKENS,
  };
  if (stage === "planner") {
    completed.created_at = draft.created_at ?? new Date().toISOString();
  }
  if (stage === "executor") {
    completed.plan_id = (input as ExecutorInput).plan.id;
  }
  if (stage === "reviewer") {
    completed.execution_id = (input as ReviewerInput).execution.id;
  }

  const finalSchema =
    stage === "planner"
      ? PlanSchema
      : stage === "executor"
        ? ExecutionResultSchema
        : ReviewFeedbackSchema;

  const finalParse = finalSchema.safeParse(completed);
  if (!finalParse.success) {
    fail(
      `completed ${stage} contract failed validation (runtime bug or bad draft):\n` +
        formatZodError("output", finalParse.error),
    );
  }

  await writeAtomically(
    join(ioDir, "output.json"),
    JSON.stringify(finalParse.data, null, 2),
  );
  ctx.log.info(`wrote ${stage} output for task ${taskId}`);
}

main().catch((e: unknown) => {
  fail(`unhandled error: ${(e as Error).stack ?? String(e)}`);
});
