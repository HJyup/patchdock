import { z } from "zod";
import {
  executionResultDataSchema,
  planDataSchema,
  reviewDataSchema,
} from "../../types.ts";

export type JsonSchema = Record<string, unknown>;

// Codex strict output schemas require every property to be listed in `required`
function toCodexOutputSchema(schema: z.ZodType): JsonSchema {
  const generated = z.toJSONSchema(schema);
  delete generated.$schema;
  const properties = generated.properties ?? {};

  return {
    ...generated,
    required: Object.keys(properties),
  };
}

export const plannerOutputSchema = toCodexOutputSchema(planDataSchema);
export const executorOutputSchema = toCodexOutputSchema(executionResultDataSchema);
export const reviewerOutputSchema = toCodexOutputSchema(reviewDataSchema);
