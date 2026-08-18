# syntax=docker/dockerfile:1

FROM golang:1.26.5-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY . .

ARG APP_VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X github.com/nofendian17/sbterm/apps/ingest/internal/infrastructure/config.version=${APP_VERSION}" \
    -o /out/sbterm-ingest ./apps/ingest/cmd/ingest

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/sbterm-ingest /usr/local/bin/sbterm-ingest

ENTRYPOINT ["sbterm-ingest"]
