// TypeScript mirror of internal/contracts (Go). Field names match the wire
// format (the Go structs' json tags) exactly — snake_case — so a value that
// validates here validates there.

// IMPORTANT!!!! KEEP IT IN SYNC WITH GO

import { z } from "zod";

export const TokenUsageSchema = z.object({
  input: z.number().int().min(0),
  output: z.number().int().min(0),
});
export type TokenUsage = z.infer<typeof TokenUsageSchema>;

export const ZERO_TOKENS: TokenUsage = { input: 0, output: 0 };

// Go time.Time marshals as RFC3339; accept any UTC-or-offset timestamp.
const timestamp = z.string().datetime({ offset: true });

export const StepSchema = z.object({
  id: z.string().min(1),
  description: z.string().min(1),
  // No omitempty on the Go side — always present, may be empty.
  rationale: z.string().default(""),
  files_to_modify: z.array(z.string()).optional(),
});
export type Step = z.infer<typeof StepSchema>;

export const PlanSchema = z.object({
  id: z.string().min(1),
  task_id: z.string().min(1),
  created_at: timestamp,
  approach: z.string().min(1),
  acceptance_criteria: z.array(z.string().min(1)).min(1),
  steps: z.array(StepSchema).min(1),
  context: z.array(z.string()).optional(),
  assumptions: z.array(z.string()).optional(),
  tokens_used: TokenUsageSchema,
});
export type Plan = z.infer<typeof PlanSchema>;

export const PlanDraftSchema = PlanSchema.omit({
  id: true,
  task_id: true,
  created_at: true,
  tokens_used: true,
}).extend({
  id: z.string().min(1).optional(),
  created_at: timestamp.optional(),
  tokens_used: TokenUsageSchema.optional(),
});
export type PlanDraft = z.infer<typeof PlanDraftSchema>;

export const ExecutionStatusSchema = z.enum([
  "success",
  "partial_success",
  "failed",
]);
export type ExecutionStatus = z.infer<typeof ExecutionStatusSchema>;

export const StepResultSchema = z.object({
  step_id: z.string().min(1),
  status: ExecutionStatusSchema,
  notes: z.string().optional(),
});
export type StepResult = z.infer<typeof StepResultSchema>;

export const ExecutionErrorSchema = z.object({
  step_id: z.string().optional(),
  message: z.string().min(1),
});
export type ExecutionError = z.infer<typeof ExecutionErrorSchema>;

export const ExecutionResultSchema = z.object({
  id: z.string().min(1),
  task_id: z.string().min(1),
  plan_id: z.string().min(1),
  status: ExecutionStatusSchema,
  // Patch is extracted by the orchestrator (git diff in the workspace);
  // agents normally leave it unset.
  patch: z.string().optional(),
  step_results: z.array(StepResultSchema),
  errors: z.array(ExecutionErrorSchema).optional(),
  tokens_used: TokenUsageSchema,
});
export type ExecutionResult = z.infer<typeof ExecutionResultSchema>;

export const ExecutionResultDraftSchema = ExecutionResultSchema.omit({
  id: true,
  task_id: true,
  plan_id: true,
  tokens_used: true,
}).extend({
  id: z.string().min(1).optional(),
  step_results: z.array(StepResultSchema).default([]),
  tokens_used: TokenUsageSchema.optional(),
});
export type ExecutionResultDraft = z.infer<typeof ExecutionResultDraftSchema>;

export const ReviewDecisionSchema = z.enum(["accept", "reject"]);
export type ReviewDecision = z.infer<typeof ReviewDecisionSchema>;

export const IssueSeveritySchema = z.enum(["blocker", "major", "minor"]);
export type IssueSeverity = z.infer<typeof IssueSeveritySchema>;

export const ReviewIssueSchema = z.object({
  severity: IssueSeveritySchema,
  message: z.string().min(1),
  step_id: z.string().optional(),
  file_path: z.string().optional(),
  line_range: z.string().optional(),
  suggestion: z.string().optional(),
});
export type ReviewIssue = z.infer<typeof ReviewIssueSchema>;

// A reject must carry actionable issues (documented invariant on the Go
// side; enforced here so it fails inside the container with a readable
// error instead of at the orchestrator boundary).
const rejectNeedsIssues = (
  v: { decision: ReviewDecision; issues?: unknown[] },
  ctx: z.RefinementCtx,
) => {
  if (v.decision === "reject" && (v.issues?.length ?? 0) === 0) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      path: ["issues"],
      message: "decision is 'reject' but no issues were provided",
    });
  }
};

export const ReviewFeedbackSchema = z
  .object({
    id: z.string().min(1),
    task_id: z.string().min(1),
    execution_id: z.string().min(1),
    decision: ReviewDecisionSchema,
    issues: z.array(ReviewIssueSchema).optional(),
    summary: z.string().min(1),
    tokens_used: TokenUsageSchema,
  })
  .superRefine(rejectNeedsIssues);
export type ReviewFeedback = z.infer<typeof ReviewFeedbackSchema>;

export const ReviewFeedbackDraftSchema = z
  .object({
    id: z.string().min(1).optional(),
    decision: ReviewDecisionSchema,
    issues: z.array(ReviewIssueSchema).optional(),
    summary: z.string().min(1),
    tokens_used: TokenUsageSchema.optional(),
  })
  .superRefine(rejectNeedsIssues);
export type ReviewFeedbackDraft = z.infer<typeof ReviewFeedbackDraftSchema>;

export const TaskSchema = z.object({
  id: z.string().min(1),
  title: z.string().min(1),
  description: z.string().default(""),
  source: z.enum(["prompt", "github_issue"]),
});
export type Task = z.infer<typeof TaskSchema>;

export const CheckResultSchema = z.object({
  name: z.string().min(1),
  command: z.string().min(1),
  exit_code: z.number().int(),
  output: z.string().default(""),
  duration_ms: z.number().int().min(0).optional(),
});
export type CheckResult = z.infer<typeof CheckResultSchema>;

export const PlannerInputSchema = z.object({
  task: TaskSchema,
});
export type PlannerInput = z.infer<typeof PlannerInputSchema>;

export const ExecutorInputSchema = z.object({
  task: TaskSchema,
  plan: PlanSchema,
  // Present only on retry attempts after a reviewer reject.
  feedback: ReviewFeedbackSchema.optional(),
});
export type ExecutorInput = z.infer<typeof ExecutorInputSchema>;

export const ReviewerInputSchema = z.object({
  task: TaskSchema,
  plan: PlanSchema,
  execution: ExecutionResultSchema,
  checks: z.array(CheckResultSchema).optional(),
});
export type ReviewerInput = z.infer<typeof ReviewerInputSchema>;
