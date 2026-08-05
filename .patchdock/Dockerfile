# Patchdock agent runtime image. One container per pipeline stage.

FROM node:22-bookworm-slim

ARG TSX_VERSION=4.22.4
ARG CODEX_VERSION=0.142.4

# ── System packages ───────────────────────────────────────────────────────────
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        git \
        make \
        ripgrep \
    && rm -rf /var/lib/apt/lists/*

# ── Git configuration ─────────────────────────────────────────────────────────
# Mounts are owned by the host user, so git rejects them without safe.directory.
RUN git config --system --add safe.directory '*' \
    && git config --system core.pager cat

# ── Codex configuration ───────────────────────────────────────────────────────
# The scaffolded agents shell out to `codex exec`. Credentials are mounted by
# the host at run time.
RUN npm install -g "@openai/codex@${CODEX_VERSION}"

# ── SDK runtime ───────────────────────────────────────────────────────────────
# Agents are .ts files imported unbuilt, so tsx is required.
RUN npm install -g "tsx@${TSX_VERSION}"

# The SDK comes from this directory's node_modules (`pnpm install` in
# .patchdock), not from source. /node_modules is where agents mounted at /agents
# resolve a bare "@patchdock/sdk".
COPY node_modules /node_modules

# The host passes no command, so this line is the whole contract.
ENTRYPOINT ["tsx", "/node_modules/@patchdock/sdk/runtime/main.ts"]

# ── Stream hygiene ────────────────────────────────────────────────────────────
# No tty: stop git prompting, and keep ANSI out of the JSON logs. CI is
# deliberately not set — test suites read it as "this environment guarantees
# services", and a stage container guarantees none.
ENV GIT_TERMINAL_PROMPT=0 \
    NO_COLOR=1

# ══ PROJECT SCOPE — everything below this line is yours ═══════════════════════
# Patchdock is a Go CLI plus a TypeScript SDK, so agents need both toolchains.

# ── Go ────────────────────────────────────────────────────────────────────────
# Pinned to the `go` directive in go.mod. GOTOOLCHAIN=local makes a version
# mismatch fail loudly instead of silently downloading another toolchain.
COPY --from=golang:1.26.3-bookworm /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}" \
    GOTOOLCHAIN=local

# gcc and libc6-dev are here for one reason: `go test -race` needs cgo.
RUN apt-get update && apt-get install -y --no-install-recommends \
        gcc \
        libc6-dev \
    && rm -rf /var/lib/apt/lists/*

# ── TypeScript ────────────────────────────────────────────────────────────────
# pnpm drives sdk/: typecheck, lint and vitest. Pinned to the packageManager
# field in sdk/package.json.
RUN corepack enable && corepack prepare pnpm@10.33.0 --activate

# Injected verbatim into every agent prompt — keep it accurate.
ENV PATCHDOCK_TOOLCHAIN_SUMMARY="Debian Bookworm. Go 1.26.3 with gcc for -race, gofmt and go vet. Node.js 22 with pnpm 10.33.0 and tsx for TypeScript. Git, ripgrep (rg), make and curl. Repository dependencies are NOT prewarmed: run `go mod download` and, in sdk/, `pnpm install --frozen-lockfile` before building or testing. The docker tests in internal/docker need a daemon and cannot pass inside a stage container."
