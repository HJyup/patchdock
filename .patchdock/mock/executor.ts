import { defineExecutor } from "@patchdock/sdk";
import { writeFile } from "node:fs/promises";
import { join } from "node:path";
import { jitter, pace } from "./lib.ts";

const MIN_SECONDS = 45;
const MAX_SECONDS = 60;
const MARKER_FILE = "MOCK_RUN.md";

export default defineExecutor({
  async run(ctx, input) {
    const workspace = ctx.paths.workspace;
    if (workspace === undefined) {
      throw new Error("executor stage started without a workspace mount");
    }

    const total = jitter(MIN_SECONDS, MAX_SECONDS);

    await pace(ctx, total * 0.6, [
      { source: "agent", event: "process_started" },
      { source: "agent", event: "session_started" },
      {
        source: "agent",
        event: "command_completed",
        command: "git status --short",
      },
      {
        source: "agent",
        event: "tool_call_completed",
        server: "fs",
        tool: "list_dir",
      },
    ]);

    const rejection = input.reviews.at(-1);
    const body = [
      "# Mock run",
      "",
      `- run: ${ctx.runId}`,
      `- attempt: ${ctx.attempt} of ${ctx.maxAttempts}`,
      `- plan: ${input.plan.summary}`,
      `- written: ${new Date().toISOString()}`,
      "",
      rejection === undefined
        ? "First attempt."
        : `Retrying after review feedback: ${rejection.feedback ?? rejection.summary}`,
      "",
      "This file was written by the mock executor. No model was called.",
    ].join("\n");

    await writeFile(join(workspace, MARKER_FILE), `${body}\n`, "utf8");

    ctx.log({
      source: "agent",
      event: "file_change_completed",
      changes: [{ path: MARKER_FILE }],
    });

    await pace(ctx, total * 0.4, [
      {
        source: "agent",
        event: "command_completed",
        command: "git diff --stat",
      },
      { source: "agent", event: "turn_completed" },
    ]);

    return {
      status: "success",
      notes: `Mock executor wrote ${MARKER_FILE} on attempt ${ctx.attempt}.`,
    };
  },
});
