import { StageSchema } from "../src/context.ts";
import { readFile, writeFile } from "node:fs/promises";
import { INPUT_FILE, IO_PATH, OUTPUT_FILE, SPECS } from "./mounts/io.ts";
import { REPO_PATH } from "./mounts/repo.ts";
import { WORKSPACE_PATH } from "./mounts/workspace.ts";
import { definitionSchema } from "../src/definitions/schema.ts";

async function main() {
  // Take what stage is currently running from the env file
  const envStage = StageSchema.safeParse(process.env.PATCHDOCK_STAGE);

  if (!envStage.success) {
    console.error(`Invalid PATCHDOCK_STAGE: ${process.env.PATCHDOCK_STAGE}`);
    process.exit(1);
  }

  const stage = envStage.data;

  // Get correct values from the io mount
  const raw = await readFile(`${IO_PATH}/${INPUT_FILE}`, "utf8");
  const input = SPECS[stage].input.parse(JSON.parse(raw));
  const mod: unknown = await import(`/agents/${stage}.ts`);

  if (typeof mod !== "object" || mod === null || !("default" in mod)) {
    throw new Error(`Agent module for stage "${stage}" has no default export`);
  }

  const definition = mod.default;
  const agents = definitionSchema.parse(definition);

  if (agents.stage !== stage) {
    throw new Error(
      `Imported agent doesn't correspond to executed stage: ${agents.stage} != ${stage}`,
    );
  }

  const ctx = {
    stage: stage,
    taskId: process.env.PATCHDOCK_TASK_ID ?? "",
    // Depending on agent we wanna restrict it.
    // IO is never passed since it's open by default to agents
    paths: { repo: REPO_PATH, workspace: WORKSPACE_PATH },
    log: (msg: string) => process.stderr.write(`[${stage}] ${msg}\n`),
  };

  const draft = await agents.run(ctx, input);
  const output = SPECS[stage].output.parse(draft);
  await writeFile(`${IO_PATH}/${OUTPUT_FILE}`, JSON.stringify(output));
}

main().then(
  () => process.exit(0),
  (err) => {
    process.stderr.write(`${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
  },
);
