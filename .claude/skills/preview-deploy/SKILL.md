---
name: preview-deploy
description: "Create docker-compose.preview.yml for any project. Generates Dockerfiles and compose config for multi-service preview deployments with Traefik routing."
version: 1.0.0
user_invocable: true
arguments: ""
---

# Preview Deploy Skill

Generate `docker-compose.preview.yml` and Dockerfiles for deploying preview environments per issue branch.

## Usage

```
/preview-deploy
```

Run in any project root to generate preview deploy configuration.

## What It Creates

1. **`docker-compose.preview.yml`** — Multi-service compose with Traefik labels
2. **`Dockerfile.preview`** per service — Optimized Docker builds
3. **`.dockerignore`** per service — Exclude node_modules, .env, etc.

## Architecture

```
Git push → Forge webhook/MCP tool → SSH to server → docker compose up
  → Traefik auto-routes: {slug}.{domain} and {slug}-api.{domain}
  → Issue previewUrl updated → Tester clicks & tests
  → Issue closed → auto-teardown
```

## Conventions

### Environment Variables (set by deploy service)
- `PREVIEW_SLUG` — e.g. `iss-42`
- `PREVIEW_DOMAIN` — e.g. `iss-42.preview.musetools.com`
- `BASE_DOMAIN` — e.g. `preview.musetools.com`

### Subdomain Pattern
- Frontend: `{slug}.{domain}`
- Backend API: `{slug}-api.{domain}`
- Additional services: `{slug}-{service}.{domain}`

### Network
- Use `preview` external network (dedicated preview Traefik on 216.18.207.12)
- Use `internal` bridge network for inter-service communication
- DB-only services need only `internal` network

### Traefik Labels Template
```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.${PREVIEW_SLUG}-{service}.rule=Host(`${PREVIEW_SLUG}-{service}.${BASE_DOMAIN}`)"
  - "traefik.http.routers.${PREVIEW_SLUG}-{service}.entrypoints=https"
  - "traefik.http.routers.${PREVIEW_SLUG}-{service}.tls=true"
  - "traefik.http.routers.${PREVIEW_SLUG}-{service}.tls.certresolver=letsencrypt"
  - "traefik.http.services.${PREVIEW_SLUG}-{service}.loadbalancer.server.port={port}"
```

## Steps

1. **Scan project** — Find all services (package.json, Cargo.toml, go.mod, etc.)
2. **Detect frameworks** — Next.js, Strapi, Express, Fastify, etc.
3. **Identify ports** — From config files, scripts, or defaults
4. **Identify env vars** — Especially URLs that connect services
5. **Generate Dockerfiles** — One per service, named `Dockerfile.preview`
6. **Generate compose** — Wire services with env vars and Traefik labels
7. **Generate .dockerignore** — For each service with a Dockerfile

## Common Patterns

### Next.js Frontend (runtime env replacement)
```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ENV NEXT_PUBLIC_API_URL=__NEXT_PUBLIC_API_URL__
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/.next/standalone/app ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
EXPOSE 3000
CMD ["sh", "-c", "find .next -name '*.js' -exec sed -i \"s|__NEXT_PUBLIC_API_URL__|$NEXT_PUBLIC_API_URL|g\" {} + && exec node server.js"]
```

Requires `output: "standalone"` in `next.config.ts`.

### Strapi Backend
```dockerfile
FROM node:20-alpine
WORKDIR /app
RUN apk add --no-cache python3 make g++ git
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build
EXPOSE 1337
CMD ["npm", "run", "start"]
```
