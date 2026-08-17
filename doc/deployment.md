# Docker Compose Deployment

This document explains how to run `sbterm-server` with Docker Compose.

## Deployment Files

Local/container deployment uses these files:

```text
Dockerfile
.dockerignore
docker-compose.yml
config.yaml.example
config.yaml          # your local config; copy from config.yaml.example
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

The app is configured exclusively through `config.yaml`, mounted read-only into the container at `/app/config.yaml`. Copy the template and edit it before starting:

```bash
cp config.yaml.example config.yaml
```

For Docker Compose the app reaches `postgres` and `redis` through the service names, so set the hosts accordingly in `config.yaml`:

```yaml
database:
  url: postgres://postgres:postgres@postgres:5432/sbterm?sslmode=disable

redis:
  url: redis://redis:6379/0
```

### Compose variables

The remaining variables below only affect Docker Compose infrastructure (image tag, host ports, and database credentials). They are unrelated to app config:

| Variable | Default | Description |
| --- | --- | --- |
| `APP_VERSION` | `dev` | Application version and image tag |
| `APP_HOST_PORT` | `8080` | Host port published to the app |

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

### QuestDB

| Variable | Default | Description |
| --- | --- | --- |
| `QUESTDB_HOST_PORT` | `9000` | QuestDB host port |
| `QDB_HTTP_USER` | `questdb` | HTTP basic auth user for REST, Web Console, and QWP |
| `QDB_HTTP_PASSWORD` | `questdb` | HTTP basic auth password |

QuestDB runs with HTTP basic auth enabled (`QDB_HTTP_USER`/`QDB_HTTP_PASSWORD`). The app authenticates through the QWP connect string in `config.yaml`:

```yaml
questdb:
  url: ws::addr=questdb:9000;username=questdb;password=questdb;
```

Keep the credentials in `config.yaml` in sync with the compose variables; changing them in `.env` requires the same update in `config.yaml`.

Example override:

```bash
APP_VERSION=0.1.0 APP_HOST_PORT=8081 docker compose up -d --build
```

Or create a `.env` file at the repository root for the compose-level variables:

```env
APP_VERSION=0.1.0
APP_HOST_PORT=8080
POSTGRES_USER=postgres
POSTGRES_PASSWORD=change-me
POSTGRES_DB=sbterm
REDIS_HOST_PORT=6379
```

> `.env` is optional and only feeds the Compose variables above; application configuration always comes from `config.yaml`.

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

To run the app on the host without Docker, use `localhost` as shown in `config.yaml.example`.

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

QuestDB uses a curl healthcheck against its authenticated REST endpoint:

```bash
curl -u "$QDB_HTTP_USER:$QDB_HTTP_PASSWORD" 'http://127.0.0.1:9000/exec?query=select%201'
```

`app` only starts after `postgres`, `redis`, and `questdb` are healthy.

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
- `log.format: json` in `config.yaml`
- port exposure for the target environment
- backup strategy for volumes/database/cache according to persistence requirements
- secret management; do not store production passwords in `config.yaml` (mount it read-only from a secret store)

If deployed behind a reverse proxy/load balancer, review the rate-limit key strategy so all traffic is not treated as coming from the same proxy IP.
