FROM node:22-slim

# tsx runs TypeScript directly (both the harness and the user's agent files)
RUN npm install -g tsx@4
WORKDIR /opt/patchdock/sdk

COPY package.json pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile --prod

COPY src ./src
COPY runtime ./runtime

RUN mkdir -p /node_modules/@patchdock \
    && ln -s /opt/patchdock/sdk /node_modules/@patchdock/sdk

ENTRYPOINT ["tsx", "/opt/patchdock/sdk/runtime/main.ts"]
