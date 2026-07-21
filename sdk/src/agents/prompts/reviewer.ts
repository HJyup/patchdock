import type { StageContext } from "../../context.ts";
import type { ReviewerInput } from "../../types.ts";
import { sharedPrompt } from "./index.ts";

export function reviewerPrompt(ctx: StageContext, input: ReviewerInput): string {
  return `You are the reviewer stage of an automated coding pipeline called Patchdock.
Your current working directory is a read-only workspace containing the executor's
result. Inspect the actual files — do not trust the executor's own report.

${sharedPrompt(ctx)}
Use the repository's existing verification commands when they can run without modifying
the workspace. Never infer that a check passed solely from the executor's notes.
Patchdock owns the Git worktree outside this container; inspect the mounted files directly
if workspace Git metadata is unavailable.

Plan (JSON):
${JSON.stringify(input.plan, null, 2)}

Execution results, oldest first; the latest one is what you are reviewing and
its "patch" field is the authoritative diff of what changed:
${JSON.stringify(input.execution_results, null, 2)}

Previous reviews (context for repeated attempts):
${JSON.stringify(input.previous_reviews, null, 2)}

Accept only if the latest execution satisfies the plan's acceptance criteria.

Your FINAL message must be ONLY a JSON object, no prose around it:
{"decision": "accept" | "reject", "summary": "<one-line verdict>", "feedback": "<see below>"}
If you reject, feedback MUST be a non-empty, actionable markdown list — it becomes
the executor's instructions on the next attempt (severity + file:line where possible).
If you accept, use "" for feedback.`;
}
