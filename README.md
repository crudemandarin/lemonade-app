# Lemonade Tycoon

A turn-based lemonade business game: buy ingredients, run your facilities, sell lemonade, and survive the market one day at a time. Angular frontend, Gin (Go) REST API, PostgreSQL.

| Service | Source | URL |
| ------- | ------ | --- |
| `web` | [lemonade-web/](lemonade-web/) (nginx) | http://localhost:4200 |
| `api` | [lemonade-api/](lemonade-api/) | http://localhost:8080 |
| `db`  | `postgres:16` | `localhost:${DB_PORT}` |

`web` forwards `/api/*` to `api` and removes the `/api` prefix, so no CORS setup is needed.

Design docs: [SPEC](docs/claude/SPEC.md), [DESIGN](docs/claude/DESIGN.md), [PLAN](docs/claude/PLAN.md), [DECISIONS](docs/claude/DECISIONS.md).

## Requirements

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (running), to start the full stack
- For local development and tests: [Go](https://go.dev/dl/) 1.27+ and [Node.js](https://nodejs.org/) 20+ with Chrome (for headless frontend tests)

## Install

```bash
cp .env.example .env                        # DB credentials (defaults work locally)
cd lemonade-api && go mod download && cd ..
cd lemonade-web && npm ci && cd ..
```

`.env.example` contains:

```
DB_HOST=127.0.0.1
DB_USERNAME=admin
DB_PASSWORD=password
DB_NAME=sample
DB_PORT=5432
```

## Run

### Full stack (Docker Compose)

```bash
docker compose up -d --build
open http://localhost:4200                  # the game
curl http://localhost:8080/healthz          # API health: {"status":"ok"}
curl http://localhost:4200/api/healthz      # same, through the web proxy
```

```bash
docker compose down           # stop
docker compose down -v        # stop and delete the database volume
docker compose logs -f api    # tail logs (also: web, db)
```

> Postgres reads `DB_PASSWORD` only when it creates a new volume. After changing it, run `docker compose down -v` (this deletes data) or `ALTER USER`.

### Local development (hot reload)

```bash
docker compose up -d db                                  # database only
cd lemonade-api && cp ../.env .env && go run .           # API on :8080
cd lemonade-web && npm start                             # web on :4200, proxies /api to :8080
```

Inside Compose the API always connects to `db:5432`. `DB_HOST` only matters when you run the API outside Docker.

## Test

```bash
cd lemonade-api && go test ./...
cd lemonade-web && npx ng test --watch=false --browsers=ChromeHeadless
```

Lint and format: `gofmt -w . && go vet ./...` in `lemonade-api/`; `npm run lint && npm run format` in `lemonade-web/`.

## Deploy

Google Cloud (Cloud Run + Cloud SQL), see [deploy/](deploy/README.md). Pushes to `main` redeploy automatically.

## Game physics (tuning)

_TODO (slice 8): document the config tables in `lemonade-api` (prices, spread, events, facility costs) and how to tune them._

## Known limitations

- Login is username-only (sent as an `X-Username` header), by design of the brief. Not secure.
