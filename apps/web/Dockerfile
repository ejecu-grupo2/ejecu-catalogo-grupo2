# syntax=docker/dockerfile:1.7
FROM node:22.22.0-alpine3.23@sha256:e4bf2a82ad0a4037d28035ae71529873c069b13eb0455466ae0bc13363826e34 AS web
WORKDIR /src
RUN npm install --global pnpm@10.13.1
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml .npmrc ./
COPY apps/web/package.json apps/web/package.json
RUN --mount=type=cache,target=/pnpm/store pnpm install --frozen-lockfile --ignore-scripts --store-dir /pnpm/store
COPY apps/web/ apps/web/
RUN pnpm build
FROM --platform=$BUILDPLATFORM golang:1.26.11-alpine3.23@sha256:db99415e2625ce0ee03eed09edf7f32ae8c58dc399f28cc602a87fe7238119e4 AS server
WORKDIR /src
COPY apps/web/server/ ./
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/spa-server .
FROM gcr.io/distroless/static-debian13:nonroot@sha256:f7f8f729987ad0fdf6b05eeeae94b26e6a0f613bdf46feea7fc40f7bd72953e6
WORKDIR /app
COPY --from=server --chown=nonroot:nonroot /out/spa-server /app/spa-server
COPY --from=web --chown=nonroot:nonroot /src/apps/web/dist /srv
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/spa-server"]
