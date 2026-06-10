FROM node:22-slim

WORKDIR /opt/patchdock

COPY package.json ./
RUN npm install --omit=dev

COPY tsconfig.json ./
COPY src ./src
COPY runtime ./runtime

# Let user agent files at /agents resolve `import "@patchdock/sdk"`:
# node walks parent dirs for node_modules, all the way to /.
RUN mkdir -p /node_modules/@patchdock \
    && ln -s /opt/patchdock /node_modules/@patchdock/sdk

ENV PATCHDOCK_IO=/io \
    PATCHDOCK_AGENTS=/agents \
    PATCHDOCK_REPO=/repo \
    PATCHDOCK_WORKSPACE=/workspace

ENTRYPOINT ["/opt/patchdock/node_modules/.bin/tsx", "/opt/patchdock/runtime/main.ts"]
