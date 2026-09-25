# DECISIONS

Deviations from DESIGN.md and real tradeoffs made during the build. Newest last.

## 1. Keep the scaffold's `main.go` at the module root (slice 0)
- **Design said:** `cmd/server/` for the entrypoint; `internal/domain`, `internal/store`, `internal/api` for layers.
- **Did:** `main.go` stays at `lemonade-api/` root. New code goes in `internal/` as designed (the health check lives in `internal/api`).
- **Why:** the Dockerfile (`go build .`), Cloud Build and deploy scripts all build the module root. Moving it costs a deploy-pipeline change for no behavior gain in a 4h box.
- **Trade-off:** the entrypoint sits at the module root instead of `cmd/server/`. The scaffold's sample `controller/ service/ repository/ model/` code has since been removed.

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

## 7. One shared level for all warehouses; quantity is per resource (slices 1, 5)
- **Design said:** six independent facility kinds (`warehouse_lemon` ... `production`), each with its own level and quantity, and `:kind` in the facility routes.
- **Did:** two facility types. All five warehouses share one level; each resource's warehouse has its own building count. Routes are `/facilities/warehouse/expand` (body `{resource}`), `/facilities/production/expand`, and `/facilities/{warehouse|production}/upgrade`. Warehouse upgrade costs `upgradeCost(level) x total warehouse buildings`. Max quantity (10) is per resource warehouse, and for production.
- **Why:** the frontend contract (`api.models.ts`, `api.service.ts`) was built that way and says the backend must return exactly those shapes. The API follows the contract, not the older DESIGN table.
- **Trade-off:** you cannot have a Barn for lemons and a Pantry for sugar. Upgrading gets more expensive as you expand warehouses.

## 8. `EndDay` returns an error, and events never re-spawn while active (slices 3, 6)
- **Did:** `EndDay(g, cfg) (DayReport, error)` returns `ErrGameOver` on a finished game (DESIGN had no error). Spawning skips any event already active, and expiry runs before spawning, so an event can start again the same tick it ends.
- **Why:** rule 11 rejects all actions after game over, and ending a day is an action. Skipping active events stops an event stacking on itself, which would make one shock unbounded.
- **Trade-off:** at most one copy of each event is active at once.

## 9. Quote rounding snaps float noise (slice 2)
- **Did:** bid/ask floor/ceil operate on `round(x * 1e6) / 1e6`.
- **Why:** `100 * 1.1 = 110.00000000000001`, so a plain `ceil` gave an ask of $111 instead of $110. Caught by the rounding test.

## 10. Seed comes from the clock in the API layer; persistence is one game row per user (slice 1)
- **Did:** `api.NewGame(..., nil)` seeds new games from `time.Now().UnixNano()`; the domain stays clock-free and takes the seed as input. `games` has a unique `user_id`, so "new game" overwrites the row instead of keeping history.
- **Why:** rule 2 says one active game per user and a new game replaces it. Games stay reproducible from the stored seed.
- **Trade-off:** no history of past games (a leaderboard would need a separate table).

## 11. Proxies pass `/api` through unchanged (frontend/backend integration)
- **Found:** the scaffold's nginx and dev-server proxies stripped the `/api` prefix, but the game API serves its routes under `/api` (DESIGN §5), so every proxied call returned 404.
- **Did:** removed the rewrite from `nginx.conf.template` and `pathRewrite` from `proxy.conf.json`. `/healthz` is no longer reachable through the web proxy; check it on the API port. The scaffold's sample routes (`/samples`) are no longer proxied either.
- **Also:** a 401 (stored username unknown to the server, e.g. after a DB reset) now clears the session and routes to `/signin` (`unauthorized.interceptor.ts`).
- **Verified:** every response from a full flow (login, game, buy, sell, both expands, upgrade, end-day with a live event, six error cases) type-checks against `api.models.ts`; a headless-Chrome run of the UI against the real API and Postgres passes.

## 12. Warehouse upgrade costs follow the addendum, not the first DESIGN draft
- **Found:** the backend charged $200/$500/$1,200 per building (DESIGN §6, first draft). `UX-MOCKS-AND-CHANGES.md` §1.3, which wins on conflicts, sets $100/$250/$600, and its example says a fresh game's 5 Pantries upgrade to 5 Garages for $500. The backend offered $1,000, which left a new player at $0 after their first upgrade.
- **Did:** changed `WarehouseTiers` in `internal/domain/config.go`; added a cost-table test and the fresh-game example as tests; corrected the stale row in DESIGN §6. Production upgrade costs were already right.
- **Audited:** end-of-day order, production min(), upkeep clamp, bankruptcy, bid/ask rounding, price walk, event stacking, and all other tier values match the addendum.

## 13. Conflicting events: `Excludes` on the event table
- **Rule:** a heat wave and a rainy week can never be active on the same day.
- **Did:** `EventDef` gets an `Excludes []string` of event keys. An event is only eligible to spawn if it is not already active and neither it nor any active event excludes the other; the check works in both directions, so a pair is declared once (on `heat_wave`). Adding another conflict is one more table entry.
- **Why:** keeps to the table-driven design (DESIGN §6) instead of hard-coding weather logic in `tickEvents`.
- **Trade-off:** a 2-day heat wave also blocks a rainy week from starting until it ends (and vice versa), so a spawn roll during that time picks from the other events. Events already active in saved games are unaffected.
- **Tests:** eligibility in both directions, the default config's pair, and a 20-seed × 300-day simulation with an event every day that fails without the rule (verified by removing it).

## 14. Deployed PWA check, nginx headers, and the lemon favicon (slice 7)
- **Verified on the live HTTPS site** (headless Chrome, fresh profile): service worker controls the page; `Page.getInstallabilityErrors` is empty; Cache Storage holds no `/api` entries; offline, the shell loads with the banner and API calls fail with the worker's 504; with a game loaded, all 19 action buttons disable when the network drops and re-enable when it returns.
- **Fixed in `nginx.conf.template`:** the manifest was served as `application/octet-stream` (now `application/manifest+json`), and no file had a `Cache-Control`, so browsers could heuristically cache `ngsw.json` and delay PWA updates. Everything outside `/api` is now `no-cache` (revalidate, cheap 304s); built JS/CSS are content-hashed so this costs little.
- **Favicon:** the whole lemon from `assets/resources/lemon.svg`, cropped to fill the tab, as `favicon.svg` plus a 16/32/48px `favicon.ico`. The previous lemon-slice-on-a-tile icon was unreadable at 16px. PWA install icons are unchanged.
- **Cloud Run reserves `/healthz`:** on the deployed API, `GET /healthz` returns Google's own 404 page, never reaching the app, so the slice 0 acceptance "healthz returns 200 deployed" cannot be met with that path. Locally it works. Needs a different path (for example `/api/health`) if a deployed health check matters.

## 15. Health check moved from `/healthz` to `/api/health`
- **Why:** Cloud Run's front end reserves the exact path `/healthz` and answers it with Google's own 404, so the deployed API could never pass the slice 0 check. `/api/health` also goes through the web proxy, so one public URL (`<web-url>/api/health`) checks both services.
- **Did:** replaced the route (no alias kept) and updated the test, READMEs, CLAUDE.md, DESIGN §5 and PLAN. Decisions 1, 2, 11 and 14 still say `/healthz` because they record what was true then. It stays liveness-only (decision 2).
- **Also:** lint and Prettier now skip `design-preview/` (throwaway design mock-ups) and `shared/event-backdrop/`, which had two `no-explicit-any` errors in its spec.

## 16. Balance pass: harder to coast, and upkeep can actually bankrupt you
- **Problem (measured):** simulated players showed the game was too easy to lose *and* too easy to win. Producing was unprofitable on 1.4% of days (the market was decoration), a careful player was never at risk, and 54-69% of careless players ended in a "zombie" state: $0 capital but holding some stock, so never bankrupt. Cause of the zombies: unpaid upkeep was forgiven at $0, and any inventory counted as a grace forever. Doubling or quintupling upkeep alone changed nothing for careful players.
- **Did (numbers):** lemonade base price $100 to $90; price volatility `Sigma` 0.08 to 0.12 and `RevertRate` 0.2 to 0.15; every upkeep value doubled (a new game costs $30/day, was $15). Kept: starting capital, spread, all build and upgrade costs, events (plus the heat-wave/rainy-week exclusion, decision 13).
- **Did (rule):** upkeep is always owed. If cash is short, stock is sold at the current bid, lemonade first, just enough to cover it; if everything sold still is not enough, the game is over. This replaces both "unpaid upkeep is forgiven" and "capital 0 and no stock". The day report gets `forcedSaleCases` and `forcedSaleProceeds`, and the UI shows a "Stock sold to cover upkeep" line. Spending to $0 mid-day is still allowed.
- **Result** (seeded simulation, 80-200 games each): careful player about 1 in 8 bankrupt within 90 days; sloppy (forgets ice 8% of days, overspends 6%) about 2 in 3 within 45 days; careless all bankrupt by about day 10; idle player bankrupt about day 34; no zombies. Producing is unprofitable on about 11% of days (average margin $22 per lemonade, was $33). Pacing: level 1 full about day 28, everything maxed about day 65 (was day 16 and day 36).
- **Alternatives considered:** upkeep x3/x4 (the careful player also started dying, 6% to 26%); lemonade at $85 or $80 (opening too slow: level 1 full by day 23 and 30); a "skip bad days" trader bot never beat always-producing, so no market-timing mechanic was added.
- **Trade-off:** harsher for real players than the original spec, and the earlier design's "leniency" trade-off (DESIGN §9 item 7) is reversed on purpose. The guard-rail tests (`TestBalance*`) keep future tuning inside these bands.
- **Also:** the balance simulator became `balance_test.go`; a few test expectations that hard-coded $100 lemonade or $15 upkeep now derive from the config or use the new values.

## 17. Forced upkeep sales use today's bid and a fixed order (decision 16 follow-up)
- **Did:** when cash is short, stock is sold at the bid in `Quotes` *before* the day's market tick, in the order lemonade, lemon, sugar, cup, ice, and only as many cases as needed (the last resource may overshoot by less than one bid).
- **Why:** upkeep is charged for the day that just ended, so it settles at that day's prices; using tomorrow's prices would let the random walk decide the outcome of a decision the player already made. Lemonade goes first because it exists only to be sold, so selling it loses the least (raw inputs still need a batch to be worth anything). Ice is last because it has already melted and is always 0.
- **Trade-off:** the player does not choose what is sold, and the sale happens silently overnight (the day report shows cases and proceeds). Player-chosen liquidation would need a new prompt and API, which is out of scope.
- **Follow-up (review item 8):** when upkeep cannot be paid, `EndDay` ends the game right after settling upkeep. The day does not advance and the market and events do not tick, so "Bankrupt on day N" is the day the player lost on and the final prices are the ones they last saw. `Expand` and `Upgrade` also return `ErrInvalidFacility` (HTTP 400, `invalid_facility_type`) for an unknown facility type instead of silently doing nothing.

## 18. Bankruptcy depends on prices, not on holding stock (decision 16 follow-up)
- **Did:** a player can be bankrupt while still holding stock, if that stock's bid value plus cash is below one day's upkeep. The game-over screen says so ("even after selling all your stock").
- **Why:** this is the point of decision 16. Under "any inventory is a grace", $0 capital plus one case of anything could never lose, which the simulations showed as 54-69% zombie games. Judging stock by what it would raise at bid is the same test the player has always had (selling at bid), applied automatically.
- **Trade-off:** harsher than the first draft, and a price drop can tip a marginal player over. The day report warns at $0 capital ("You're out of cash"), and `TestBalance*` keeps the difficulty in a known band.
- **Not done:** a grace day or partial-payment ledger; listed in Future work.

## 19. Usernames: 5 to 40 ASCII characters, case-insensitive; header lookup on every request
- **Did:** usernames are trimmed, lowercased, and must be 5 to 40 printable ASCII characters with no spaces (`normalizeUsername` in `internal/api/game.go`, applied to login and to the `X-Username` header). The frontend mirrors the rule and sends the lowercase name.
- **Why case-insensitive and ASCII:** with no password, `Joe`, `joe` and a look-alike Unicode name would otherwise be three players that are easy to confuse, and normalizing to one form removes the confusion. The 5-character minimum makes trivially guessable names like `a` or `bob` unavailable, which matters a little when the name is the only credential; 40 is the original upper bound.
- **Why a database lookup per request (not a signed session):** the brief asks for username-only login, so the header is the identity. `requireUser` resolves it to a user row on every game call, a single indexed query, which also means a name the server does not know (for example after a database reset) gets a 401 and the UI signs out (decision 11). Cheap at this scale; a session token or cache would be the first change if real authentication were added (Future work).
- **Trade-off:** anyone can play as anyone by typing their name. Known and documented in the README. Users created before this change with longer or mixed-case names cannot log in as before; a fresh database has none.


## 20. Price sparklines and the two-chart game timeline
- **Charts:** two charts on one shared time axis, never one dual-axis plot: **capital** (a neutral ink step line with a dot for every trade and facility purchase) above **stock per resource** (one line each). Capital is dollars in the thousands and stock is cases in the tens, so a shared axis would mislead. Each price row also gets a small sparkline from the 14-day history the API already sent. The same charts show on the game page all the time and, full size, on the bankruptcy report with a summary of the whole game.
- **Marker language:** buy is a hollow circle, sell a filled circle, a built facility a square, an upgrade a diamond. Shape and fill carry the meaning, so it never depends on color, and no resource hue is reused for an action.
- **Colors:** the resource line colors come from the dataviz reference palette in a fixed order, remapped so lemon is yellow, ice blue, lemonade orange (sugar magenta, cups aqua), validated with the palette script for light and dark card surfaces (neighbor color-blind separation 9.2, normal-vision 19.6). Three sit under 3:1 on white, so every chart also has a legend (click to hide a resource), a hover crosshair shared by both charts, arrow-key navigation, and a table view.
- **Data:** the backend records a snapshot (capital and stock) after every buy, sell, expansion, upgrade and end of day, and running `stats` for the report. Repeated clicks on the same trade in a day merge into one point. To bound the payload, only the last 7 days keep every trade; older days keep milestones (facility purchases, end of day), with a hard cap of 400 points. Stats are never compacted, so report totals stay exact. Stored as two JSONB columns; games saved before this get a synthetic start point.
- **Reviewed and accepted:** the 400-point cap eventually drops the oldest milestones in a very long game (roughly day 350+), and every mutation returns the whole timeline (about 13 KB typical, 50 KB worst case). Both are fine at this scope. "Peak cash" on the game-over screen counts cash only, not stock value, and says so. Games saved before the timeline existed are not supported.
- **Trade-off:** older days' individual buys and sells disappear from the chart (the capital and stock lines still show each day's result). Layout is measured, not fixed: phones get smaller margins, touch-scrub with vertical scroll preserved, and 32px legend tap targets.

## 21. Event conflicts are derived from multipliers (slice 9b)
- **Did:** two events conflict when, for any resource, one multiplier is above 1 and the other below (`conflicts` in `internal/domain/events.go`). `eligibleEvents` blocks a candidate that conflicts with any active event, in both directions, or is listed in an `Excludes`. `Excludes` stays as an explicit override.
- **Why:** the hand-written `Excludes` covered only Heat Wave and Rainy Week, so Holiday (lemonade ×1.35) could overlap Rainy Week (×0.75) and cancel it out. Deriving the rule means a new event row is checked automatically.
- **Trade-off:** Holiday and Rainy Week are now mutually exclusive, which shifts the balance slightly (the `TestBalance*` bands still pass). Games saved with overlapping events just let them expire. No random numbers are drawn differently on the no-conflict path.

## 22. The header projection shares `produceQty` with `EndDay` (slice 9c)
- **Did:** production is computed by one pure function, `produceQty` in `internal/domain/endday.go`, used by both `produce()` and `PreviewEndDay`. The game view carries `projection {lemonadeToProduce, iceToMelt, limitedBy}`, recomputed on every view. `iceToMelt` is the ice left after production has used its share.
- **Why:** the preview cannot drift from the real end of day, and a property test (500 random games) checks it against `EndDay` on the same state.
- **Choices:** on a tie the input or storage space is named, not production (it is the thing a player can act on); `limitedBy` is empty only when there is no production capacity. The UI hides the ice figure when 0.

## 23. Bulk trades: a shared amount selector and `clamp` (slice 9d)
- **Did:** the market panel has one selector (10, 50, 100, All; default 10, remembered in `localStorage`) and each row has only Buy and Sell using it. Buy and sell requests take an optional `clamp`. Clamped buy trades `min(qty, affordable at ask, free space)`; clamped sell trades `min(qty, stock)`; if that is 0 the usual error for the binding limit is returned (cash before space). "All" is sent as `qty` 1,000,000 with `clamp`, so the server does the maths. Without `clamp` nothing changes.
- **Why:** the user wanted fast bulk trading without per-row number inputs. Doing the limit in the domain keeps the UI from computing outcomes, and avoids a read-then-write race.
- **Choices:** every UI trade now sends `clamp: true`, so "Buy 100" with room for 40 buys 40 rather than erroring. Row buttons are disabled when the action is impossible (no stock, cannot afford one case, warehouse full). The response is still just the game view; the number actually traded is not returned (the change shows in stock and capital).

## 24. Selling buildings: half of build cost, quantity only (slice 10a)
- **Did:** `POST /api/game/facilities/:kind/sell` sells one building for `floor(ResaleRate × BuildCost[level])`, `ResaleRate` = 0.5 in `Config`. Levels are never sold or downgraded. The last production building and each resource's last warehouse must stay (`min_facility`), and the remaining capacity must still hold the current stock (`stock_exceeds_capacity`, "Sell N cases first"); stock is never sold silently to make room. Upkeep drops at once. The view carries `sellValue`, `canSell` and a reason per warehouse row and for production; the UI asks for confirmation.
- **Why half:** a test (`TestResaleIsAlwaysBelowWhatABuildingCost`) asserts that for every tier, resale is below the cheapest way to own that building (build it at the level, or build at level 1 and upgrade). If a config change breaks that, buying and selling buildings becomes a money machine and the test fails. Example: a Food Truck costs $500 + $1,000 via upgrades, or $1,500 built directly, and resells for $750.
- **Trade-off:** upgrades are lost when a building is sold, by design. Sold buildings appear on the chart as a triangle, so shape alone tells it apart from built (square) and upgraded (diamond).

## 25. Average purchase price is a weighted-average cost basis (slice 10b)
- **Did:** `Game.CostBasis` holds the total dollars paid for each resource's stock, so average cost is basis ÷ stock. Buying adds the cost; selling, forced sales and production take a proportional share (rounded to the nearest dollar; taking the last case zeroes the basis exactly). Production removes its inputs' basis and gives the same total to the lemonade it makes, so a lemonade's cost is what its inputs cost. Ice that melts loses its basis. The view carries `avgCost` and `unrealizedGain` (stock at the current bid minus its cost), and the market row shows both as text with a sign, with colour as a second cue.
- **Why weighted average, not FIFO:** it needs one number per resource, has no lots to persist, and matches how the player thinks ("what did these cost me on average").
- **Old saves:** a game saved before this has no basis (`cost_basis` is NULL). On load it is seeded as stock × the current walked price, which is an approximation; from then on it is exact. A Postgres round-trip test covers an old-shape row.
- **Invariants tested:** basis is never negative; no stock means no basis; production conserves cost exactly; each sale's share is within $1 of the exact proportion (random-play test).

## 26. A third chart for prices, fed by a daily price log (slice 10c)
- **Did:** the game stores one `PricePoint` per day (effective prices after the tick, as the player sees them, plus active event keys): one at game start and one at the end of every day that does not end the game. The view returns it as `priceLog`, with event names and `basePrices`. The timeline component gets a third chart on the shared time axis, one line per commodity in the fixed resource colours, with a legend toggle, the shared crosshair and keyboard steps, a table twin, and a `$` / `% of base` switch because lemonade at $90 dwarfs everything else. Event days are shaded with a neutral tint and named on hover, so colour carries no meaning.
- **Why a third chart, not a dual axis:** the two-chart rule (decision 20) stands: one series set per chart, each in its own unit.
- **Old saves:** the log starts at the day of the upgrade, seeded from the newest price history entry, so the chart begins there.
- **Known:** the event backdrop (another agent's feature) makes the page 5 px wider than a 390 px phone under mobile emulation; the app's own content fits.



## 27. Runs: identity, atomic effects, and score = final net worth (slice 11a)
- **Did:** every game has a `RunID` (UUID), assigned at new game and, for older saves, on their first successful mutation. `Repository.Mutate` now takes a function returning `Effects{Report, Finished}`, and the store saves them in the same transaction as the game (a day report row per ended day, a run row when the run ends; both idempotent). Bankruptcy records its run in the same `EndDay` transaction that flips the status. `NetWorth` is cash + stock at the current bid + what every building would resell for, and is shown in the header.
- **Score (user's decision):** a run's score is its final net worth and nothing else. The day count is stored and shown beside it on records and leaderboards, not added into it, so the handoff's proposed `ScorePerDay` is not built.
- **Choices:** `Days` on a record is the day the run ended on (a run lost on day 12 says 12). `difficulty` exists on `runs` with default 3 and is not used yet. Reports and records are read back by a later slice; this one only writes them.

## 28. Give up (slice 11b)
- **Did:** `POST /api/game/give-up` sets the status `gave_up` and records the run (score = net worth right then) in the same transaction. Nothing is sold or charged, and the day does not advance. Every other action already refused anything not `active`, so it returns 409 `game_over` afterwards; a second give up does too, and a test covers each. The game page has a quiet red "Give up" button at the very bottom with a confirmation that shows the score it would record; the result screen is the bankruptcy screen with the title "You called it on day N", the final score with the day beside it, and a cash / stock / buildings breakdown.
- **`run_active` (confirmed by the user):** `POST /api/game/new` returns 409 `run_active` while the run is still active, so a run can only be replaced after it ends, by bankruptcy or by giving up, and every run leaves a record. This is a behaviour change: before, a new game silently discarded an unfinished one. The UI already offered "New game" only on the result screen, so no flow changed; the API test that expected a replacement now expects the refusal. The check and the replacement are two steps, not one transaction, which is safe: a finished run can never become active again.

## 29. Past day reports are read on demand (slice 11c)
- **Did:** every ended day's report is stored (decision 27). `GET /api/game/reports` returns a light list (day, produced, capital before and after, event names, bankrupt) and `GET /api/game/reports/:day` one full report, both for the current run or, with `?runId=`, a finished run of the same player. Another player's run and a made-up one are both 404, so run IDs cannot be probed. Reports are never sent with the game view, so the game does not grow. The game page has a "Past days" button that opens a drawer with previous, next and jump-to-day, reusing the same report body as the end-of-day modal.
- **"Game time" means game days**, not wall-clock time: there is no real-time tick, so the list is simply one entry per ended day.
- **Old saves:** history starts when the update deployed; the drawer's empty state says days played before then are not kept.

## 30. Scores: one row per player, owner-only run detail (slice 13)
- **Did:** `GET /api/scores` ranks each player's best finished run by score, an earlier finish winning a tie, with the caller's own row and rank returned separately so it shows even far down the board. `GET /api/runs` lists the caller's runs with the best flagged, and `GET /api/runs/:runId` returns one run in full (stats, timeline, price log, report index) for its owner only. The boards are computed in one SQL statement (`DISTINCT ON (user_id)` plus `ROW_NUMBER`); the in-memory fake ranks the same way, and one contract test runs against both. Only finished runs appear.
- **Owner only:** a global board row shows no run ID and cannot be opened. Letting players read each other's runs is a privacy and scope change, not made here. Usernames are public on the board and login is username-only, so this is documented as a known limitation.
- **Difficulty and rival are cut:** the board has no difficulty filter, no `rivalResult`, and no difficulty parameter. `runs.difficulty` still exists with its default, so a filter could be added later. The score is final net worth alone (decision 27), so the run detail shows cash and "stock and buildings" rather than a per-day weight.
- **UI:** `/scores` has Global and Mine tabs; a personal-record row opens `/runs/:id` with the totals, all three charts and the day stepper (extracted from the past-days drawer and shared). The game page shows "Best: $X on day N", and the result screen says "New personal best" when the finished run is the player's best run.

