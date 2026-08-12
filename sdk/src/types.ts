import { z } from "zod";

const executionStatusSchema = z.enum(["success", "partial_success", "failed"]);
const reviewDecisionSchema = z.enum(["accept", "reject"]);

const taskSchema = z.object({
  title: z.string().optional(),
  description: z.string().min(1),
  labels: z.array(z.string()).optional(),
});

export const planDataSchema = z.object({
  summary: z.string().min(1),
  body: z.string().min(1),
});

const planSchema = planDataSchema.extend({
  created_at: z.string(),
});

export const executionResultDataSchema = z.object({
  status: executionStatusSchema,
  notes: z.string().optional(),
});

const reviewFields = z.object({
  decision: reviewDecisionSchema,
  summary: z.string().min(1),
  feedback: z.string().optional(),
});

export const reviewDataSchema = reviewFields.refine(
  (r) => r.decision !== "reject" || (r.feedback ?? "").length > 0,
  { message: "feedback is required when decision is reject", path: ["feedback"] },
);

export const plannerInputSchema = z.object({
  task: taskSchema,
});

// History arrays are ordered oldest attempt first: index 0 is attempt 1, and
// the last entry is the attempt that just ran.
export const executorInputSchema = z.object({
  plan: planSchema,
  reviews: z.array(reviewFields),
});

export const reviewerInputSchema = z.object({
  plan: planSchema,
  patch: z.string(),
  execution_results: z.array(executionResultDataSchema),
  previous_reviews: z.array(reviewFields),
});

export type PlanData = z.infer<typeof planDataSchema>;
export type ExecutionResultData = z.infer<typeof executionResultDataSchema>;
export type ReviewData = z.infer<typeof reviewDataSchema>;

export type PlannerInput = z.infer<typeof plannerInputSchema>;
export type ExecutorInput = z.infer<typeof executorInputSchema>;
export type ReviewerInput = z.infer<typeof reviewerInputSchema>;
