import type { StageContext } from "../../context.ts";

// Context shared by every stage prompt. Backend adapters can reuse this while
// keeping the role-specific instructions in their own prompt modules.
export function sharedPrompt(ctx: StageContext): string {
  const toolchains =
    process.env.PATCHDOCK_TOOLCHAIN_SUMMARY ??
    "The runtime did not publish a toolchain inventory. Inspect available commands before relying on them.";
  const budget = ctx.tokenBudget
    ? `- Advisory token budget: about ${ctx.tokenBudget} tokens. Prefer being concise over being exhaustive.\n`
    : "";

  return `Runtime environment:
- Available toolchains: ${toolchains}.
- Inspect repository manifests and lockfiles before choosing commands.
- Prefer repository-provided scripts and the package manager selected by its lockfile.
- Do not install system packages at runtime. If a required tool is unavailable, report its exact name instead of claiming the related verification passed.
${budget}`;
}
