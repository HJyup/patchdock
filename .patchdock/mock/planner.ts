import { definePlanner } from "@patchdock/sdk";
import { firstLine, jitter, pace } from "./lib.ts";

const MIN_SECONDS = 15;
const MAX_SECONDS = 20;

export default definePlanner({
  async run(ctx, input) {
    const title = firstLine(input.task.description);

    await pace(ctx, jitter(MIN_SECONDS, MAX_SECONDS), [
      { source: "agent", event: "process_started" },
      { source: "agent", event: "session_started" },
      {
        source: "agent",
        event: "command_completed",
        command: "rg -n 'TODO' .",
      },
      {
        source: "agent",
        event: "tool_call_completed",
        server: "fs",
        tool: "read_file",
      },
      { source: "agent", event: "turn_completed" },
    ]);

    return {
      summary: `Mock plan: ${title}`,
      body: [
        `Mock plan for task ${ctx.taskId}.`,
        "",
        `Request: ${title}`,
        "",
        "Steps:",
        "1. Write a marker file into the workspace.",
        "2. Hand the diff to the reviewer.",
        "",
        "No model was called to produce this plan.",
      ].join("\n"),
    };
  },
});
