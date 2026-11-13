# syntax=docker/dockerfile:1.7
FROM node:22-bookworm-slim AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci

FROM node:22-bookworm-slim AS build
WORKDIR /app
ARG NEXT_PUBLIC_RELEASE=development
ENV NEXT_PUBLIC_RELEASE=$NEXT_PUBLIC_RELEASE
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN --mount=type=cache,target=/app/.next/cache npm run build

FROM node:22-bookworm-slim
WORKDIR /app
ARG NEXT_PUBLIC_RELEASE=development
LABEL org.opencontainers.image.revision=$NEXT_PUBLIC_RELEASE
ENV NODE_ENV=production
# Optional OIDC settings are read at runtime by the server routes:
# OIDC_ISSUER, OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, OIDC_REDIRECT_URI,
# OIDC_ALLOWED_EMAILS, and NEXT_PUBLIC_OIDC_ENABLED=true.
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
COPY --from=build /app/public ./public
USER node
EXPOSE 3000
CMD ["node", "server.js"]
