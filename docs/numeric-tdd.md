# Technical Design: Lemonade Tycoon

Turn-based lemonade business game. One loop: a **day**. Score = capital. Angular frontend, Gin (Go) backend, PostgreSQL. Username-only login, installable PWA. Sources: `docs/claude/` (SPEC, DESIGN, PLAN, DECISIONS, UX-MOCKS). Where they conflict, DECISIONS and the UX addendum win.

## 1. Scope
**In:** login; buy/sell five resources at market bid/ask; expand and upgrade two facility types; end day with a report; bankruptcy; seeded price walk plus random events; PWA shell (no offline play); README that installs, runs, and tests from a fresh clone (with the tuning knobs).
**Out (Future work):** mixed tiers, demand simulation, price impact from player trades, event forecasting, loans, real auth, leaderboard, e2e tests.

## 2. Architecture
- **Server is the source of truth.** The UI renders server state and never computes outcomes; every mutation returns the full game view.
- **Pure domain** (`internal/domain`: no I/O, Gin, or SQL) ← thin `internal/api` (Gin handlers, DTOs, error mapping, username middleware) ← `internal/store` (`Repository` interface, Postgres/GORM, in-memory fake). `main.go` stays at the module root because the Dockerfile and deploy scripts build it.
- **Determinism:** `rand.New(rand.NewSource(seed ^ int64(day)))` inside `EndDay`; no clock in the domain. Seed comes from the clock at game creation (API layer), so the same seed and actions give the same game.
- **Concurrency:** every mutation is `load → domain call → save` in one transaction with `SELECT ... FOR UPDATE` (proven by a 20-goroutine test).

## 3. Game rules (all money is whole-dollar `int`; unit = one "case")
- **Resources:** lemon, sugar, ice, cup (inputs); lemonade (product). Recipe: 1 of each input → 1 lemonade. Start: **$1,000**, empty stock, everything level 1.
- **Prices:** base lemon $20, sugar/ice/cup $10, lemonade $90. Walk (float): `p' = p + 0.15(base−p) + p·0.12·N(0,1)`, clamped to [0.25×, 4×] base. **Effective price** = `max(1, round(walked × Π active event multipliers))` (events never change the walk). **Ask** = `ceil(price×1.1)`, **bid** = `max(1, floor(price×0.9))` on float-snapped values (`100×1.1` must give 110, not 111). Selling is unlimited depth at bid, including raw inputs. Buy/sell fail atomically: no partial fills.
- **Facilities:** two types, one **shared level (1–4) per type**. *Warehouse*: one per resource, each with its own building count (max 10); holds only its own resource; capacity = `count × size`. *Production*: one count (max 10); output/day = `count × rate`.

| | L1 | L2 | L3 | L4 |
|---|---|---|---|---|
| Warehouse | Pantry | Garage | Barn | Industrial Warehouse |
| Size (cases) · build · upgrade→next · upkeep | 10 · $100 · $100 · $2 | 20 · $300 · $250 · $6 | 40 · $800 · $600 · $16 | 80 · $2,000 · n/a · $40 |
| Production | Kitchen | Food Truck | Bottling Plant | Lemonade Factory |
| Rate/day · build · upgrade→next · upkeep | 10 · $500 · $1,000 · $20 | 20 · $1,500 · $2,500 · $50 | 40 · $4,000 · $6,000 · $120 | 80 · $10,000 · n/a · $280 |

  **Expand** adds one building at the current level (warehouse: player picks the resource). **Upgrade** raises the whole type one level for `per-building cost × total buildings of the type` (fresh game: 5 Pantries → Garages = $500). Costs, sizes, upkeep, and event data live in one `Config` struct.
- **End of day (one atomic step):** (1) produce `min(rate, stock of each input, free lemonade space)`; (2) ice melts to 0; (3) pay upkeep `Σ buildings × upkeep(level)` (start: $30/day). Upkeep is always owed: a cash shortfall is covered by selling stock at bid (lemonade, lemon, sugar, cup, ice); if that still can't cover it, pay what's left and the game is over; (4) day + 1, market walks, events expire then may spawn; (5) build the day report. Bankruptcy is only checked here, so spending to $0 mid-day is legal. After game over every action is rejected until "New game" (replaces the row).
- **Events** (25%/day, table-driven; adding one is one row): Heat Wave (lemonade ×1.4, ice ×1.3, 2d), Rainy Week (lemonade ×0.75, 3d), Lemon Blight (lemon ×1.7, 3d), Sugar Glut (sugar ×0.7, 2d), Holiday (lemonade ×1.35, 1d), Cup Shortage (cup ×1.5, 2d). Multipliers stack; visible the day they start; an active event never re-spawns; `Excludes` blocks pairs (heat wave ↔ rainy week).

## 4. Data model
`users(id, username unique, created_at)`; `games(id, user_id unique, seed, day, capital, status active|bankrupt, warehouse_level, production_level, production_qty, inventory JSONB, warehouse_qty JSONB, market JSONB {price float, previousEffective, history ≤14}, events JSONB, updated_at)`. Scalars a leaderboard would query are real columns; carried-along state is JSONB. Tier names are derived from type + level, never stored. One game row per user (no history of past games).

## 5. API (JSON, camelCase, contract = `lemonade-web/src/app/core/api.models.ts`)
`X-Username` header on all game routes (intentionally not secure); unknown or missing → 401, and the UI then clears the session. Errors: `{"error": "<code>", "message": "..."}` (400 `invalid_*`; 409 `insufficient_funds|insufficient_stock|capacity_exceeded|max_level|max_quantity|game_over`).

| Route | Purpose |
|---|---|
| `POST /api/login {username}` | create-or-get user (+ new game; username 5–40 ASCII chars, case-insensitive); returns `{id, username}` |
| `GET /api/game` · `POST /api/game/new` | view · fresh game |
| `POST /api/game/buy \| sell {resource, qty}` | trade at ask · bid |
| `POST /api/game/facilities/warehouse/expand {resource}` · `.../production/expand` | add one building |
| `POST /api/game/facilities/{warehouse\|production}/upgrade` | upgrade whole type |
| `POST /api/game/end-day` | `{report, game}` |
| `GET /api/health` | liveness only (Cloud Run reserves `/healthz`) |

The view is display-ready (tier names, bid/ask, capacities, costs, `upgrade: null` at max level), so the client never recomputes rules.

## 6. Frontend and PWA
Angular 17 standalone components and signals. Only `GameStore` calls `ApiService`; components are presentational. Failed actions show the server message and keep the view; End day is disabled while a request is in flight. PWA: manifest, icons, service worker (production builds only) prefetching the shell with **no data group for `/api/**`**; `online.service` drives an offline banner and disables all actions offline. Installability is verified with Chrome's `getInstallabilityErrors` (Lighthouse 12 dropped its PWA audit); nginx serves the manifest as `application/manifest+json` with `no-cache`.

## 7. Testing
Table-driven domain tests per rule (buy/sell limits, capacity, rounding across $1–$500, expand/upgrade cost scaling, production `min()`, melt, upkeep, bankruptcy truth table, clamp over 1,000 days, event stacking/expiry/exclusion, determinism); API handler tests on the in-memory repo (status codes, error shape, contract fields); one Postgres integration test (`DATABASE_URL`, skipped if unset, cleanup scoped to test users); frontend store, component, and offline tests. `go test ./...` and `ng test` must pass before each slice is ticked.

## 8. Key trade-offs and limits
Pure domain (needs DTO mapping) · JSONB (no SQL analytics) · seed-derived RNG (no generator state stored) · bid/ask spread (blocks same-day arbitrage) · whole dollars (bulk-sized prices) · shared level per type (no Barn-for-lemons, upgrade cost grows with expansion) · liquidation grace instead of "any inventory survives" (a player can't sit at $0 holding stock) · header auth (documented) · PWA shell only (no stale-state bugs, no offline play). Open: the brief mentioned three facility types but named two (modeled as two; a third is another type with its own level and tiers).

---

# Appendix: UX mocks
Copy is sentence case, no emoji, one icon set, **one primary button per screen**. Routes: `/` Home, `/signin`, `/game` (also shows game over when `status = bankrupt`; redirects to `/signin` with no stored username). "Log out" clears the stored username (there is no session).

**Nav (every page)**
```
Signed out: | [logo] Lemonade Tycoon                                   [ Sign in ]   |
Signed in:  | [logo] Lemonade Tycoon                       (user) lemonjoe [Log out] |
```

**Home** (signed in: "Play game" reads "Continue game" → `/game`)
```
|                             [lemon icon]                               |
|                          Lemonade Tycoon                               |
|      Buy low, sell high, and grow your stand into a factory.           |
|                   [ Play game (primary) ]  [ Sign in ]                 |
```

**Sign in** (empty name → inline "Enter a username"; login creates or resumes, then `/game`)
```
|                    Username                                            |
|                    [ lemonjoe                    ]                     |
|                    [ Continue (primary)          ]                     |
|                    New name? We'll start a game for it.                |
```

**Game page**
```
| Day         Capital        Upkeep per day                              |
| 4           $1,240         $30                        [ End day -> ]   |
+------------------------------------------------------------------------+
| (sun) Heat wave: lemonade x1.4 and ice x1.3 for 2 more days            |  events banner
+------------------------------------------------------------------------+
| Market and inventory                                                   |
| Resource  Stock         Price        Buy at ask       Sell at bid      |
| Lemon     [####------]  $20 ^ ~~~    [1] [Buy $22]    [1] [Sell $18]   |
|           4 / 10                                                       |
| Lemonade  [##########]  $140 ^ ~~~   [1] [Buy $154]   [1] [Sell $126]  |
|           10 / 10            (sugar, ice, cups rows follow the same)   |
+------------------------------------------------------------------------+
| Facilities                                                             |
| +---------------------------------------------+ +--------------------+ |
| | [img] Pantry                                | | [img] Kitchen      | |
| |       Warehouses, level 1 of 4              | |       Production,  | |
| |       Upkeep $10 per day                    | |       level 1 of 4 | |
| | [ Upgrade all warehouses to Garage: $500 ]  | | 1 building         | |
| | Covers 5 buildings at $100 each. Each       | | Makes 10 lemonade  | |
| | building goes from 10 to 20 cases.          | | per day            | |
| |---------------------------------------------| | Upkeep $20 per day | |
| | Lemon     x1  Holds 10 cases  [Expand $100] | | [ Expand: $500 ]   | |
| | Sugar / Ice / Cups / Lemonade rows the same | | [ Upgrade to Food  | |
| +---------------------------------------------+ |   Truck: $1,000 ]  | |
+------------------------------------------------------------------------+
```
Only "End day" is primary. Stock bar = stock / (`count × size`). Price cell = effective price, trend arrow vs. yesterday, 14-day sparkline (stretch). Buy shows the ask and Sell the bid. One Upgrade button covers a whole type and reads "Max level" at level 4; Expand is per warehouse resource, and Production has its own. Buy/Sell/Expand/Upgrade stay enabled and show the server error (except offline).

**Day report** (modal; built from the end-day `DayReport`; dismiss returns to the game)
```
| Day 4 report                       |
| Lemonade produced        +10       |
| Ice melted               2 cases   |
| Upkeep paid              -$30      |
| Lemonade price           $100 -> $140
| New event                Heat wave |
| Capital           $1,240 -> $1,210 |
| [ Start day 5 (primary) ]          |
```
Shows a warning when capital ends at $0 but the game continues.

**Game over** (`New game` → `POST /api/game/new`)
```
|                          [sad face icon]                               |
|                       Bankrupt on day 12                               |
|         You ran out of cash and stock. Final capital: $0.              |
|                       [ New game (primary) ]                           |
```

**Component map:** `nav-bar` (sign in/out) · `stats-strip` (day, capital, upkeep; end day) · `events-banner` · `market-panel` (buy/sell) · `facilities-panel` (expandWarehouse, expandProduction, upgrade) · `day-report-modal` · `game-over` · `offline-banner`. Open UX: signed-in users are not auto-redirected from `/`; mobile stacks each market row as a card.
