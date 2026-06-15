import { z } from "zod";

const executionStatusSchema = z.enum(["success", "partial_success", "failed"]);
const reviewDecisionSchema = z.enum(["accept", "reject"]);
const issueSeveritySchema = z.enum(["blocker", "major", "minor"]);

const taskSchema = z.object({
  id: z.string().min(1),
  title: z.string().optional(),
  description: z.string().min(1),
  labels: z.array(z.string()).optional(),
});

const stepSchema = z.object({
  id: z.string().min(1),
  description: z.string().min(1),
  rationale: z.string().optional(),
  files_to_modify: z.array(z.string()).optional(),
});

export const planDataSchema = z.object({
  approach: z.string().min(1),
  acceptance_criteria: z.array(z.string().min(1)).min(1),
  steps: z.array(stepSchema).min(1),
  context: z.array(z.string()).optional(),
  assumptions: z.array(z.string()).optional(),
});

const planSchema = planDataSchema.extend({
  id: z.string().min(1),
  task_id: z.string().min(1),
  created_at: z.string(),
});

const stepResultSchema = z.object({
  step_id: z.string().min(1),
  status: executionStatusSchema,
  notes: z.string().optional(),
});

const executionErrorSchema = z.object({
  step_id: z.string().optional(),
  message: z.string().min(1),
});

export const executionResultDataSchema = z.object({
  status: executionStatusSchema,
  step_results: z.array(stepResultSchema),
  errors: z.array(executionErrorSchema).optional(),
});

const executionResultSchema = executionResultDataSchema.extend({
  id: z.string().min(1),
  task_id: z.string().min(1),
  plan_id: z.string().min(1),
  patch: z.string().optional(),
});

const reviewIssueSchema = z.object({
  severity: issueSeveritySchema,
  message: z.string().min(1),
  step_id: z.string().optional(),
  file_path: z.string().optional(),
  line_range: z.string().optional(),
  suggestion: z.string().optional(),
});

export const reviewDataSchema = z.object({
  decision: reviewDecisionSchema,
  issues: z.array(reviewIssueSchema).optional(),
  summary: z.string().min(1),
});

const reviewSchema = reviewDataSchema.extend({
  id: z.string().min(1),
  task_id: z.string().min(1),
  execution_id: z.string().min(1),
});

export const plannerInputSchema = z.object({
  task: taskSchema,
});

export const executorInputSchema = z.object({
  plan: planSchema,
  review: z.array(reviewSchema),
});

export const reviewerInputSchema = z.object({
  plan: planSchema,
  execution_results: z.array(executionResultSchema),
  previous_reviews: z.array(reviewSchema),
});

export type PlanData = z.infer<typeof planDataSchema>;
export type ExecutionResultData = z.infer<typeof executionResultDataSchema>;
export type ReviewData = z.infer<typeof reviewDataSchema>;

export type PlannerInput = z.infer<typeof plannerInputSchema>;
export type ExecutorInput = z.infer<typeof executorInputSchema>;
export type ReviewerInput = z.infer<typeof reviewerInputSchema>;
