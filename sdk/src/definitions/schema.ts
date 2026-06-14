import { z } from "zod";
import { plannerDefinitionSchema } from "./planner.ts";
import { executorDefinitionSchema } from "./executor.ts";
import { reviewerDefinitionSchema } from "./reviewer.ts";

export const definitionSchema = z.discriminatedUnion("stage", [
  plannerDefinitionSchema,
  executorDefinitionSchema,
  reviewerDefinitionSchema,
]);
