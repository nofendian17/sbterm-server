# syntax=docker/dockerfile:1

FROM golang:1.26.5-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY . .

ARG APP_VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X github.com/nofendian17/sbterm/apps/stream/internal/infrastructure/config.version=${APP_VERSION}" \
    -o /out/sbterm-stream ./apps/stream/cmd/stream

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

COPY --from=builder /out/sbterm-stream /usr/local/bin/sbterm-stream

ENTRYPOINT ["sbterm-stream"]
