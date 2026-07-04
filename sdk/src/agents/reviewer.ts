import type { StageContext } from "../context.ts";
import type { ReviewData, ReviewerInput } from "../types.ts";
import { z } from "zod";

export const reviewerDefinitionSchema = z.object({
  stage: z.literal("reviewer"),
  run: z.custom<ReviewerConfig["run"]>((value) => typeof value === "function", {
    message: "run must be a function",
  }),
});

interface ReviewerConfig {
  run: (ctx: StageContext, input: ReviewerInput) => Promise<ReviewData>;
}

interface Reviewer extends ReviewerConfig {
  stage: "reviewer";
}

export function defineReviewer(config: ReviewerConfig): Reviewer {
  return {
    stage: "reviewer",
    run: config.run,
  };
}
