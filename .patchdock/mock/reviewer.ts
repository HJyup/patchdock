import { defineReviewer } from "@patchdock/sdk";
import { jitter, pace } from "./lib.ts";

const MIN_SECONDS = 10;
const MAX_SECONDS = 15;
const REJECT_CHANCE = 0.25;

export default defineReviewer({
  async run(ctx, input) {
    const bytes = input.patch.length;

    if (input.patch.trim() === "") {
      ctx.log({
        source: "agent",
        event: "message",
        level: "warn",
        message: "reviewer received an empty patch",
      });
    }

    await pace(ctx, jitter(MIN_SECONDS, MAX_SECONDS), [
      { source: "agent", event: "process_started" },
      { source: "agent", event: "session_started" },
      {
        source: "agent",
        event: "command_completed",
        command: `git diff --stat  # ${bytes} bytes`,
      },
      { source: "agent", event: "turn_completed" },
    ]);

    if (Math.random() < REJECT_CHANCE) {
      return {
        decision: "reject",
        summary: `Mock reviewer rejected attempt ${ctx.attempt}`,
        feedback: [
          "Rejected by the mock reviewer on a coin flip, not on the diff.",
          `Attempt ${ctx.attempt} of ${ctx.maxAttempts}, patch was ${bytes} bytes.`,
        ].join(" "),
      };
    }

    return {
      decision: "accept",
      summary: `Mock reviewer accepted attempt ${ctx.attempt} (${bytes}-byte patch)`,
    };
  },
});
