FROM node:22-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g tsx@4
RUN npm install -g @openai/codex@0.142.4
WORKDIR /opt/patchdock/sdk

COPY package.json pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile --prod

COPY src ./src
COPY runtime ./runtime

RUN mkdir -p /node_modules/@patchdock \
    && ln -s /opt/patchdock/sdk /node_modules/@patchdock/sdk

ENTRYPOINT ["tsx", "/opt/patchdock/sdk/runtime/main.ts"]
