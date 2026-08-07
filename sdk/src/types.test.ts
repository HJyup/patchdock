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
      patch: "diff --git a/src/greet.ts b/src/greet.ts\n",
      execution_results: [fullExecutionResult()],
      previous_reviews: [fullReview()],
    });
    expect(result.success).toBe(true);
  });

  test("accepts an empty patch (the executor changed nothing)", () => {
    const result = reviewerInputSchema.safeParse({
      plan: fullPlan(),
      patch: "",
      execution_results: [fullExecutionResult()],
      previous_reviews: [],
    });
    expect(result.success).toBe(true);
  });

  test("rejects a missing patch (Go always sends the field)", () => {
    const result = reviewerInputSchema.safeParse({
      plan: fullPlan(),
      execution_results: [fullExecutionResult()],
      previous_reviews: [],
    });
    expect(result.success).toBe(false);
  });

  test("accepts an execution result without notes (failed early)", () => {
    const { notes, ...bare } = fullExecutionResult();
    void notes;
    const result = reviewerInputSchema.safeParse({
      plan: fullPlan(),
      patch: "",
      execution_results: [bare],
      previous_reviews: [],
    });
    expect(result.success).toBe(true);
  });

  test("rejects legacy singular and camelCase result keys", () => {
    const result = reviewerInputSchema.safeParse({
      plan: fullPlan(),
      patch: "",
      execution_result: [fullExecutionResult()],
      previousReviews: [fullReview()],
    });
    expect(result.success).toBe(false);
  });

  test("rejects null arrays from the host", () => {
    expect(
      reviewerInputSchema.safeParse({
        plan: fullPlan(),
        patch: "",
        execution_results: null,
        previous_reviews: [],
      }).success,
    ).toBe(false);
    expect(
      reviewerInputSchema.safeParse({
        plan: fullPlan(),
        patch: "",
        execution_results: [fullExecutionResult()],
        previous_reviews: null,
      }).success,
    ).toBe(false);
  });

  test.each(["id", "task_id", "execution_id"] as const)(
    "rejects a previous review missing %s",
    (field) => {
      const review: Record<string, unknown> = { ...fullReview() };
      delete review[field];

      const result = reviewerInputSchema.safeParse({
        plan: fullPlan(),
        patch: "",
        execution_results: [fullExecutionResult()],
        previous_reviews: [review],
      });

      expect(result.success).toBe(false);
    },
  );
});

describe("planDataSchema (planner output)", () => {
  test("accepts a minimal plan: summary plus markdown body", () => {
    const result = planDataSchema.safeParse({
      summary: "small focused change",
      body: "## Steps\n1. do it",
    });
    expect(result.success).toBe(true);
  });

  test("rejects an empty summary or body", () => {
    expect(planDataSchema.safeParse({ summary: "", body: "b" }).success).toBe(false);
    expect(planDataSchema.safeParse({ summary: "s", body: "" }).success).toBe(false);
  });

  test("does not accept id/task_id — identity belongs to the host", () => {
    const result = planDataSchema.safeParse({
      summary: "s",
      body: "b",
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
  test("accepts an accept decision without feedback", () => {
    const result = reviewDataSchema.safeParse({
      decision: "accept",
      summary: "looks good",
    });
    expect(result.success).toBe(true);
  });

  test("accepts an accept decision with optional feedback (nits)", () => {
    const result = reviewDataSchema.safeParse({
      decision: "accept",
      summary: "looks good",
      feedback: "minor: naming nit, fine to ship",
    });
    expect(result.success).toBe(true);
  });

  test("rejects a reject decision without feedback (retry must not fly blind)", () => {
    const result = reviewDataSchema.safeParse({
      decision: "reject",
      summary: "does not compile",
    });
    expect(result.success).toBe(false);
  });

  test("accepts a reject decision with feedback", () => {
    const result = reviewDataSchema.safeParse({
      decision: "reject",
      summary: "does not compile",
      feedback: "- **blocker** — src/greet.ts:12 missing closing brace",
    });
    expect(result.success).toBe(true);
  });

  test("rejects decisions outside the enum", () => {
    const result = reviewDataSchema.safeParse({ decision: "maybe", summary: "hmm" });
    expect(result.success).toBe(false);
  });

  test("rejects empty summaries for accept and reject decisions", () => {
    expect(reviewDataSchema.safeParse({ decision: "accept", summary: "" }).success).toBe(
      false,
    );
    expect(
      reviewDataSchema.safeParse({
        decision: "reject",
        summary: "",
        feedback: "explain the blocker",
      }).success,
    ).toBe(false);
  });

  test("rejects empty feedback for a reject decision", () => {
    const result = reviewDataSchema.safeParse({
      decision: "reject",
      summary: "does not compile",
      feedback: "",
    });
    expect(result.success).toBe(false);
  });

  test("does not accept host-owned identity fields", () => {
    const result = reviewDataSchema.safeParse({
      decision: "accept",
      summary: "looks good",
      feedback: "ship it",
      id: "review-forged",
      task_id: "task-forged",
      execution_id: "exec-forged",
    });
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data).not.toHaveProperty("id");
      expect(result.data).not.toHaveProperty("task_id");
      expect(result.data).not.toHaveProperty("execution_id");
    }
  });
});

describe("task fixture stays a valid contract", () => {
  test("full task with optional fields parses", () => {
    const result = plannerInputSchema.safeParse({ task: task() });
    expect(result.success).toBe(true);
  });
});
