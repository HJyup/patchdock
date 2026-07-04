import type { StageContext } from "../context.ts";
import type { ExecutionResultData, ExecutorInput } from "../types.ts";
import { z } from "zod";

export const executorDefinitionSchema = z.object({
  stage: z.literal("executor"),
  run: z.custom<ExecutorConfig["run"]>((value) => typeof value === "function", {
    message: "run must be a function",
  }),
});

interface ExecutorConfig {
  run: (ctx: StageContext, input: ExecutorInput) => Promise<ExecutionResultData>;
}

interface Executor extends ExecutorConfig {
  stage: "executor";
}

export function defineExecutor(config: ExecutorConfig): Executor {
  return {
    stage: "executor",
    run: config.run,
  };
}
