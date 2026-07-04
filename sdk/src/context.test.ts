import { describe, expect, test } from "vitest";
import { parseAttempt, parseTokenBudget } from "./context.ts";

describe("parseTokenBudget", () => {
  test("absent env var means unlimited", () => {
    expect(parseTokenBudget(undefined)).toBeNull();
  });

  test("empty string means unlimited", () => {
    expect(parseTokenBudget("")).toBeNull();
  });

  test("garbage means unlimited, not NaN", () => {
    expect(parseTokenBudget("abc")).toBeNull();
    expect(parseTokenBudget("abc")).not.toBeNaN();
  });

  test("zero means unlimited (host only sends budgets > 0)", () => {
    expect(parseTokenBudget("0")).toBeNull();
  });

  test("negative and fractional values are rejected", () => {
    expect(parseTokenBudget("-5")).toBeNull();
    expect(parseTokenBudget("3.5")).toBeNull();
  });
});

describe("parseAttempt", () => {
  test("absent env var defaults to first attempt", () => {
    expect(parseAttempt(undefined)).toBe(1);
  });

  test("parses a positive integer", () => {
    expect(parseAttempt("3")).toBe(3);
  });

  test("garbage and non-positive values default to 1", () => {
    expect(parseAttempt("abc")).toBe(1);
    expect(parseAttempt("0")).toBe(1);
    expect(parseAttempt("-2")).toBe(1);
  });
});
