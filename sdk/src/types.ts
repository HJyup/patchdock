import { z } from "zod";

const executionStatusSchema = z.enum(["success", "partial_success", "failed"]);
const reviewDecisionSchema = z.enum(["accept", "reject"]);

const taskSchema = z.object({
  id: z.string().min(1),
  title: z.string().optional(),
  description: z.string().min(1),
  labels: z.array(z.string()).optional(),
});

// Agent-authored payloads are deliberately loose: a short summary the
// runtime displays, plus one markdown field the next stage reads. Structure
// inside the prose (steps, issue lists, severities) is a prompt convention —
// only fields the runtime branches on or displays are typed strictly.

export const planDataSchema = z.object({
  summary: z.string().min(1),
  body: z.string().min(1),
});

const planSchema = planDataSchema.extend({
  id: z.string().min(1),
  task_id: z.string().min(1),
  created_at: z.string(),
});

export const executionResultDataSchema = z.object({
  status: executionStatusSchema,
  notes: z.string().optional(),
});

const executionResultSchema = executionResultDataSchema.extend({
  id: z.string().min(1),
  task_id: z.string().min(1),
  plan_id: z.string().min(1),
  patch: z.string().optional(),
});

const reviewFields = z.object({
  decision: reviewDecisionSchema,
  summary: z.string().min(1),
  feedback: z.string().optional(),
});

// A reject must carry feedback so a retry never flies blind.
export const reviewDataSchema = reviewFields.refine(
  (r) => r.decision !== "reject" || (r.feedback ?? "").length > 0,
  { message: "feedback is required when decision is reject", path: ["feedback"] },
);

// Host-stitched reviews were already validated on the host, so inputs reuse
// the plain field shape without re-running the cross-field rule.
const reviewSchema = reviewFields.extend({
  id: z.string().min(1),
  task_id: z.string().min(1),
  execution_id: z.string().min(1),
});

export const plannerInputSchema = z.object({
  task: taskSchema,
});

export const executorInputSchema = z.object({
  plan: planSchema,
  reviews: z.array(reviewSchema),
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
