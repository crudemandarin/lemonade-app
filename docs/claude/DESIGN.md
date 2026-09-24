# DESIGN: Lemonade Tycoon

## 1. Stack
- **Backend:** Go + Gin. **DB:** PostgreSQL (use whatever driver the scaffold already has). **Frontend:** Angular + TypeScript (standalone components, signals). Scaffold and cloud deploy already exist; PWA support does not (added in slice 7, see §7).
- The backend is the source of truth. The frontend never computes game outcomes; it renders server state.

## 2. Guiding principle
All game rules live in a **pure Go domain package** (no I/O, no Gin, no SQL). API handlers load state, call a domain function, and save the result. This makes rules fast to test and keeps the API and DB layers thin.

## 3. Backend modules
```
cmd/server/            main: wiring, config, router
internal/domain/       PURE. types, config, market, events, actions, facilities, endday, bankruptcy
internal/store/        Repository interface + Postgres impl (+ in-memory fake for tests)
internal/api/          Gin handlers, DTOs, error mapping, username middleware
```
Domain public surface (mutating funcs return a typed error; `EndDay` returns a report):
```go
NewGame(cfg Config, seed int64) Game
Buy(g *Game, r Resource, qty int) error
Sell(g *Game, r Resource, qty int) error
Expand(g *Game, k FacilityKind) error     // quantity + 1 at current level
Upgrade(g *Game, k FacilityKind) error    // level + 1 for all buildings
EndDay(g *Game, cfg Config) DayReport     // produce, melt, upkeep, advance, market tick, events, bankruptcy
Quotes(g Game) map[Resource]Quote         // effective price, bid, ask (whole dollars)
Capacity(g Game, r Resource) int          // quantity * size(level)
// bankruptcy is decided inside EndDay: upkeep unpayable even after selling all stock at bid (DECISIONS 12)
```
**Determinism:** randomness comes from `rand.New(rand.NewSource(seed ^ int64(day)))` created inside `EndDay`. No `time.Now()` in the domain. Same seed and same actions give the same game.

## 4. Data model
Money is `int` whole dollars. Each user has one active game. The game's mutable state is small, so it is stored relationally where queried and as JSONB where it is just carried along.

| Table | Columns |
|---|---|
| `users` | `id`, `username` (unique), `created_at` |
| `games` | `id`, `user_id` (unique among active), `seed`, `day`, `capital`, `status` (`active`/`bankrupt`), `inventory` JSONB `{lemon,sugar,ice,cup,lemonade}`, `facilities` JSONB `{kind: {level, quantity}}`, `market` JSONB (walked price as float + price history array per resource, last 14), `events` JSONB (active events with days remaining), `updated_at` |

Facility kinds: `warehouse_lemon`, `warehouse_sugar`, `warehouse_ice`, `warehouse_cup`, `warehouse_lemonade`, `production`. The tier name (Pantry, Garage, ...) is derived from kind family + level via config, never stored.

Rationale: the domain loads and saves a whole `Game` in one transaction, so JSONB avoids join tables that add no query value. Trade-off: no SQL analytics on inventory or history; fine for this scope (a leaderboard would use `capital` and `day`, both real columns).

Persistence rule: every mutating request does `load game -> domain call -> save game` in one transaction with a row lock (`SELECT ... FOR UPDATE`) to prevent double-submit races.

## 5. API (all JSON, `X-Username` header except login)
| Method & path | Purpose |
|---|---|
| `POST /api/login` `{username}` | Create-or-get user; returns user |
| `GET /api/game` | Game view: day, capital, inventory, capacities, facilities (tier name, level, quantity, capacity, upkeep, expand cost, upgrade cost), quotes, events, history |
| `POST /api/game/new` | Start fresh game (replaces bankrupt or active one) |
| `POST /api/game/buy` `{resource, qty}` | Buy at ask |
| `POST /api/game/sell` `{resource, qty}` | Sell at bid |
| `POST /api/game/facilities/:kind/expand` | Add one building |
| `POST /api/game/facilities/:kind/upgrade` | Upgrade all buildings one level |
| `POST /api/game/end-day` | Returns `DayReport` + new game view |
| `GET /api/health` | Health (deployment check; not `/healthz`, which Cloud Run reserves) |

Every mutation returns the updated game view so the UI needs no follow-up fetch.

## 6. Balance config (`internal/domain/config.go`; every knob in one struct)
| Item | Value |
|---|---|
| Starting capital | $1,000 |
| Base price per case | lemon $20, sugar $10, ice $10, cup $10, lemonade $90 (was $100; DECISIONS 12) |
| Spread | 10%. Ask = ceil(p·1.1); bid = max(1, floor(p·0.9)) |
| Walk | `p' = p + 0.15(base-p) + p·σ·N(0,1)`, σ = 0.12 (was 0.2 / 0.08); clamp [0.25, 4]×base; float state, rounded at quote time |
| Max level / max quantity | 4 / 10 |
| Event chance per day | 25% |

Facility tiers (L1 → L4). Size/rate is per building; costs and upkeep are per building.

| | L1 | L2 | L3 | L4 |
|---|---|---|---|---|
| **Warehouse name** | Pantry | Garage | Barn | Industrial Warehouse |
| Size (cases) | 10 | 20 | 40 | 80 |
| Expand (build) cost | $100 | $300 | $800 | $2,000 |
| Upgrade cost to next | $100 | $250 | $600 | n/a |
| Upkeep/day | $2 | $6 | $16 | $40 |
| **Production name** | Kitchen | Food Truck | Bottling Plant | Lemonade Factory |
| Rate (lemonade/day) | 10 | 20 | 40 | 80 |
| Expand (build) cost | $500 | $1,500 | $4,000 | $10,000 |
| Upgrade cost to next | $1,000 | $2,500 | $6,000 | n/a |
| Upkeep/day | $20 | $50 | $120 | $280 |

Upgrade total = per-building upgrade cost × quantity. Upkeep total = per-building upkeep × quantity, summed over all six facilities.

Sanity check: start = 5 Pantries + 1 Kitchen, upkeep $30/day, 10 cases capacity each. One batch of 10 costs about $550 at ask; it yields 10 lemonade selling at about $81 bid = $810. Profit ≈ $230/day at start after upkeep. To scale up, production and **all** input and output warehouses must grow together, so the bottleneck shifts between them (the intended strategic tension). Balance was tuned by simulation (`balance_test.go`); see the README's "Game physics" section. Upkeep is always owed: short cash sells stock at bid, and if that is not enough the game ends (DECISIONS 12).

Events (table-driven): Heat Wave (lemonade ×1.4, ice ×1.3, 2d), Rainy Week (lemonade ×0.75, 3d), Lemon Blight (lemon ×1.7, 3d), Sugar Glut (sugar ×0.7, 2d), Holiday (lemonade ×1.35, 1d), Cup Shortage (cup ×1.5, 2d). Adding an event is one table row.

## 7. Frontend
- `core/api.service.ts`: typed HTTP client, adds `X-Username`. `core/game.store.ts`: signals holding the latest game view and last day report.
- Pages: `login`, `game` (dashboard: header stats, `market-panel`, `inventory-panel`, `facilities-panel`, `events-banner`, `day-report-modal`), `game-over`.
- `app-event-backdrop` (app shell) draws a looping CSS-only background for up to two active events; see EVENT-BACKDROPS.md.
- Components are presentational, with inputs and outputs. Only `game.store` talks to the API. Sparkline is a small inline-SVG component (no chart library).
- Facility images: 4 warehouse-tier and 4 production-tier static assets, chosen by level, with a small resource icon for warehouses.
- **PWA (slice 7):** added with `ng add @angular/pwa` (manifest, icons, `ngsw-config.json`, service worker registered in production builds only). `ngsw-config.json` prefetches the app shell as an asset group and defines **no** data group for `/api/**`, so API calls always hit the network. An `online.service.ts` signal (from `navigator.onLine` and the `online`/`offline` events) drives an offline banner and disables all action buttons. Installability is verified with Lighthouse on the deployed HTTPS URL, since service workers do not run in `ng serve`.

## 8. Testing strategy
- **Domain (most tests):** table-driven Go unit tests per SPEC rule (buy/sell limits, capacity = quantity × size, bid/ask rounding, expand, upgrade multiplying by quantity, production min(), melt, upkeep clamp, bankruptcy truth table (capital 0 × any inventory; ice is already melted), price clamp, event expiry). Determinism test: same seed and actions give identical state.
- **API:** handler tests using the in-memory repo fake: status codes, error JSON shape, username middleware.
- **Store:** one integration test against real Postgres (skipped if `DATABASE_URL` is unset): save/load round trip and lock behavior.
- **Frontend:** unit test for `game.store`; component tests for the market panel (buy/sell emit), facilities panel (expand/upgrade emit), and game-over; `online.service` (mocked online/offline events) and offline-disabled action buttons. Manual Lighthouse installability check on the deployed URL. No e2e (Future work).
- Full test suite plus a manual run of the app before every slice is ticked.

## 9. Key trade-offs (each becomes a DECISIONS.md entry when built)
1. Pure domain, thin layers: costs a DTO mapping step, buys fast, deterministic tests.
2. JSONB for game internals vs. normalized tables: faster to build, less queryable.
3. Seed-derived RNG per day vs. stored RNG state: reproducible without persisting generator state.
4. Bid/ask spread vs. flat price: one line of code that blocks a trivial buy-sell exploit.
5. Whole-dollar money with "case" units: avoids float and cent handling, at the cost of bulk-sized prices and rounding on quotes.
6. One shared level per facility (upgrade is all-or-nothing): simple and matches the design; loses mixed-tier setups.
7. Bankruptcy = upkeep cannot be paid even after selling all stock at bid (DECISIONS 12-14): no zombie games at $0 holding a little stock, at the cost of a harsher game than the first draft.
8. Username-header "auth": matches the brief, not secure; documented.
9. PWA = installable shell only, API never cached: avoids stale-state bugs in a server-authoritative game, at the cost of no offline play.
