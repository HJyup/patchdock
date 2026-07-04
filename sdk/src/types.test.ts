import { describe, expect, test } from "vitest";
import {
  plannerInputSchema,
  executorInputSchema,
  reviewerInputSchema,
  planDataSchema,
  reviewDataSchema,
} from "./types.ts";
import { fullPlan, fullExecutionResult, fullReview, task } from "./testing/models.ts";

describe("plannerInputSchema", () => {
  test("accepts a task without title and labels (Go omits empty fields)", () => {
    const result = plannerInputSchema.safeParse({
      task: { id: "task-1", description: "do the thing" },
    });
    expect(result.success).toBe(true);
  });

  test("rejects a task without description", () => {
    const result = plannerInputSchema.safeParse({ task: { id: "task-1" } });
    expect(result.success).toBe(false);
  });
});

describe("executorInputSchema", () => {
  test("accepts an empty reviews array (first attempt)", () => {
    const result = executorInputSchema.safeParse({ plan: fullPlan(), reviews: [] });
    expect(result.success).toBe(true);
  });

  test("rejects null reviews (Go must marshal [], not nil)", () => {
    const result = executorInputSchema.safeParse({ plan: fullPlan(), reviews: null });
    expect(result.success).toBe(false);
  });

  test("rejects the pre-rename singular 'review' key", () => {
    const result = executorInputSchema.safeParse({ plan: fullPlan(), review: [] });
    expect(result.success).toBe(false);
  });
});

describe("reviewerInputSchema", () => {
  test("accepts full inputs from the host", () => {
    const result = reviewerInputSchema.safeParse({
      plan: fullPlan(),
      execution_results: [fullExecutionResult()],
      previous_reviews: [fullReview()],
    });
    expect(result.success).toBe(true);
  });

  test("accepts an execution result without a patch (failed before modifying)", () => {
    const { patch, ...withoutPatch } = fullExecutionResult();
    void patch;
    const result = reviewerInputSchema.safeParse({
      plan: fullPlan(),
      execution_results: [withoutPatch],
      previous_reviews: [],
    });
    expect(result.success).toBe(true);
  });
});

describe("planDataSchema (planner output)", () => {
  test("accepts a minimal plan: no rationale, files, context, or assumptions", () => {
    const result = planDataSchema.safeParse({
      approach: "small focused change",
      acceptance_criteria: ["it works"],
      steps: [{ id: "step-1", description: "do it" }],
    });
    expect(result.success).toBe(true);
  });

  test("rejects empty acceptance_criteria and empty steps", () => {
    const base = {
      approach: "a",
      acceptance_criteria: ["done"],
      steps: [{ id: "s", description: "d" }],
    };
    expect(planDataSchema.safeParse({ ...base, acceptance_criteria: [] }).success).toBe(
      false,
    );
    expect(planDataSchema.safeParse({ ...base, steps: [] }).success).toBe(false);
  });

  test("does not accept id/task_id — identity belongs to the host", () => {
    const result = planDataSchema.safeParse({
      approach: "a",
      acceptance_criteria: ["done"],
      steps: [{ id: "s", description: "d" }],
      id: "plan-forged",
      task_id: "task-forged",
    });
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data).not.toHaveProperty("id");
      expect(result.data).not.toHaveProperty("task_id");
    }
  });
});

describe("reviewDataSchema (reviewer output)", () => {
  test("accepts an accept decision without issues", () => {
    const result = reviewDataSchema.safeParse({
      decision: "accept",
      summary: "looks good",
    });
    expect(result.success).toBe(true);
  });

  test("rejects decisions outside the enum", () => {
    const result = reviewDataSchema.safeParse({ decision: "maybe", summary: "hmm" });
    expect(result.success).toBe(false);
  });
});

describe("task fixture stays a valid contract", () => {
  test("full task with optional fields parses", () => {
    const result = plannerInputSchema.safeParse({ task: task() });
    expect(result.success).toBe(true);
  });
});
