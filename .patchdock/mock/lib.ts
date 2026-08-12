import type { StageContext, StageLogEvent } from "@patchdock/sdk";

export function jitter(minSeconds: number, maxSeconds: number): number {
  return (minSeconds + Math.random() * (maxSeconds - minSeconds)) * 1000;
}

export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// Spreads steps evenly across totalMs so the live activity feed has something
// to render while the stage pretends to work.
export async function pace(
  ctx: StageContext,
  totalMs: number,
  steps: StageLogEvent[],
): Promise<void> {
  const slice = totalMs / steps.length;

  for (const step of steps) {
    ctx.log(step);
    await sleep(slice);
  }
}

export function firstLine(text: string): string {
  const line = text.split("\n").find((candidate) => candidate.trim() !== "");
  return line?.trim() ?? "untitled task";
}
