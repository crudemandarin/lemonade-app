# lemonade-api

The Gin + PostgreSQL API for Lemonade Tycoon. All game rules live in a pure domain package; the API layer loads a game, calls a domain function, and saves the result. See [DESIGN](../docs/claude/DESIGN.md).

## Dependencies

- [Go](https://go.dev/dl/) 1.27+ (modules are fetched automatically from `go.mod`)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (for Postgres, and optionally the API)

## Project layout

```
main.go                 entrypoint: loads secrets, connects DB, migrates, wires routes
internal/domain/        pure game rules (no I/O, no Gin, no SQL): config, market, events,
                        actions, facilities, end of day, bankruptcy
internal/store/         Repository interface, Postgres implementation, in-memory fake
internal/api/           Gin handlers, DTOs (camelCase JSON), error mapping, /api/health
libraries/              secrets loading, database connection
Dockerfile              API image (also used by the top-level docker-compose.yml)
```

## 1. Configure environment

```bash
cp .env.example .env
```

```
DB_HOST=127.0.0.1
DB_USERNAME=admin
DB_PASSWORD=password
DB_NAME=sample
DB_PORT=5432
```

`.env` is gitignored, so never commit real credentials.

## 2. Run the backend

Requires Postgres (step 3).

```bash
go run .    # reads .env, listens on :8080
```

### With Docker

```bash
docker build -t lemonade-api .
docker run --rm --name lemonade-api -p 8080:8080 \
  --env-file .env \
  -e DB_HOST=host.docker.internal \
  --add-host=host.docker.internal:host-gateway \
  lemonade-api
```

`127.0.0.1` inside a container means the container itself, so `DB_HOST=host.docker.internal` points the API at Postgres on your machine. `--add-host` is only needed on Linux.

## 3. Start Postgres

```bash
set -a; source .env; set +a
docker run -d --name lemonade-api-db \
  -e POSTGRES_USER="$DB_USERNAME" \
  -e POSTGRES_PASSWORD="$DB_PASSWORD" \
  -e POSTGRES_DB="$DB_NAME" \
  -p "$DB_PORT":5432 \
  -v lemonade-api-db-data:/var/lib/postgresql/data \
  postgres:16
```

```bash
docker stop lemonade-api-db    # stop
docker start lemonade-api-db   # start again
docker rm -f lemonade-api-db && docker volume rm lemonade-api-db-data   # delete container and data
```

Postgres reads the credentials only when it sets up an empty volume. To change the password later, delete the volume or run `ALTER USER`.

## 4. Try it out

```bash
curl http://localhost:8080/api/health
curl -X POST http://localhost:8080/api/login -H "Content-Type: application/json" -d '{"username":"lemonjoe"}'
curl http://localhost:8080/api/game -H "X-Username: lemonjoe"
```

## Test

```bash
go test ./...    # the Postgres integration test is skipped unless DATABASE_URL is set
```

## Format and lint

```bash
gofmt -w .           # format (VS Code does this on save)
golangci-lint run    # lint using .golangci.yml; install with: brew install golangci-lint
```

## API reference

Every `/api/game` route needs an `X-Username` header (username-only auth, intentionally not secure). Every mutation returns the updated game view; errors return `{"error": "<code>", "message": "..."}`. The response shapes are defined in [api.models.ts](../lemonade-web/src/app/core/api.models.ts).

| Method | Path | Description |
|--------|------|-------------|
| GET    | `/api/health` | Liveness check (no auth) |
| POST   | `/api/login` `{username}` | Create or get a user (a new user gets a new game) |
| GET    | `/api/game` | Current game view |
| POST   | `/api/game/new` | Start a fresh game |
| POST   | `/api/game/buy` `{resource, qty}` | Buy at ask |
| POST   | `/api/game/sell` `{resource, qty}` | Sell at bid |
| POST   | `/api/game/facilities/warehouse/expand` `{resource}` | Add a warehouse building for one resource |
| POST   | `/api/game/facilities/production/expand` | Add a production building |
| POST   | `/api/game/facilities/:type/upgrade` | Upgrade every building of `warehouse` or `production` |
| POST   | `/api/game/end-day` | End the day; returns the day report and the new game view |

The game view also carries `timeline` (capital and stock after each action, for the history charts) and `stats` (running totals for the end-of-game report). Old days are compacted to milestones to keep it small.

Game balance (prices, spread, events, tier costs) is one struct: `DefaultConfig()` in `internal/domain/config.go`.
