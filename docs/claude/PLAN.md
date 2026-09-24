# PLAN: Lemonade Tycoon

Vertical slices, thinnest end-to-end first. Budget ≈ 2h10m build including PWA; the last slice is a stretch. Each slice ships something you can see in the running app. Follow the CLAUDE.md workflow (failing tests, implement, run everything, commit, tick, log deviations). Rule numbers refer to SPEC.md.

Time estimates are rough and cumulative.

- [x] **Slice 0: Verify scaffold and docs (≈10m)** _(deployed health check: use `/api/health`; see DECISIONS 15)_
  - Do: confirm backend, frontend, DB, and deploy all run; fill in every `TODO` command in CLAUDE.md; add `GET /api/health`; write a README skeleton (install, run, test).
  - Acceptance: from a fresh clone, README steps start backend, frontend, and DB; `/api/health` returns 200 locally and deployed; CLAUDE.md has no `TODO`; PWA is confirmed absent (it is added in slice 7).
  - Tests: health handler test.

- [x] **Slice 1: Login to see a persisted game (≈20m, cumulative 30m)**
  - Do: domain `Game`, `Config` (tier tables, prices), `NewGame`, `Capacity`; `users` and `games` tables (migration); `POST /api/login`, `GET /api/game`, `POST /api/game/new`; Angular login page and a read-only dashboard (day, capital, inventory vs. capacity, facilities with tier name/level/quantity, static initial quotes).
  - Acceptance: rules 1-4. A new username creates a game at Day 1 with $1,000, empty inventory, and every facility at level 1, quantity 1 (5 Pantries at 10 capacity each, 1 Kitchen at 10/day); logging in again shows the same game; a different username gets a separate game.
  - Tests: `NewGame` values; `Capacity` = quantity × size(level); repo save/load round trip; API login create-or-get; `game.store` unit test.

- [x] **Slice 2: Buy and sell (≈15m, 45m)**
  - Do: domain `Buy`, `Sell`, `Quotes` (whole-dollar bid/ask with ceil/floor); endpoints; market panel and inventory panel with qty inputs and buttons; error display.
  - Acceptance: rules 5-8, 11. Buying reduces capital by ask × qty and fills inventory; capacity and funds violations show a message and change nothing; selling at bid returns cash; all amounts are whole dollars.
  - Tests: table tests for success, insufficient funds, capacity exceeded, insufficient stock, qty ≤ 0; bid < ask and bid ≥ $1 across a price range including $1; API error shape; market-panel component emits.

- [x] **Slice 3: End day, production, upkeep, bankruptcy (≈20m, 65m)**
  - Do: domain `EndDay` (production, ice melt, upkeep with clamp, day advance, bankruptcy check) with **static prices for now**; `POST /api/game/end-day`; day-report modal; game-over page with New game.
  - Acceptance: rules 12-17 and 26. The full loop works end to end: buy inputs, end day, get lemonade, sell it, see capital change. Ending a day with capital 0 and no inventory gives game over and blocks further actions; capital 0 with any inventory left (e.g. unsold lemonade) does not; spending to $0 mid-day does not.
  - Tests: production = min(rate × quantity, each input, free lemonade space); ice melts; upkeep clamps at 0; bankruptcy truth table (capital 0 × any inventory; ice counts as gone because it melts first); actions rejected after game over.
  - **Milestone: fully playable loop at ≈65m. Deploy and check.**

- [x] **Slice 4: Market simulation (≈15m, 80m)**
  - Do: seeded mean-reverting price walk with clamp in `EndDay`; store price history (last 14) in `market`; show price change arrows in UI.
  - Acceptance: rules 18, 19. Prices change each day within bounds; same seed and same actions give identical prices; quotes remain whole dollars (min $1).
  - Tests: determinism; clamp bounds over 1000 simulated days; mean reversion sanity (average stays near base); history capped at 14; quote rounding.

- [x] **Slice 5: Facilities: expand and upgrade (≈25m, 105m)**
  - Do: domain `Expand` and `Upgrade`; tier tables (names, size/rate, build cost, upgrade cost, upkeep) in config; upkeep from level × quantity; capacity and production from level × quantity; endpoints `.../expand` and `.../upgrade`; facilities panel with tier name, image, level, quantity, capacity, upkeep, and both buttons with costs.
  - Acceptance: rules 8-10, 14, 24. Example from the spec holds: 2 Pantries (20) upgraded gives 2 Garages (40), costing 2 × $200. Expanding adds one building's worth of capacity at the current level. Max level and max quantity enforced; insufficient funds rejected. Upkeep in the day report reflects level × quantity. A higher-tier Kitchen or more Kitchens produces more per day.
  - Tests: expand success, at max quantity, insufficient funds; upgrade success (cost scales with quantity), at max level, insufficient funds; capacity lookups across levels and quantities; upkeep sum; end-day uses upgraded Production rate; facilities-panel component emits.

- [x] **Slice 6: Random events (≈10m, 115m)**
  - Do: event table in config, spawn chance, duration countdown, multipliers applied to effective price; events banner in UI; events listed in the day report.
  - Acceptance: rules 20, 21. An event affects only its target resources' effective prices, stacks multiplicatively, and expires after its duration.
  - Tests: spawn with a forced RNG, expiry, stacking, no effect on the walked price.

- [x] **Slice 7: PWA (≈15m, 130m)**
  - Do: run `ng add @angular/pwa`; set app name, theme color, and lemon-themed icons in `manifest.webmanifest`; `ngsw-config.json` prefetches the app shell and has no data group for `/api/**`; `online.service.ts` signal; offline banner; disable all action buttons while offline; deploy the production build.
  - Acceptance: rule 28. The deployed HTTPS site passes Chrome's installability check and Chrome offers Install; with the network off, the cached shell loads and shows the offline banner with actions disabled; API responses are never served from the service worker cache; `ng build` output contains `ngsw.json`.
  - Tests: `online.service` reacts to online/offline events; action buttons disabled when offline; manual installability check recorded in DECISIONS 6. Note: the service worker runs in production builds only, so verify with a built app, not `ng serve`.

- [x] **Slice 8 (stretch): Sparklines and polish (≈10m, 140m)**
  - Do: inline-SVG sparkline per resource from price history; loading and empty states; README section on tuning the physics.
  - Acceptance: rule 27; README documents the config tables and how to tune them.
  - Tests: sparkline component renders N points for N history entries.

## Hold points
- **≈2:30 elapsed:** first Reviewer pass (see kickoff prompt 3). Fix only SPEC gaps and bugs.
- **≈3:30 elapsed:** second Reviewer pass plus fresh-clone check; then walkthrough prep.

## Future work (cut for time)
- Mixed tiers within one facility (some Pantries, some Garages) and selling or downgrading buildings.
- More facility types (fridge, marketing sign) and per-resource specialty warehouses (e.g. freezer for ice).
- Demand simulation with player-set price and weather-driven customer counts.
- Price impact from the player's own trades and limited market depth.
- Event forecasting ("heat wave expected tomorrow").
- Bulk-purchase discounts and contracts; loans and interest.
- Gentler bankruptcy variants (e.g. a one-day grace period) if the balance proves too harsh.
- User-tweakable recipe.
- Leaderboard using `capital` and `day`.
- Real authentication.
- PWA extras: update-available prompt, read-only cached game view offline, push notifications.
- End-to-end browser tests; CI pipeline.
- Balance tuning for actual fun.
