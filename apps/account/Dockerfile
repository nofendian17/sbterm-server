# syntax=docker/dockerfile:1

FROM golang:1.26.5-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY . .

ARG APP_VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X github.com/nofendian17/sbterm/apps/account/internal/infrastructure/config.version=${APP_VERSION}" \
    -o /out/sbterm-account ./apps/account/cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

COPY --from=builder /out/sbterm-account /usr/local/bin/sbterm-account

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8081/healthz >/dev/null || exit 1

ENTRYPOINT ["sbterm-account"]
