import {
  plannerInputSchema,
  executorInputSchema,
  reviewerInputSchema,
  planDataSchema,
  executionResultDataSchema,
  reviewDataSchema,
} from "../../src/types.ts";
import type { Stage } from "../../src/context.ts";
import { z } from "zod";

// Constants inside the IO mount
export const IO_PATH = "/io";
export const INPUT_FILE = "input.json";
export const OUTPUT_FILE = "output.json";

// What crosses /io per stage.
export const SPECS = {
  planner: { input: plannerInputSchema, output: planDataSchema },
  executor: { input: executorInputSchema, output: executionResultDataSchema },
  reviewer: { input: reviewerInputSchema, output: reviewDataSchema },
} satisfies Record<Stage, { input: z.ZodTypeAny; output: z.ZodTypeAny }>;
