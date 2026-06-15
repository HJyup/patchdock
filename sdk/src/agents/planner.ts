import type { Stage, StageContext } from "../context.ts";
import type { PlanData, PlannerInput } from "../types.ts";
import { z } from "zod";

export const plannerDefinitionSchema = z.object({
  stage: z.literal("planner"),
  run: z.custom<PlannerConfig["run"]>((value) => typeof value === "function", {
    message: "run must be a function",
  }),
});

interface PlannerConfig {
  run: (ctx: StageContext, input: PlannerInput) => Promise<PlanData>;
}

interface Planner extends PlannerConfig {
  stage: Stage;
}

export function definePlanner(config: PlannerConfig): Planner {
  return {
    stage: "planner",
    run: config.run,
  };
}
