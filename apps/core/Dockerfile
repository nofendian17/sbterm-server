# syntax=docker/dockerfile:1

FROM golang:1.26.5-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY . .

ARG APP_VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X github.com/nofendian17/sbterm/apps/core/internal/infrastructure/config.version=${APP_VERSION}" \
    -o /out/sbterm-core ./apps/core/cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

COPY --from=builder /out/sbterm-core /usr/local/bin/sbterm-core

EXPOSE 8082

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8082/healthz >/dev/null || exit 1

ENTRYPOINT ["sbterm-core"]
