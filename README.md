# Lemonade Tycoon

A turn-based lemonade business game: buy ingredients, run your facilities, sell lemonade, and survive the market one day at a time. Angular frontend, Gin (Go) REST API, PostgreSQL.

| Service | Source | URL |
| ------- | ------ | --- |
| `web` | [lemonade-web/](lemonade-web/) (nginx) | http://localhost:4200 |
| `api` | [lemonade-api/](lemonade-api/) | http://localhost:8080 |
| `db`  | `postgres:16` | `localhost:${DB_PORT}` |

`web` forwards `/api/*` to `api` unchanged (the API serves its routes under `/api`), so no CORS setup is needed.

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

Every balance number lives in one struct: `DefaultConfig()` in [lemonade-api/internal/domain/config.go](lemonade-api/internal/domain/config.go). Change a value there and restart the API (`docker compose up -d --build api`). The frontend shows whatever the API sends, so it needs no changes.

### The economy

| Knob | Value | What it does |
| ---- | ----- | ------------ |
| `StartingCapital` | $1,000 | Cash in a new game |
| `BasePrice` | lemon $20, sugar $10, ice $10, cup $10, **lemonade $90** | Each resource's long-run price per case |
| `Spread` | 10% | You buy at the *ask* (price +10%, rounded up) and sell at the *bid* (price −10%, rounded down, at least $1), so buying and selling straight back loses about 18% |
| `Sigma` | 0.12 | Daily price volatility: each day the price moves by about ±12% |
| `RevertRate` | 0.15 | How strongly prices are pulled back toward their base (15% of the gap per day) |
| `ClampMin` / `ClampMax` | 0.25× / 4× | Hard floor and ceiling on the walked price |
| `EventChance` | 25% | Chance per day that a new market event starts |
| `MaxLevel` / `MaxQuantity` | 4 / 10 | Highest facility level, and most buildings per warehouse resource (and for production) |
| `HistoryLength` | 14 | Days of price history kept per resource |

Prices follow `p' = p + RevertRate × (base − p) + p × Sigma × N(0,1)`, seeded by the game's seed and day number, so a game is fully reproducible. Events multiply the *quoted* price and never change the underlying walk. Quotes are whole dollars, at least $1.

One lemonade needs one case each of lemon, sugar, ice and cup. That costs about $55 at the ask and sells for about $81 at the bid, so a lemonade earns **about $22 (+40%)**, but the margin swings from a $7 loss (5th percentile) to a $59 profit (95th). Producing loses money on roughly 1 day in 9.

### Events

A new event is picked at random from this table (`Events`). Multipliers stack if several are active. `Excludes` names events that cannot run at the same time, in either direction.

| Event | Effect | Days | Excludes |
| ----- | ------ | ---- | -------- |
| Heat Wave | lemonade ×1.4, ice ×1.3 | 2 | Rainy Week |
| Rainy Week | lemonade ×0.75 | 3 | (Heat Wave) |
| Lemon Blight | lemon ×1.7 | 3 | |
| Sugar Glut | sugar ×0.7 | 2 | |
| Holiday | lemonade ×1.35 | 1 | |
| Cup Shortage | cup ×1.5 | 2 | |

To add an event, add one `EventDef` row. Nothing else changes.

### Facilities

Warehouse (one per resource; capacity = buildings × size) and Production (rate = buildings × rate). Each type has one shared level. Costs are per building; an upgrade costs *per-building cost × every building of that type*.

| | L1 | L2 | L3 | L4 |
| --- | --- | --- | --- | --- |
| **Warehouse** | Pantry | Garage | Barn | Industrial Warehouse |
| Size (cases) | 10 | 20 | 40 | 80 |
| Build one | $100 | $300 | $800 | $2,000 |
| Upgrade to next | $100 | $250 | $600 | n/a |
| Upkeep per day | $2 | $6 | $16 | $40 |
| **Production** | Kitchen | Food Truck | Bottling Plant | Lemonade Factory |
| Rate (lemonade/day) | 10 | 20 | 40 | 80 |
| Build one | $500 | $1,500 | $4,000 | $10,000 |
| Upgrade to next | $1,000 | $2,500 | $6,000 | n/a |
| Upkeep per day | $20 | $50 | $120 | $280 |

A new game has 5 Pantries and 1 Kitchen: **$30 a day** in upkeep against about $220 a day of gross profit on average. To grow, production and *all* the warehouses must grow together, and each step also raises upkeep, so over-expanding is the way to go under.

### Upkeep and bankruptcy

Upkeep is charged every night and is always owed. If cash falls short, your stock is sold at the current bid, just enough to cover it (lemonade first, then lemon, sugar and cups; ice has already melted). If even that isn't enough, the game is over. The day report lists any stock sold this way. Spending down to $0 during the day is fine.

### Why these numbers

Tuned by simulation, not by feel: simulated players (a careful one, a sloppy one that sometimes forgets the ice or spends its cushion, a careless random one, and an idle one) play hundreds of seeded games. The outcome under the defaults:

| Player | Result |
| ------ | ------ |
| Careful, keeps reinvesting | Fills level 1 by about day 28, reaches level 4 by about day 65; about 1 in 8 still go bankrupt within 90 days |
| Sloppy (forgets ice 8% of days, overspends 6%) | About 2 in 3 go bankrupt within 45 days |
| Careless (random actions) | All go bankrupt, typically by day 10 |
| Idle (does nothing) | Bankrupt around day 34 |

Two earlier problems drove the last tuning pass: the market was almost riskless (producing was unprofitable on only 1.4% of days) and a player holding *any* stock could never lose. Lemonade is now $90 (was $100), prices are more volatile, upkeep is doubled, and unpaid upkeep is no longer forgiven.

### Tuning it yourself

| To make the game… | Change |
| ----------------- | ------ |
| easier | raise `StartingCapital`, lower the `Upkeep` values, lower `Sigma`, raise `BasePrice[Lemonade]` |
| harder | the opposite, or raise `EventChance` |
| faster to progress | lower `BuildCost` and `UpgradeCost` |
| slower to progress | raise them |

Guard-rail tests (`TestBalance*` in [balance_test.go](lemonade-api/internal/domain/balance_test.go)) fail if a change makes the game too easy, too harsh, riskless or stalled. To see the numbers behind them:

```bash
cd lemonade-api
go test ./internal/domain                                            # guard-rail tests
BALANCE_REPORT=1 go test ./internal/domain -run TestBalanceReport -v # full report
```

Changing a price, cost or upkeep also changes a few exact numbers asserted in the API and domain tests (for example the $500 first warehouse upgrade); update those alongside.

## Known limitations

- Login is username-only (sent as an `X-Username` header), by design of the brief. Not secure.
