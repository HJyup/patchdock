import { z } from "zod";

export const StageSchema = z.enum(["planner", "executor", "reviewer"]);
export type Stage = z.infer<typeof StageSchema>;

export interface StageLogEvent {
  source: string;
  event: string;
  level?: "debug" | "info" | "warn" | "error";
  message?: string;
  [field: string]: unknown;
}

// Mount paths handed to the agent:
//   Planner  - repo
//   Executor - workspace
//   Reviewer - workspace (read only)
interface MountPaths {
  repo?: string;
  workspace?: string;
}

type Nullable<T> = T | null;

interface StageContextData {
  stage: Stage;
  taskId: string;
  paths: MountPaths;
  tokenBudget: Nullable<number>;
  attempt: number;
  maxAttempts: number;
}

export interface StageContext extends StageContextData {
  log: (entry: string | StageLogEvent) => void;
}

export function parseTokenBudget(budget: string | undefined): Nullable<number> {
  return z.coerce.number().int().positive().nullable().catch(null).parse(budget);
}

export function parseAttempt(value: string | undefined): number {
  return z.coerce.number().int().positive().catch(1).parse(value);
}
