# Docker Compose Deployment

This document explains how to run `sbterm-server` with Docker Compose.

## Deployment Files

Local/container deployment uses these files:

```text
Dockerfile
.dockerignore
docker-compose.yml
```

Service composition:

- `app`: Go binary for `sbterm-server`
- `postgres`: PostgreSQL database
- `redis`: Redis cache/in-memory data store
- `postgres-data`: named volume for PostgreSQL data, mounted to `/var/lib/postgresql` for compatibility with the PostgreSQL 18+ image layout
- `redis-data`: named volume for Redis data at `/data`

## Quick Start

Build and run all services:

```bash
docker compose up --build
```

Run in the background:

```bash
docker compose up -d --build
```

Check service status:

```bash
docker compose ps
```

Check application logs:

```bash
docker compose logs -f app
```

Check the health endpoint:

```bash
curl http://localhost:8080/health
```

Stop services:

```bash
docker compose down
```

Stop services and remove database/cache volumes:

```bash
docker compose down -v
```

## Configuration

`docker-compose.yml` uses environment variables with default values.

### App

| Variable | Default | Description |
| --- | --- | --- |
| `APP_NAME` | `sbterm-server` | Application name |
| `APP_VERSION` | `dev` | Application version and image tag |
| `APP_HOST_PORT` | `8080` | Host port published to the app |
| `APP_DB_MAX_CONNS` | `10` | Max database connections |
| `APP_DB_MIN_CONNS` | `0` | Min database connections |
| `APP_REDIS_URL` | `redis://redis:6379/0` | Redis URL inside the Docker network |
| `APP_REDIS_MAX_RETRIES` | `3` | Max Redis retry attempts |
| `APP_REDIS_POOL_SIZE` | `10` | Redis connection pool size |
| `APP_REDIS_MIN_IDLE_CONNS` | `0` | Min idle Redis connections |
| `APP_REDIS_DIAL_TIMEOUT` | `5s` | Redis dial timeout |
| `APP_REDIS_READ_TIMEOUT` | `3s` | Redis read timeout |
| `APP_REDIS_WRITE_TIMEOUT` | `3s` | Redis write timeout |
| `APP_LOG_LEVEL` | `info` | Log level |
| `APP_LOG_FORMAT` | `json` | Container log format |
| `APP_RATE_LIMIT_RATE` | `10` | Request rate per second |
| `APP_RATE_LIMIT_BURST` | `20` | Burst limit |

### PostgreSQL

| Variable | Default | Description |
| --- | --- | --- |
| `POSTGRES_USER` | `postgres` | Database user |
| `POSTGRES_PASSWORD` | `postgres` | Database password |
| `POSTGRES_DB` | `sbterm` | Database name |
| `POSTGRES_HOST_PORT` | `5432` | PostgreSQL host port |

### Redis

| Variable | Default | Description |
| --- | --- | --- |
| `REDIS_HOST_PORT` | `6379` | Redis host port |

Example shell override:

```bash
APP_VERSION=0.1.0 APP_HOST_PORT=8081 docker compose up -d --build
```

Or create a local `.env` file:

```env
APP_VERSION=0.1.0
APP_HOST_PORT=8080
POSTGRES_USER=postgres
POSTGRES_PASSWORD=change-me
POSTGRES_DB=sbterm
REDIS_HOST_PORT=6379
APP_LOG_FORMAT=json
```

> `.env` is not committed because it is already listed in `.gitignore`.

## Service URLs

Inside the Docker network, the application uses service names as hosts instead of `localhost`.

PostgreSQL:

```text
postgres://postgres:postgres@postgres:5432/sbterm?sslmode=disable
```

Redis:

```text
redis://redis:6379/0
```

To run the app on the host without Docker, use `localhost` as shown in `.env.example`.

## Healthcheck

The app container uses `curl -fsS` for a healthcheck against:

```text
http://127.0.0.1:8080/health
```

PostgreSQL uses:

```bash
pg_isready
```

Redis uses:

```bash
redis-cli ping
```

`app` only starts after `postgres` and `redis` are healthy.

## Build Notes

The Dockerfile uses a multi-stage build:

1. `golang:1.26.5-alpine` to compile a static binary.
2. `alpine:3.22` as the small runtime image.

Build arg `APP_VERSION` is embedded into the binary through ldflags:

```text
-X github.com/nofendian17/sbterm-server/internal/infrastructure/config.version=<version>
```

## Production Notes

For production, at minimum change or review:

- `POSTGRES_PASSWORD`
- `APP_VERSION`
- `APP_LOG_FORMAT=json`
- port exposure for the target environment
- backup strategy for volumes/database/cache according to persistence requirements
- secret management; do not store production passwords in a plain `.env` file

If deployed behind a reverse proxy/load balancer, review the rate-limit key strategy so all traffic is not treated as coming from the same proxy IP.
