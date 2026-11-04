# The website (Next.js), with the duel board package it is built with. Build
# from the repository root:
#
#   docker build -f apps/web/Dockerfile -t duel-masters-web .
#
# The site is built as a Next.js standalone server, which carries only the
# files it runs with, and runs as the image's non-root user. Point it at the
# Go server with GO_SERVER_URL when it starts (not when it is built).

FROM node:24-alpine AS build

# The pnpm version pinned in package.json.
RUN corepack enable

WORKDIR /src

# Dependencies first, so changing the code does not download them again.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/
COPY packages/duel-board/package.json packages/duel-board/
RUN --mount=type=cache,id=pnpm-store,target=/pnpm-store \
    pnpm install --frozen-lockfile --store-dir /pnpm-store --filter @duel-masters/web...

COPY packages/duel-board/ packages/duel-board/
COPY apps/web/ apps/web/

ENV NEXT_TELEMETRY_DISABLED=1 \
    NEXT_STANDALONE=1
# Builds the duel board, then the site.
RUN pnpm --filter @duel-masters/web build

FROM node:24-alpine

WORKDIR /app

ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    HOSTNAME=0.0.0.0 \
    PORT=3000

# The standalone server keeps the repository's layout, so the site is in
# apps/web. It does not include the static files and public folder; they are
# copied next to it.
COPY --from=build --chown=node:node /src/apps/web/.next/standalone ./
COPY --from=build --chown=node:node /src/apps/web/.next/static ./apps/web/.next/static
COPY --from=build --chown=node:node /src/apps/web/public ./apps/web/public

USER node
EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=6s --start-period=15s --retries=3 \
    CMD ["node", "-e", "fetch('http://127.0.0.1:3000/card-back.svg').then(r => process.exit(r.ok ? 0 : 1), () => process.exit(1))"]

CMD ["node", "apps/web/server.js"]
