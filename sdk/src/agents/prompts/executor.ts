import type { StageContext } from "../../context.ts";
import type { ExecutorInput } from "../../types.ts";
import { sharedPrompt } from "./index.ts";

export function executorPrompt(ctx: StageContext, input: ExecutorInput): string {
  const reviews =
    input.reviews.length > 0
      ? `Previous attempts were rejected. The reviews below tell you exactly what to
fix — address every point of the most recent feedback:
${JSON.stringify(input.reviews, null, 2)}\n`
      : "";

  return `You are the executor stage of an automated coding pipeline called Patchdock.
Your current working directory is a writable workspace clone of the repository.
Implement the plan below by editing files directly. Do not commit, push, or
create branches — the pipeline extracts your changes as a diff afterwards.

${sharedPrompt(ctx)}
Run the repository's relevant focused checks after editing. In your notes, distinguish
checks that passed, checks that failed, and checks you could not run.
Patchdock owns the Git worktree and extracts the final diff outside this container, so do
not depend on workspace Git metadata being available.

Plan (JSON):
${JSON.stringify(input.plan, null, 2)}

${reviews}Follow the plan's steps and satisfy its acceptance criteria.

Your FINAL message must be ONLY a JSON object, no prose around it:
{"status": "success" | "partial_success" | "failed", "notes": "<what you did, or what stopped you>"}
Use "success" only when every step is done. Use "" for notes when there is nothing to add.`;
}
