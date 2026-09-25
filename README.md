# Lemonade Tycoon

A turn-based lemonade business game: buy ingredients, run your facilities, sell lemonade, and survive the market one day at a time. Angular frontend, Gin (Go) REST API, PostgreSQL.

> **Design:** the technical design doc, [docs/numeric-tdd.md](docs/numeric-tdd.md), covers architecture, game rules, data model, API, balance and tuning, and trade-offs. Start there.

| Service | Source | URL |
| ------- | ------ | --- |
| `web` | [lemonade-web/](lemonade-web/) (nginx) | http://localhost:4200 |
| `api` | [lemonade-api/](lemonade-api/) | http://localhost:8080 |
| `db`  | `postgres:16` | `localhost:${DB_PORT}` |

`web` forwards `/api/*` to `api` unchanged (the API serves its routes under `/api`), so no CORS setup is needed.

Working docs (in [docs/claude/](docs/claude/)): [SPEC](docs/claude/SPEC.md) (rules), [DESIGN](docs/claude/DESIGN.md) (architecture), [PLAN](docs/claude/PLAN.md) (slices), [DECISIONS](docs/claude/DECISIONS.md) (deviations and trade-offs), [ai-usage-log](docs/claude/ai-usage-log.md).

## Code structure

```
lemonade-api/    Go + Gin REST API. internal/domain holds every game rule (pure, no I/O);
                 internal/store is Postgres persistence; internal/api is thin HTTP handlers
lemonade-web/    Angular 17 SPA + PWA. Renders server state only; GameStore is the one API client
deploy/          Terraform + scripts for Google Cloud (Cloud Run, Cloud SQL)
docs/            spec, design, decisions, plan, AI usage log
docker-compose.yml   db + api + web for local use
.github/         deploy-on-push workflow
```

Each codebase has its own README with layout and commands: [lemonade-api](lemonade-api/README.md), [lemonade-web](lemonade-web/README.md), [deploy](deploy/README.md).

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
curl http://localhost:4200/api/health      # API health via the web proxy: {"status":"ok"}
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
(cd lemonade-api && cp ../.env .env && go run .)         # API on :8080 (keeps running; use a new terminal for the next line)
(cd lemonade-web && npm start)                           # web on :4200, proxies /api to :8080
```

Inside Compose the API always connects to `db:5432`. `DB_HOST` only matters when you run the API outside Docker.

## Test

```bash
(cd lemonade-api && go test ./...)
(cd lemonade-web && npx ng test --watch=false --browsers=ChromeHeadless)
```

Lint and format: `gofmt -w . && go vet ./...` in `lemonade-api/`; `npm run lint && npm run format` in `lemonade-web/`.

## Deploy

Google Cloud (Cloud Run + Cloud SQL), see [deploy/](deploy/README.md). Pushes to `main` redeploy automatically.

## Tuning the game

All balance numbers (prices, volatility, events, facility costs and upkeep) are one struct: `DefaultConfig()` in [lemonade-api/internal/domain/config.go](lemonade-api/internal/domain/config.go). Change a value and restart the API (`docker compose up -d --build api`); the frontend needs no changes. The values, what each knob does and the balance analysis are in the [technical design doc, section 9](docs/numeric-tdd.md#9-balance-and-tuning).

```bash
cd lemonade-api
go test ./internal/domain                                            # balance guard-rail tests
BALANCE_REPORT=1 go test ./internal/domain -run TestBalanceReport -v # full report
```

Changing a price, cost or upkeep also changes a few exact numbers asserted in the API and domain tests (for example the $500 first warehouse upgrade); update those alongside.

## Known limitations

- Login is username-only (sent as an `X-Username` header), by design of the brief. Not secure: anyone can play as anyone by typing their name. Usernames are 3 to 40 ASCII characters and not case sensitive (`Joe` and `joe` are the same player).
- Usernames are public on the global high score board, and because login is username-only, anyone can type another player's name and play (or read their run history) as them. A player's run detail is readable only with their username; other players' runs are not viewable from the board.
