import { z } from "zod";
import { plannerDefinitionSchema } from "./planner.ts";
import { executorDefinitionSchema } from "./executor.ts";
import { reviewerDefinitionSchema } from "./reviewer.ts";
import type { StageContext } from "../context.ts";
import type { ExecutionResultData, PlanData, ReviewData } from "../types.ts";
import { SPECS } from "../../runtime/mounts/io.ts";

export const definitionSchema = z.discriminatedUnion("stage", [
  plannerDefinitionSchema,
  executorDefinitionSchema,
  reviewerDefinitionSchema,
]);

export async function runAgent(
  agents: z.infer<typeof definitionSchema>,
  ctx: StageContext,
  raw: unknown,
): Promise<PlanData | ExecutionResultData | ReviewData> {
  switch (agents.stage) {
    case "planner": {
      const input = SPECS.planner.input.parse(raw);
      return SPECS.planner.output.parse(await agents.run(ctx, input));
    }
    case "executor": {
      const input = SPECS.executor.input.parse(raw);
      return SPECS.executor.output.parse(await agents.run(ctx, input));
    }
    case "reviewer": {
      const input = SPECS.reviewer.input.parse(raw);
      return SPECS.reviewer.output.parse(await agents.run(ctx, input));
    }
  }
}
