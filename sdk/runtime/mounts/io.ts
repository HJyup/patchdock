import { plannerInputSchema, planSchema } from "../../src/types.ts";
import type { Stage } from "../../src/context.ts";
import { z } from "zod";

// Constants inside the IO mount
export const IO_PATH = "/io";
export const INPUT_FILE = "input.json";
export const OUTPUT_FILE = "output.json";

// Specification of what we expect as input/output for IO mount
export const SPECS = {
  planner: { input: plannerInputSchema, output: planSchema },
  executor: { input: plannerInputSchema, output: planSchema },
  reviewer: { input: plannerInputSchema, output: planSchema },
} satisfies Record<Stage, { input: z.ZodTypeAny; output: z.ZodTypeAny }>;
