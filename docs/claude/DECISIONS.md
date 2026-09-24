# DECISIONS

Deviations from DESIGN.md and real tradeoffs made during the build. Newest last.

## 1. Keep the scaffold's `main.go` at the module root (slice 0)
- **Design said:** `cmd/server/` for the entrypoint; `internal/domain`, `internal/store`, `internal/api` for layers.
- **Did:** `main.go` stays at `lemonade-api/` root. New code goes in `internal/` as designed (`/healthz` is in `internal/api`). The scaffold's `controller/ service/ repository/ model/` sample code stays until the game's own layers replace it.
- **Why:** the Dockerfile (`go build .`), Cloud Build and deploy scripts all build the module root. Moving it costs a deploy-pipeline change for no behavior gain in a 4h box.
- **Trade-off:** mixed layout for a few slices (scaffold packages next to `internal/`). Removing the sample resource is a cleanup, not part of any slice.

## 2. `/healthz` is liveness only (slice 0)
- **Did:** `GET /healthz` returns `200 {"status":"ok"}` without pinging the database.
- **Why:** the server exits at startup if it cannot connect to Postgres, so a running process implies a DB connection was made. Simple, and it can't flap on a slow query.
- **Trade-off:** does not detect a DB that drops after startup. A readiness check with `db.Ping()` is a one-line change if needed.

## 3. Frontend built ahead of the backend; `api.models.ts` is the contract
- **Did:** at the user's request, built the whole frontend structure (routes, store, API client, every screen from the UX mocks) before the game backend exists. `lemonade-web/src/app/core/api.models.ts` defines every request and response shape; the Go DTOs must match it.
- **Why:** backend and deploy work can proceed in parallel; the UI is verified by component tests with fixtures (`core/testing/fixtures.ts`).
- **Trade-off:** the contract may shift when the domain is implemented; any change is made in `api.models.ts` first so the compiler flags every affected component.

## 4. API JSON uses camelCase; the view carries display-ready values
- **Did:** fields are camelCase (`previousPrice`, `upkeepPerDay`), not the scaffold's snake_case. The game view includes derived values the UI shows (tier names, bid/ask, capacities, expand and upgrade costs, `upgrade: null` at max level) so the frontend never recomputes rules.
- **Why:** matches TypeScript naming with no mapping layer, and keeps the "server is the source of truth" rule strict.
- **Trade-off:** Go structs need explicit `json:"camelCase"` tags; a slightly larger payload.

## 5. UI error and busy behavior
- **Did:** a failed action keeps the current view and shows the server's `message` in a dismissible alert (the next successful call clears it). Buy/Sell/Expand stay enabled per the mocks; Upgrade is disabled only at max level ("Max level"). End day is disabled while any request is in flight, so the day cannot end mid-trade.
- **Trade-off:** no client-side validation of qty, funds, or capacity; the server is the single place those rules live.

## 6. PWA: installability checked via Chrome, not Lighthouse (slice 7)
- **Design said:** verify installability with Lighthouse.
- **Did:** Lighthouse 12 removed its PWA category, so installability is checked with Chrome's own `Page.getInstallabilityErrors` (the check behind the Install button; also visible in DevTools → Application → Manifest). Locally: no errors, service worker controls the page, zero `/api` entries in Cache Storage, offline reload shows the shell and the offline banner, and offline API calls get the service worker's 504 (never a cached response).
- **Also:** `ngsw-config.json` adds `!/api/**` to `navigationUrls`, so the service worker never answers an address-bar visit to an API URL with the app shell. No `dataGroups`, so no API caching.
- **Icons:** the Angular template icons stay until real assets arrive (user request). Theme color is lemon (`#facc15`).

## 3. One shared level for all warehouses; quantity is per resource (slices 1, 5)
- **Design said:** six independent facility kinds (`warehouse_lemon` ... `production`), each with its own level and quantity, and `:kind` in the facility routes.
- **Did:** two facility types. All five warehouses share one level; each resource's warehouse has its own building count. Routes are `/facilities/warehouse/expand` (body `{resource}`), `/facilities/production/expand`, and `/facilities/{warehouse|production}/upgrade`. Warehouse upgrade costs `upgradeCost(level) x total warehouse buildings`. Max quantity (10) is per resource warehouse, and for production.
- **Why:** the frontend contract (`api.models.ts`, `api.service.ts`) was built that way and says the backend must return exactly those shapes. The API follows the contract, not the older DESIGN table.
- **Trade-off:** you cannot have a Barn for lemons and a Pantry for sugar. Upgrading gets more expensive as you expand warehouses.

## 4. `EndDay` returns an error, and events never re-spawn while active (slices 3, 6)
- **Did:** `EndDay(g, cfg) (DayReport, error)` returns `ErrGameOver` on a finished game (DESIGN had no error). Spawning skips any event already active, and expiry runs before spawning, so an event can start again the same tick it ends.
- **Why:** rule 11 rejects all actions after game over, and ending a day is an action. Skipping active events stops an event stacking on itself, which would make one shock unbounded.
- **Trade-off:** at most one copy of each event is active at once.

## 5. Quote rounding snaps float noise (slice 2)
- **Did:** bid/ask floor/ceil operate on `round(x * 1e6) / 1e6`.
- **Why:** `100 * 1.1 = 110.00000000000001`, so a plain `ceil` gave an ask of $111 instead of $110. Caught by the rounding test.

## 6. Seed comes from the clock in the API layer; persistence is one game row per user (slice 1)
- **Did:** `api.NewGame(..., nil)` seeds new games from `time.Now().UnixNano()`; the domain stays clock-free and takes the seed as input. `games` has a unique `user_id`, so "new game" overwrites the row instead of keeping history.
- **Why:** rule 2 says one active game per user and a new game replaces it. Games stay reproducible from the stored seed.
- **Trade-off:** no history of past games (a leaderboard would need a separate table).

## 7. Proxies pass `/api` through unchanged (frontend/backend integration)
- **Found:** the scaffold's nginx and dev-server proxies stripped the `/api` prefix, but the game API serves its routes under `/api` (DESIGN §5), so every proxied call returned 404.
- **Did:** removed the rewrite from `nginx.conf.template` and `pathRewrite` from `proxy.conf.json`. `/healthz` is no longer reachable through the web proxy; check it on the API port. The scaffold's sample routes (`/samples`) are no longer proxied either.
- **Also:** a 401 (stored username unknown to the server, e.g. after a DB reset) now clears the session and routes to `/signin` (`unauthorized.interceptor.ts`).
- **Verified:** every response from a full flow (login, game, buy, sell, both expands, upgrade, end-day with a live event, six error cases) type-checks against `api.models.ts`; a headless-Chrome run of the UI against the real API and Postgres passes.

## 8. Warehouse upgrade costs follow the addendum, not the first DESIGN draft
- **Found:** the backend charged $200/$500/$1,200 per building (DESIGN §6, first draft). `UX-MOCKS-AND-CHANGES.md` §1.3, which wins on conflicts, sets $100/$250/$600, and its example says a fresh game's 5 Pantries upgrade to 5 Garages for $500. The backend offered $1,000, which left a new player at $0 after their first upgrade.
- **Did:** changed `WarehouseTiers` in `internal/domain/config.go`; added a cost-table test and the fresh-game example as tests; corrected the stale row in DESIGN §6. Production upgrade costs were already right.
- **Audited:** end-of-day order, production min(), upkeep clamp, bankruptcy, bid/ask rounding, price walk, event stacking, and all other tier values match the addendum.

## 9. Conflicting events: `Excludes` on the event table
- **Rule:** a heat wave and a rainy week can never be active on the same day.
- **Did:** `EventDef` gets an `Excludes []string` of event keys. An event is only eligible to spawn if it is not already active and neither it nor any active event excludes the other; the check works in both directions, so a pair is declared once (on `heat_wave`). Adding another conflict is one more table entry.
- **Why:** keeps to the table-driven design (DESIGN §6) instead of hard-coding weather logic in `tickEvents`.
- **Trade-off:** a 2-day heat wave also blocks a rainy week from starting until it ends (and vice versa), so a spawn roll during that time picks from the other events. Events already active in saved games are unaffected.
- **Tests:** eligibility in both directions, the default config's pair, and a 20-seed × 300-day simulation with an event every day that fails without the rule (verified by removing it).

## 10. Deployed PWA check, nginx headers, and the lemon favicon (slice 7)
- **Verified on the live HTTPS site** (headless Chrome, fresh profile): service worker controls the page; `Page.getInstallabilityErrors` is empty; Cache Storage holds no `/api` entries; offline, the shell loads with the banner and API calls fail with the worker's 504; with a game loaded, all 19 action buttons disable when the network drops and re-enable when it returns.
- **Fixed in `nginx.conf.template`:** the manifest was served as `application/octet-stream` (now `application/manifest+json`), and no file had a `Cache-Control`, so browsers could heuristically cache `ngsw.json` and delay PWA updates. Everything outside `/api` is now `no-cache` (revalidate, cheap 304s); built JS/CSS are content-hashed so this costs little.
- **Favicon:** the whole lemon from `assets/resources/lemon.svg`, cropped to fill the tab, as `favicon.svg` plus a 16/32/48px `favicon.ico`. The previous lemon-slice-on-a-tile icon was unreadable at 16px. PWA install icons are unchanged.
- **Cloud Run reserves `/healthz`:** on the deployed API, `GET /healthz` returns Google's own 404 page, never reaching the app, so the slice 0 acceptance "healthz returns 200 deployed" cannot be met with that path. Locally it works. Needs a different path (for example `/api/health`) if a deployed health check matters.
