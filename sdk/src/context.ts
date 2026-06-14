import { z } from "zod";

export const StageSchema = z.enum(["planner", "executor", "reviewer"]);
export type Stage = z.infer<typeof StageSchema>;

// Mount paths handed to the agent:
//   Planner  - repo
//   Executor - workspace
//   Reviewer - workspace (read only)
interface MountPaths {
  repo?: string;
  workspace?: string;
}

interface StageContextData {
  stage: Stage;
  taskId: string;
  paths: MountPaths;
}

export interface StageContext extends StageContextData {
  log: (msg: string) => void;
}
