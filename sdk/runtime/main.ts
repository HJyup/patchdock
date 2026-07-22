import {
  StageSchema,
  type StageContext,
  type StageLogEvent,
  parseTokenBudget,
  parseAttempt,
} from "../src/context.ts";
import { readFile, writeFile } from "node:fs/promises";
import { INPUT_FILE, IO_PATH, OUTPUT_FILE } from "../src/mounts/io.ts";
import { REPO_PATH, WORKSPACE_PATH } from "../src/mounts/code.ts";
import { definitionSchema, runAgent } from "../src/agents/index.ts";

function writeStageLog(
  stage: StageContext["stage"],
  entry: string | StageLogEvent,
): void {
  const event: StageLogEvent =
    typeof entry === "string"
      ? { source: "agent", event: "message", level: "info", message: entry }
      : entry;

  process.stderr.write(
    `${JSON.stringify({
      ...event,
      timestamp: new Date().toISOString(),
      stage,
    })}\n`,
  );
}

async function main() {
  // Which stage are we? Comes from the environment (untrusted), so it's parsed.
  const envStage = StageSchema.safeParse(process.env.PATCHDOCK_STAGE);
  if (!envStage.success) {
    console.error(`Invalid PATCHDOCK_STAGE: ${process.env.PATCHDOCK_STAGE}`);
    process.exit(1);
  }
  const stage = envStage.data;

  const raw: unknown = JSON.parse(await readFile(`${IO_PATH}/${INPUT_FILE}`, "utf8"));

  // The host names the agent file via config (stages: planner: planner.ts);
  // fall back to the conventional name when it doesn't.
  const agentFile = process.env.PATCHDOCK_AGENT_FILE ?? `${stage}.ts`;
  const mod: unknown = await import(`/agents/${agentFile}`);
  if (typeof mod !== "object" || mod === null || !("default" in mod)) {
    throw new Error(
      `Agent module "${agentFile}" for stage "${stage}" has no default export`,
    );
  }

  const agents = definitionSchema.parse(mod.default);
  if (agents.stage !== stage) {
    throw new Error(
      `Imported agent doesn't correspond to executed stage: ${agents.stage} != ${stage}`,
    );
  }

  const ctx: StageContext = {
    stage,
    taskId: process.env.PATCHDOCK_TASK_ID ?? "",
    // IO is never passed since it's defined by default to agents
    paths: { repo: REPO_PATH, workspace: WORKSPACE_PATH },
    log: (entry) => writeStageLog(stage, entry),
    tokenBudget: parseTokenBudget(process.env.PATCHDOCK_TOKEN_BUDGET),
    attempt: parseAttempt(process.env.PATCHDOCK_ATTEMPT),
    maxAttempts: parseAttempt(process.env.PATCHDOCK_MAX_ATTEMPTS),
  };

  const output = await runAgent(agents, ctx, raw);
  await writeFile(`${IO_PATH}/${OUTPUT_FILE}`, JSON.stringify(output));
}

main().then(
  () => process.exit(0),
  (err) => {
    process.stderr.write(
      `${JSON.stringify({
        timestamp: new Date().toISOString(),
        stage: process.env.PATCHDOCK_STAGE ?? "unknown",
        source: "runtime",
        event: "fatal_error",
        level: "error",
        message: err instanceof Error ? err.message : String(err),
      })}\n`,
    );
    process.exit(1);
  },
);
