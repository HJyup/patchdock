FROM node:22-bookworm-slim

ARG CODEX_VERSION=0.142.4
ARG PNPM_VERSION=10.33.0
ARG TSX_VERSION=4.22.4

# The agent image is deliberately useful without installing system packages at
# runtime. It covers TypeScript/JavaScript repositories and includes the native
# build prerequisites commonly required by Node dependencies.
RUN apt-get update && apt-get install -y --no-install-recommends \
        bash \
        build-essential \
        ca-certificates \
        curl \
        fd-find \
        file \
        findutils \
        git \
        jq \
        less \
        openssh-client \
        pkg-config \
        procps \
        python-is-python3 \
        python3 \
        ripgrep \
        tree \
        unzip \
        zip \
    && ln -s /usr/bin/fdfind /usr/local/bin/fd \
    && rm -rf /var/lib/apt/lists/*

ENV PATCHDOCK_TOOLCHAIN_SUMMARY="Debian Bookworm; Node.js 22 with npm and pnpm ${PNPM_VERSION}; TypeScript execution through tsx ${TSX_VERSION}; Python 3, make, GCC/G++, and pkg-config for native Node dependencies; Git, ripgrep (rg), fd, jq, curl, and common archive/process tools"

RUN corepack enable \
    && corepack prepare "pnpm@${PNPM_VERSION}" --activate \
    && npm install -g "tsx@${TSX_VERSION}" "@openai/codex@${CODEX_VERSION}" \
    && git config --system core.pager cat \
    && git config --system --add safe.directory /repo \
    && git config --system --add safe.directory /workspace
WORKDIR /opt/patchdock/sdk

COPY package.json pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile --prod

COPY src ./src
COPY runtime ./runtime

RUN mkdir -p /node_modules/@patchdock \
    && ln -s /opt/patchdock/sdk /node_modules/@patchdock/sdk

ENTRYPOINT ["tsx", "/opt/patchdock/sdk/runtime/main.ts"]
