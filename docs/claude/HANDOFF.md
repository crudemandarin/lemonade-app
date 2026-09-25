# Handoff: Lemonade Tycoon, state of the project and what a planner needs to know

Written at the end of a long build session so a fresh agent can plan a major update without re-deriving anything. It is a snapshot: **check `git log` and `git status` before trusting any detail here**, because other agents have been editing the same working tree and may have moved things since.

The user has not yet said what the major update is. **Ask them for the goals first**, then plan (see [section 10](#10-questions-to-ask-the-user-before-planning)).

---

## 1. What this is

A turn-based lemonade-business game. Buy ingredients, run facilities, sell lemonade, survive the market one "day" at a time. Score is capital. It began as a **time-boxed (4h) full-stack take-home** with a walkthrough at the end (`docs/DEMO.md` has the demo script), so the codebase favors clear abstractions, tests, and being able to explain every decision.

- **Live:** https://lemonade.nyko.run (Cloud Run). GCP project `lemonade-app-509618`, region `us-central1`.
- **Status:** the whole original plan is built (slices 0 to 8 in `PLAN.md` are ticked), deployed, and the user has verified PWA install works on their computer.

## 2. Read these first (in this order)

| File | Why |
|---|---|
| `docs/claude/CLAUDE.md` | The working rules for any agent here (see section 8). |
| `docs/claude/SPEC.md` + `docs/claude/UX-MOCKS-AND-CHANGES.md` | Behavior rules. **The addendum wins where it conflicts with SPEC/DESIGN.** Some rules there were later revised (each revision points to a DECISIONS number). |
| `docs/claude/DECISIONS.md` (20 entries) | Every deviation and trade-off, with why. The best single source for "why is it like this?". |
| `docs/numeric-tdd.md` | Technical design doc with an architecture diagram and the balance notes. |
| `docs/claude/DESIGN.md` | Original design. Partly stale (numbers, bankruptcy rule); DECISIONS is authoritative. |
| `docs/claude/PLAN.md` | Slices and the **Future work** list (a ready source of update ideas). |
| `docs/claude/EVENT-BACKDROPS.md` | Design of the animated event backdrops (another agent's feature). |
| `docs/claude/ai-usage-log.md` | 11+ entries on how AI was used; useful if the update is also for a walkthrough. |
| `README.md`, `lemonade-api/README.md`, `lemonade-web/README.md` | Run/test/tune instructions; the root README's "Tuning the game" section documents every balance knob. |

## 3. Stack and repo map

- **Backend:** Go 1.27, Gin, GORM on PostgreSQL 16. **Frontend:** Angular 17 (standalone components, signals), plain SVG charts with no chart library, installable PWA. **Infra:** Terraform (Cloud Run for `web` and `api`, Cloud SQL, Artifact Registry, Cloud Build), deploy via `deploy/scripts/*.sh` and a GitHub Action that redeploys on pushes to `main` touching `lemonade-api/**` or `lemonade-web/**`. Local: `docker compose up -d --build` (web :4200, api :8080, db :5432).
- **Server is the source of truth.** The frontend renders server state and never computes game outcomes.

```
lemonade-api/
  main.go                 wiring only: secrets, DB, migrate, routes
  libraries/              secrets loading, DB connection
  internal/domain/        PURE game rules (no I/O, no Gin, no SQL)
    config.go             every balance knob in one struct: DefaultConfig()
    actions.go            Buy, Sell, Expand, Upgrade
    endday.go             EndDay: produce, melt ice, settle upkeep, advance, market tick, events
    bankruptcy.go         settleUpkeep + forced sales (insolvency rule)
    market.go quotes.go   price walk; bid/ask (whole dollars)
    events.go             event table, spawn/expiry, Excludes (conflicts)
    facilities.go         capacity, production rate, upkeep
    timeline.go           history recording + Stats (feeds the charts and report)
    *_test.go             domain tests; balance_test.go = simulated-player guard rails
  internal/store/         Repository interface, Postgres impl (JSONB), in-memory fake
  internal/api/           Gin handlers, DTOs (camelCase), error mapping, username middleware

lemonade-web/src/app/
  core/                   api.models.ts (THE contract), api.service, game.store (signals),
                          session/online services, interceptors (X-Username, 401), auth guard,
                          testing/fixtures.ts
  shared/                 nav-bar, card, icon (SVG mask icons), money pipe, offline-banner,
                          price-sparkline, timeline-charts (two-chart component + pure layout module),
                          event-backdrop (CSS scenes)
  pages/                  home, signin, game (dashboard + game-over report; components/ inside)
lemonade-web/src/assets/  SVG icons, resources, facilities (from another agent)
lemonade-web/design-preview/  standalone event-backdrop preview (ignored by lint/prettier)
deploy/                   Terraform + scripts
```

## 4. Game rules as they stand now

**Resources:** lemon, sugar, ice, cup (inputs) and lemonade (product), per-case whole-dollar prices. Recipe: 1 lemon + 1 sugar + 1 ice + 1 cup makes 1 lemonade.

**Facilities:** two types, each with **one shared level (1 to 4)**. *Warehouse* (a separate building count per resource; capacity = count × size) and *Production* (rate = buildings × per-building rate). Max 10 buildings per warehouse resource and for production. Expand adds one building at the current level; Upgrade raises the whole type one level for `per-building cost × all buildings of that type`.

**A day:** the player acts freely (buy at ask, sell at bid, expand, upgrade), then **End day**, one atomic step in this order:
1. Produce `min(production rate, stock of each input, free lemonade space)`.
2. **Ice melts to 0** (the only perishable; buy it the day you want it used).
3. Settle upkeep (always owed): short cash sells stock at the current bid (lemonade first, then lemon, sugar, cup; only as many cases as needed). If even everything sold cannot cover it, the game ends **on that day** (day does not advance).
4. Advance the day, tick the market, start/expire events.

**Current numbers** (all in `DefaultConfig()`; README documents each): start $1,000; base prices lemon $20, sugar $10, ice $10, cup $10, **lemonade $90**; 10% spread (ask = ceil(p×1.1), bid = max(1, floor(p×0.9))); price walk `p' = p + 0.15(base-p) + p·0.12·N(0,1)` clamped to 0.25×–4× base, seeded by `seed ^ day`; 25% daily event chance; warehouse tiers Pantry/Garage/Barn/Industrial (size 10/20/40/80, build $100/$300/$800/$2,000, upgrade $100/$250/$600, upkeep $2/$6/$16/$40); production tiers Kitchen/Food Truck/Bottling Plant/Lemonade Factory (rate 10/20/40/80, build $500/$1,500/$4,000/$10,000, upgrade $1,000/$2,500/$6,000, upkeep $20/$50/$120/$280). A new game costs $30/day upkeep.

**Events** (table-driven; one row to add): Heat Wave (lemonade ×1.4, ice ×1.3, 2d), Rainy Week (lemonade ×0.75, 3d), Lemon Blight (lemon ×1.7, 3d), Sugar Glut (sugar ×0.7, 2d), Holiday (lemonade ×1.35, 1d), Cup Shortage (cup ×1.5, 2d). Multipliers stack on the effective price only, never on the walked price. **Heat Wave and Rainy Week can never overlap** (`Excludes`, decision 13).

**Bankruptcy:** insolvency as above, not "capital 0 and no stock" (that original rule let $0-capital players hold one case and never lose; decisions 16, 18).

**Usernames:** username-only login, sent as `X-Username` on every request, 3 to 40 ASCII chars, case-insensitive (decision 19). Intentionally not secure; documented.

## 5. API (all JSON, camelCase; every mutation returns the updated game view)

| Method and path | Purpose |
|---|---|
| `GET /api/health` | Liveness (**not** `/healthz`: Cloud Run reserves that exact path, decision 15) |
| `POST /api/login` `{username}` | Create-or-get a user (new user gets a new game) |
| `GET /api/game` | Game view (day, capital, status, upkeep, per-resource stock/capacity/price/bid/ask/history, facilities with costs and upgrade options, events, `timeline`, `stats`) |
| `POST /api/game/new` | Fresh game |
| `POST /api/game/buy`, `/sell` `{resource, qty}` | Trade |
| `POST /api/game/facilities/warehouse/expand` `{resource}`, `.../production/expand`, `.../:type/upgrade` | Facilities |
| `POST /api/game/end-day` | Returns `{report, game}` |

Errors are `{error: "<code>", message}` with sensible statuses (409 for rule violations, 401 unknown user, 400 bad input). **`lemonade-web/src/app/core/api.models.ts` is the contract**; a Go DTO change must be mirrored there (a past check compiled real responses against it with `tsc --strict`; that technique is worth reusing).

**Persistence:** one game row per user; scalars as columns, the rest as JSONB (`inventory`, `warehouseQty`, `market`, `events`, `timeline`, `stats`). Every mutation is `load, domain call, save` in one transaction with `SELECT ... FOR UPDATE`. `AutoMigrate` adds columns; a game saved before a new field existed loads with the zero value (the API synthesizes a start point for an empty timeline). **Plan schema changes with old saved games in mind.**

## 6. History, charts, and the report (built last, most recent design)

- The domain records a **timeline** (capital and stock after each buy/sell/expand/upgrade/end-of-day) and running **stats** (peak capital and day, cases traded, spend/earnings, facilities built, upkeep paid, lemonade produced). Repeated clicks on the same trade in a day merge into one point; only the last 7 days keep every trade; a hard cap of 400 points; stats are never compacted.
- Frontend: price **sparklines** in each market row; a **two-chart timeline** (capital step line with marker dots: hollow circle = buy, filled = sell, square = built, diamond = upgraded; and a per-resource stock chart), shown on the game page always and full-size on the bankruptcy report. Deliberately **two charts, not one dual-axis chart**. Resource colors are fixed per resource and validated with the dataviz palette validator for light and dark. The charts have a legend (toggle a resource), a shared hover crosshair, keyboard navigation, touch scrubbing, and a "view as table" twin.
- Interpretation note: the user asked for "resource capacities (one line per resource)"; I built **stock held** per resource, not warehouse capacity. Confirm if that matters.

## 7. How to run, test, and verify

```bash
# full stack
docker compose up -d --build          # web http://localhost:4200, api :8080
curl localhost:8080/api/health

# backend (from lemonade-api/)
gofmt -l . ; go vet ./... ; go test ./...
DATABASE_URL="postgres://admin:password@localhost:5432/sample?sslmode=disable" go test ./internal/store   # real-Postgres round trip
BALANCE_REPORT=1 go test ./internal/domain -run TestBalanceReport -v                                      # balance numbers

# frontend (from lemonade-web/)
npx ng test --watch=false --browsers=ChromeHeadless   # 132+ specs at last count of mine; others added more
npm run lint ; npm run format:check ; npm run build
```

Techniques that worked (the scripts lived in a scratch directory and are **not** in the repo, so recreate as needed):
- **Real-stack check beats mocks.** Drive the API with a script, then a headless-Chrome (CDP over a websocket) script for UI flows and screenshots at 1280 and 390 px, including a horizontal-overflow check.
- A throwaway mock API plus `ng serve --proxy-config` lets the UI be checked with no backend.
- **PWA needs a production build** (the service worker is off in `ng serve`). Compose's `web` container serves the production build.

## 8. Rules of engagement for this repo

From `CLAUDE.md` plus what the user has shown they want:
- **Test first**: write failing tests for the acceptance criteria, implement, run everything, commit. Domain logic stays pure; UI/API layers stay thin; inject anything nondeterministic.
- **No scope creep**: features outside SPEC go to PLAN's Future work. **If a requirement is ambiguous, ask; do not silently guess.** Log real deviations in `DECISIONS.md` (append; the numbering was already renumbered once).
- **Shared working tree with other agents.** Others edit and commit here concurrently. **Never `git add -A`**; stage explicit paths, commit only your own files, and leave other agents' uncommitted work alone. At the time of writing, `deploy/**` and `docs/DEMO.md` had uncommitted edits that were not mine.
- The user prefers: short status updates when asked ("hurry up. status check"); to finish the current task before switching; to be asked for an opinion on design ("what do you think?") and to confirm before a big design goes into the UI (backdrops were reviewed in a standalone preview before integration); mobile-friendly everything; consistent visual language; commit messages ending with the attribution line the harness specifies.
- Keep `docs/claude/ai-usage-log.md` current: flag "Log candidate:" moments and append entries on request (target 5 to 10 strong ones; it is already at 11).
- Only one primary button per screen. Copy is sentence case with no terminal punctuation on labels. Money is whole dollars everywhere.

## 9. Gotchas and known limits

- **Docker compose keeps getting torn down** by other sessions. If the API logs "no such host db", run `docker compose up -d` (the volume survives; nothing is lost).
- **Cloud Run reserves `/healthz`**; that is why health is `/api/health`.
- nginx and the dev proxy pass `/api` through **unchanged**; the scaffold used to strip it (decision 11). Changing the API prefix means changing both proxies.
- **Angular 17 limits:** no `@let`; `as` aliasing only on the primary `@if` branch; use `afterNextRender` (not `ngAfterViewInit`) to measure element width. ESLint and Prettier deliberately ignore `design-preview/` and `shared/event-backdrop/`.
- Every mutation returns the **whole timeline** (about 13 KB typical, 50 KB worst); accepted. The 400-point cap eventually drops the oldest milestones in a very long game (about day 350+).
- **Balance is guarded by tests with bands, not exact numbers** (`TestBalance*`). Changing prices, costs, or upkeep also changes a few exact figures asserted in the API/domain tests and in `README`/`DESIGN`/`SPEC`; update them together.
- **Auth is intentionally weak** (anyone can type anyone's username). It is documented as a known limitation.
- The stock "capacity" wording, the "three facility types" question from the original brief (modeled as two: Warehouse and Production), and the grace-rule alternatives are the original open questions; see `SPEC.md` section 6.
- Not verified by me: whether the very latest commits are pushed and deployed, and a Lighthouse-style score (Lighthouse 12 removed its PWA audit; Chrome's own installability check was used instead, decision 14).

## 10. Questions to ask the user before planning

1. **What is the "major update"?** New gameplay, a new system (accounts, multiplayer, leaderboard), scale/infra, a redesign, or polish for the walkthrough?
2. Is this still the **take-home** (deadline, walkthrough) or a longer-running project? That changes how much process and documentation to keep.
3. Should **old saved games keep working** (migration) or is a reset acceptable?
4. Any constraint on the stack (stay on Angular 17, keep Go, keep Cloud Run)?
5. Is difficulty and pacing now where they want it? (They asked for "too hard to lose" to be fixed earlier and it was; they may want to retune.)

## 11. Candidate directions (ideas only, none requested)

From `PLAN.md` Future work and from things noticed along the way. Treat these as a menu to discuss, not a backlog:
- **Gameplay depth:** player-chosen liquidation instead of silent forced sales; a grace day or partial-payment ledger; market depth and price impact from the player's own trades; demand simulation with player-set price; event forecasting; loans and interest; contracts or bulk discounts; mixed-tier buildings, selling or downgrading; more facility types (fridge, marketing sign) and specialty warehouses; a user-tweakable recipe.
- **Social/product:** leaderboard (capital and day are already real columns), real authentication, replay of a finished game (the timeline plus the seed make this feasible).
- **Engineering:** end-to-end browser tests and a CI pipeline (only the deploy workflow exists); an optional history endpoint instead of returning the timeline on every mutation; PWA extras (update prompt, read-only offline view, push).
- **Presentation:** a demo mode or seeded scenarios for showing the game quickly.

## 12. How the project got here (chronological, with commits)

`9f2e47a` scaffold. `84cd603` slice 0: `/healthz` and docs commands. `e534a96` sample frontend removed and the game UI built from the UX mocks against a typed API client (before the backend existed). `7737f19` PWA plus SVG assets. `dc542b8` frontend connected to the real backend (found the `/api` proxy bug; added the 401 handler). `79bc21c` backend committed, warehouse upgrade costs corrected to the addendum, sample API removed. `d99d3e7` heat wave and rainy week made mutually exclusive. `4666848` lemon favicon; manifest content type and no-cache headers. `c7c3f79` health moved to `/api/health`; lint ignores. `278c9b1`/`d6418d7` event backdrops (another agent). `a211562` mobile-friendly frontend. `497956a` balance pass by simulation (lemonade $90, more volatile, upkeep doubled, forced-sale insolvency rule) plus README physics section. `825d345` username rules. `a13a61c` sparklines and the two-chart timeline with stats and the bankruptcy report. Later commits (`e9706e7` game ends on the day it is lost; `70e4851` more specs; `13b1ecb`/`d54fadc` docs and architecture diagram; `efc6eb9` demo) were made by other sessions.

The AI usage log (`docs/claude/ai-usage-log.md`) records the notable techniques: simulated players to tune balance, compiling real responses against the frontend contract, choosing two charts over a dual axis and validating chart colors by script, plus an independent reviewer pass by another agent.
