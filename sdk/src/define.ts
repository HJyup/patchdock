// The define* wrappers — the only patchdock API a user's agent file must
// touch. Each tags the user's function with its stage so the runtime can
// verify the right file was loaded, and carries the input/output schemas
// the harness validates against.

import { z } from "zod";
import type { StageContext, StageName } from "./context.ts";
import {
  ExecutorInputSchema,
  ExecutionResultDraftSchema,
  PlannerInputSchema,
  PlanDraftSchema,
  ReviewerInputSchema,
  ReviewFeedbackDraftSchema,
  type ExecutorInput,
  type ExecutionResultDraft,
  type PlannerInput,
  type PlanDraft,
  type ReviewerInput,
  type ReviewFeedbackDraft,
} from "./contracts.ts";

export interface StageDefinition<In, Out> {
  readonly __patchdock: true;
  readonly stage: StageName;
  readonly inputSchema: z.ZodType<In>;
  readonly outputSchema: z.ZodType<Out>;
  readonly run: (input: In, ctx: StageContext) => Out | Promise<Out>;
}

export type AnyStageDefinition = StageDefinition<unknown, unknown>;

function define<In, Out>(
  stage: StageName,
  inputSchema: z.ZodType<In>,
  outputSchema: z.ZodType<Out>,
  run: (input: In, ctx: StageContext) => Out | Promise<Out>,
): StageDefinition<In, Out> {
  return { __patchdock: true, stage, inputSchema, outputSchema, run };
}

export function definePlanner(
  run: (
    input: PlannerInput,
    ctx: StageContext,
  ) => PlanDraft | Promise<PlanDraft>,
): StageDefinition<PlannerInput, PlanDraft> {
  return define(
    "planner",
    PlannerInputSchema as z.ZodType<PlannerInput>,
    PlanDraftSchema as z.ZodType<PlanDraft>,
    run,
  );
}

export function defineExecutor(
  run: (
    input: ExecutorInput,
    ctx: StageContext,
  ) => ExecutionResultDraft | Promise<ExecutionResultDraft>,
): StageDefinition<ExecutorInput, ExecutionResultDraft> {
  return define(
    "executor",
    ExecutorInputSchema as z.ZodType<ExecutorInput>,
    ExecutionResultDraftSchema as z.ZodType<ExecutionResultDraft>,
    run,
  );
}

export function defineReviewer(
  run: (
    input: ReviewerInput,
    ctx: StageContext,
  ) => ReviewFeedbackDraft | Promise<ReviewFeedbackDraft>,
): StageDefinition<ReviewerInput, ReviewFeedbackDraft> {
  return define(
    "reviewer",
    ReviewerInputSchema as z.ZodType<ReviewerInput>,
    ReviewFeedbackDraftSchema as z.ZodType<ReviewFeedbackDraft>,
    run,
  );
}

export function isStageDefinition(v: unknown): v is AnyStageDefinition {
  return (
    typeof v === "object" &&
    v !== null &&
    (v as Record<string, unknown>).__patchdock === true &&
    typeof (v as Record<string, unknown>).run === "function"
  );
}
