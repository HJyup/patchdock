import { describe, expect, test } from "vitest";
import { reviewerOutputSchema } from "./schemas.ts";

describe("reviewerOutputSchema", () => {
  test("requires every reviewer output property for Codex strict schemas", () => {
    expect(reviewerOutputSchema.required).toEqual(
      expect.arrayContaining(["decision", "summary", "feedback"]),
    );
    expect(reviewerOutputSchema.required).toHaveLength(3);
  });

  test("contains only reviewer output properties", () => {
    expect(reviewerOutputSchema.properties).toHaveProperty("decision");
    expect(reviewerOutputSchema.properties).toHaveProperty("summary");
    expect(reviewerOutputSchema.properties).toHaveProperty("feedback");
    expect(reviewerOutputSchema.properties).not.toHaveProperty("id");
    expect(reviewerOutputSchema.properties).not.toHaveProperty("task_id");
    expect(reviewerOutputSchema.properties).not.toHaveProperty("execution_id");
  });
});
