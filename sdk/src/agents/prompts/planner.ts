import type { StageContext } from "../../context.ts";
import type { PlannerInput } from "../../types.ts";
import { sharedPrompt } from "./index.ts";

export function plannerPrompt(ctx: StageContext, input: PlannerInput): string {
  return `You are the planner stage of an automated coding pipeline called Patchdock.
The target repository is mounted read-only at your current working directory.
Explore it as needed, but do not modify any files — planning is a read-only stage.

${sharedPrompt(ctx)}
Your job: produce an implementation plan for the task below. A separate executor
agent will follow your plan literally, so be specific about files and steps.

Task (JSON):
${JSON.stringify(input.task, null, 2)}

Write the plan body as markdown with exactly these sections:
## Approach — the strategy in a few sentences
## Steps — a numbered list of concrete edits (file paths where possible)
## Acceptance criteria — bullet points a reviewer can check objectively

Your FINAL message must be ONLY a JSON object, no prose around it:
{"summary": "<one-line description of the plan>", "body": "<the markdown plan>"}
Both fields are required and must be non-empty.`;
}
