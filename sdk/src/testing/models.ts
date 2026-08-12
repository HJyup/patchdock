import type { StageContext } from "../context.ts";

export function task() {
  return {
    title: "add farewell",
    description: "add a farewell function with a test",
    labels: ["demo"],
  };
}

export function fullPlan() {
  return {
    created_at: "2026-07-02T21:27:04.949582Z",
    summary: "small focused change",
    body: [
      "## Approach",
      "Add a farewell function next to greet.",
      "",
      "## Steps",
      "1. implement farewell",
      "",
      "## Acceptance criteria",
      "- farewell exists and is tested",
    ].join("\n"),
  };
}

export function fullExecutionResult() {
  return {
    status: "success",
    notes: "Implemented farewell in src/greet.ts.",
  };
}

export function fullReview() {
  return {
    decision: "reject",
    summary: "missing test",
    feedback: "- **major** — no test was added; cover farewell() with one test",
  };
}

export function stageContext(overrides: Partial<StageContext> = {}): StageContext {
  return {
    stage: "planner",
    runId: "run-1",
    paths: { repo: "/repo", workspace: "/workspace" },
    tokenBudget: null,
    attempt: 1,
    maxAttempts: 1,
    log: () => {},
    ...overrides,
  };
}
