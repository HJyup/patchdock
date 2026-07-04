import type { StageContext } from "../context.ts";

export function task() {
  return {
    id: "task-1",
    title: "add farewell",
    description: "add a farewell function with a test",
    labels: ["demo"],
  };
}

export function fullPlan() {
  return {
    id: "plan-1",
    task_id: "task-1",
    created_at: "2026-07-02T21:27:04.949582Z",
    approach: "small focused change",
    acceptance_criteria: ["farewell exists and is tested"],
    steps: [{ id: "step-1", description: "implement farewell" }],
  };
}

export function fullExecutionResult() {
  return {
    id: "exec-1",
    task_id: "task-1",
    plan_id: "plan-1",
    status: "success",
    patch: "diff --git a/src/greet.ts b/src/greet.ts\n",
    step_results: [{ step_id: "step-1", status: "success" }],
  };
}

export function fullReview() {
  return {
    id: "review-1",
    task_id: "task-1",
    execution_id: "exec-1",
    decision: "reject",
    summary: "missing test",
    issues: [{ severity: "major", message: "no test was added" }],
  };
}

export function stageContext(overrides: Partial<StageContext> = {}): StageContext {
  return {
    stage: "planner",
    taskId: "task-1",
    paths: { repo: "/repo", workspace: "/workspace" },
    tokenBudget: null,
    attempt: 1,
    maxAttempts: 1,
    log: () => {},
    ...overrides,
  };
}
