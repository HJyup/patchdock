// StageContext is the capability bag handed to every agent function.

export type StageName = "planner" | "executor" | "reviewer";

export interface StageLogger {
  info(msg: string): void;
  warn(msg: string): void;
  error(msg: string): void;
}

export interface StagePaths {
  /** Contract exchange dir (input.json / output.json). */
  io: string;
  /** Read-only mount of the target repo (planner, reviewer). */
  repo: string;
  /** Read-write sandbox copy (executor only). */
  workspace: string;
}

export interface StageContext {
  taskId: string;
  stage: StageName;
  log: StageLogger;
  paths: StagePaths;
}

export function makeLogger(stage: StageName): StageLogger {
  // Plain stdout: container logs are streamed and demuxed by the
  // orchestrator; they are never parsed for data.
  const emit = (level: string, msg: string) =>
    console.log(`[${stage}] ${level}: ${msg}`);
  return {
    info: (msg) => emit("info", msg),
    warn: (msg) => emit("warn", msg),
    error: (msg) => emit("error", msg),
  };
}
