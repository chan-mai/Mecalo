FROM node:24-alpine AS build
RUN corepack enable
WORKDIR /src
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY api/package.json api/
COPY frontend/package.json frontend/
RUN pnpm install --frozen-lockfile --filter @mecalo/frontend --ignore-scripts
COPY frontend frontend
RUN pnpm --filter @mecalo/frontend build

FROM node:24-alpine
WORKDIR /app
COPY --from=build /src/frontend/.output .output
ENV HOST=0.0.0.0 \
	PORT=3000 \
	NUXT_MECALO_BE_API_URI=http://backend:8080
USER node
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
