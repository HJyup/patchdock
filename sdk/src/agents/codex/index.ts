import type { StageContext } from "../../context.ts";
import { SPECS } from "../../mounts/io.ts";
import type {
  ExecutionResultData,
  ExecutorInput,
  PlanData,
  PlannerInput,
  ReviewData,
  ReviewerInput,
} from "../../types.ts";
import { executorPrompt } from "../prompts/executor.ts";
import { plannerPrompt } from "../prompts/planner.ts";
import { reviewerPrompt } from "../prompts/reviewer.ts";
import { runCodex } from "./exec.ts";
import {
  executorOutputSchema,
  plannerOutputSchema,
  reviewerOutputSchema,
  type JsonSchema,
} from "./schemas.ts";

export interface CodexOptions {
  model?: string;
}

export function codex(
  ctx: StageContext,
  input: PlannerInput,
  options?: CodexOptions,
): Promise<PlanData>;

export function codex(
  ctx: StageContext,
  input: ExecutorInput,
  options?: CodexOptions,
): Promise<ExecutionResultData>;

export function codex(
  ctx: StageContext,
  input: ReviewerInput,
  options?: CodexOptions,
): Promise<ReviewData>;

export async function codex(
  ctx: StageContext,
  input: PlannerInput | ExecutorInput | ReviewerInput,
  options: CodexOptions = {},
): Promise<PlanData | ExecutionResultData | ReviewData> {
  switch (ctx.stage) {
    case "planner": {
      const plannerInput = SPECS.planner.input.parse(input);
      return SPECS.planner.output.parse(
        await invoke(
          ctx,
          ctx.paths.repo,
          plannerPrompt(ctx, plannerInput),
          plannerOutputSchema,
          options,
        ),
      );
    }
    case "executor": {
      const executorInput = SPECS.executor.input.parse(input);
      return SPECS.executor.output.parse(
        await invoke(
          ctx,
          ctx.paths.workspace,
          executorPrompt(ctx, executorInput),
          executorOutputSchema,
          options,
        ),
      );
    }
    case "reviewer": {
      const reviewerInput = SPECS.reviewer.input.parse(input);
      return SPECS.reviewer.output.parse(
        await invoke(
          ctx,
          ctx.paths.workspace,
          reviewerPrompt(ctx, reviewerInput),
          reviewerOutputSchema,
          options,
        ),
      );
    }
  }
}

async function invoke(
  ctx: StageContext,
  cwd: string | undefined,
  prompt: string,
  outputSchema: JsonSchema,
  options: CodexOptions,
): Promise<unknown> {
  const { lastMessage } = await runCodex({
    prompt,
    cwd: cwd ?? process.cwd(),
    outputSchema,
    log: (event) => ctx.log(event),
    model: options.model,
  });

  return parseEnvelope(lastMessage);
}

// The final message should be bare JSON (the prompt demands it and
// --output-schema enforces it), but strip markdown fences defensively.
function parseEnvelope(message: string): unknown {
  const fenced = /^```(?:json)?\s*([\s\S]*?)\s*```$/.exec(message.trim());
  const body = fenced?.[1] ?? message.trim();
  try {
    return JSON.parse(body);
  } catch {
    throw new Error(`codex final message is not valid JSON (${body.length} characters)`);
  }
}
