import { z } from "zod";

// Artifact IDs and timestamps are not part of the sdk
// constructors in Go (NewTask/NewPlan/...) own them.

const taskSchema = z.object({
  id: z.string().min(1, "task id is required"),
  title: z.string().optional(),
  description: z.string().min(1, "description is required"),
  labels: z.array(z.string()).optional(),
});

export const plannerInputSchema = z.object({
  task: taskSchema,
});
export type PlannerInput = z.infer<typeof plannerInputSchema>;

const stepSchema = z.object({
  id: z.string().min(1, "step id is required"),
  description: z.string().min(1, "step description is required"),
  rationale: z.string().optional(),
  files_to_modify: z.array(z.string()).optional(),
});

export const planSchema = z.object({
  approach: z.string().min(1, "approach is required"),
  acceptance_criteria: z
    .array(z.string().min(1))
    .min(1, "at least one acceptance criterion is required"),
  steps: z.array(stepSchema).min(1, "at least one step is required"),
  context: z.array(z.string()).optional(),
  assumptions: z.array(z.string()).optional(),
});
export type Plan = z.infer<typeof planSchema>;
