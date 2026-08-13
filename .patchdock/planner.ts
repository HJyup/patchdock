import { codex, definePlanner } from "@patchdock/sdk";

export default definePlanner({
  async run(ctx, input) {
    return codex(ctx, input);
  },
});
