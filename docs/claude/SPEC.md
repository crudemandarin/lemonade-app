# SPEC: Lemonade Tycoon

Turn-based lemonade business game. One core loop: the passage of a **day**. Score = current capital.

## 1. Agreed deliverables
- Full-stack web app: Angular (TypeScript) frontend, Gin (Go) backend, PostgreSQL. Already scaffolded and deployed; this spec covers the game on top of it.
- Username-only login (no password).
- Installable PWA: web manifest, icons, and a service worker that caches the app shell. Gameplay requires the network (the server is the source of truth).
- Playable loop: buy/sell resources at market prices, expand and upgrade facilities, end the day, see a day report, lose by bankruptcy.
- Simulated commodity market for all five resources, with random events that move prices.
- README that lets a fresh clone install, run, and test using only its instructions. It also documents the "physics" knobs (prices, events, costs) and how to tune them.

## 2. Domain vocabulary
- **Resources:** `lemon`, `sugar`, `ice`, `cup` (raw inputs) and `lemonade` (product). One **unit** = one "case" (bulk lot). Prices are per case.
- **Facilities:** two types. **Warehouse** (storage; one warehouse per resource, each with its own building count) and **Production**. Each type has one `level` (1-4) shared by all its buildings; each warehouse resource and Production has a building `count` (1-10).
- **Tiers:** the level names the building type.
  - Warehouse: Pantry (L1) → Garage (L2) → Barn (L3) → Industrial Warehouse (L4).
  - Production: Kitchen (L1) → Food Truck (L2) → Bottling Plant (L3) → Lemonade Factory (L4).
- **Capital:** the player's money, in **whole dollars**.
- **Day:** one turn. **Upkeep:** daily fee charged on facilities.
- **Market:** current price per resource with a bid (sell) and ask (buy) side.
- **Event:** a temporary market shock (weather, holiday, etc.).

## 3. Behavior rules

### Game setup
1. Logging in with an unknown username creates the user and a new game. A known username resumes their game.
2. Each user has exactly one active game. Starting a new game after game over replaces it.
3. A new game starts on Day 1 with starting capital, empty inventory, every facility type at **level 1**, one building per warehouse resource and one Production building, and initial market prices.

### Money
4. All money (capital, prices, costs, upkeep) is whole dollars. There are no cents in the domain, API, or UI. Quotes are rounded to whole dollars (rule 7).

### Player actions (during a day)
5. **Buy:** the player buys `qty` cases of a resource at the current **ask** price. Fails if capital is insufficient or the resource's warehouse lacks free space. No partial fills.
6. **Sell:** the player sells `qty` cases of any resource at the current **bid** price. Fails if inventory is insufficient. Lemonade is sold this way (no price-setting, no demand simulation).
7. Bid = floor(price × (1 − spread)), min $1. Ask = ceil(price × (1 + spread)). This rounding keeps bid < ask and prevents same-day arbitrage.
8. **Capacity:** resource `r`'s warehouse capacity = `count[r] × size(warehouse level)`; a warehouse holds only its own resource. Production capacity per day = `count × rate(production level)`.
9. **Expand** (add one building): for Warehouse the player picks the resource; for Production there is one target. Adds one building at the type's current level for that level's build cost. Fails if capital is insufficient or the count is at the max (10).
10. **Upgrade** (raise the level of a whole type): raises **every** building of that type (all five warehouses, or Production) to the next level, for `upgrade_cost(level) × total buildings of that type`. There are no mixed tiers. Example: a fresh game's 5 Pantries upgrade to 5 Garages for 5 × $100 = $500, and each capacity goes 10 → 20. Fails if at max level or capital is insufficient.
11. Actions are unlimited within a day and take effect immediately. All actions are rejected once the game is over.

### End of day (single atomic transition)
12. **Production:** the Production facility converts inputs into lemonade. Recipe is fixed: 1 lemon + 1 sugar + 1 ice + 1 cup → 1 lemonade. Amount produced = min(daily production capacity, stock of each input, free lemonade warehouse space). Production is automatic.
13. **Melt:** all remaining ice is set to 0. Other resources carry forward.
14. **Upkeep:** per type, `total buildings × upkeep(level)` is owed every night (Warehouse counts buildings across all five resources). Upkeep is always paid in full or the game ends: if cash is short, stock is sold at the current bid (pre-tick quotes; lemonade first, then lemon, sugar, cup, ice), just enough to cover it. If everything sold still cannot cover it, the player pays what they can and the game is over (rule 16). Unpaid upkeep is never forgiven.
15. **Advance:** the day counter increments, the market ticks (rule 18), and events may start or expire (rule 20).
16. **Bankruptcy:** the game is over if upkeep (rule 14) cannot be covered by cash plus the bid value of all stock. Holding stock is not a grace by itself; it only counts for what it would raise at bid. Ice is always 0 by then because it melts in step 13. The check runs only at end of day, so spending down to $0 mid-day is allowed.
17. The player receives a **day report**: production, melted ice, upkeep paid, stock sold to cover upkeep (cases and proceeds), price changes, new and ended events, whether the game ended, and capital before/after.

### Market and events
18. Each resource has a base price. Each day the walked price does a mean-reverting random walk toward its base price, bounded to [0.25×, 4×] base. Walk state may be fractional internally; every quote shown or charged is a whole dollar (min $1).
19. Market evolution is deterministic given the game's seed and day number (so tests and replays are reproducible).
20. Each day there is a chance a new event starts. An event has a name, description, duration in days (1-3), and price multipliers on one or more resources. Active event multipliers stack multiplicatively on the walked price to give the effective price.
21. Events are visible to the player as soon as they are active (same day prices reflect them). No forecasting. Each active event also shows a looping, low-detail animated background (at most two at once; see EVENT-BACKDROPS.md); it is decorative only.

### API and access
22. Every game endpoint requires an identified user (username in the `X-Username` header). This is intentionally not secure, per the brief's "no password" login. Usernames are 5 to 40 printable ASCII characters (no spaces) and case-insensitive: they are lowercased on login and on every request, so `Joe` and `joe` are the same player. Anything else is rejected with `invalid_username`.
23. Errors return a stable JSON shape `{ "error": "<code>", "message": "..." }` with sensible HTTP status codes. The UI displays the message.

### UI
24. Dashboard shows: day, capital, inventory vs. warehouse capacity per resource, current bid/ask per resource, active events, and facilities (tier name, image, level, quantity, capacity/rate, upkeep, next expand and upgrade costs).
25. Player can buy, sell, expand, upgrade, and end the day from the dashboard. After each end-of-day, the day report is displayed.
26. Game-over screen shows final day reached and final capital, with a "New game" button.
27. Each resource shows a small price-history sparkline (last 14 days). *(Stretch; see PLAN slice 8.)*

### PWA
28. The frontend is an installable PWA: web app manifest, icons, and a service worker that caches the app shell (HTML, JS, CSS, icons). Offline, the shell loads and shows an offline notice. API responses are never cached, and buy, sell, expand, upgrade, and end-day are disabled while offline. The service worker is enabled in production builds only.

## 4. Starting values
Tunable, all in one config file (see DESIGN §6). Starting capital $1,000; base prices per case: lemon $20, sugar $10, ice $10, cup $10, lemonade $90 (tuned, DECISIONS 12); spread 10%; sizes, rates, costs, and upkeep per tier in DESIGN §6 (current numbers and tuning guide: README "Game physics").

## 5. Assumptions
- Warehouse tier sizes (10 / 20 / 40 / 80) and Production rates (10 / 20 / 40 / 80 lemonade per day) double per level.
- Each facility type (Warehouse, Production) tracks a single `level` shared by all its buildings (confirmed): upgrade is all-or-nothing. Warehouse counts are per resource; the level is shared across all five.
- The scaffold had no PWA support; it was added in PLAN slice 7.
- Building count is capped at 10 per warehouse resource and for Production; level at 4. Both tunable.
- Ice melts at end of day, **after** production, so ice bought today is used tonight.
- Selling is at unlimited depth (no price impact from the player's trades).
- "Sell resources" includes selling raw inputs back at the bid price.
- The game is endless; there is no win condition. Score is capital (and day reached).
- Facility images are static assets: one per tier (4 warehouse, 4 production); placeholders/emoji-style SVGs are acceptable.
- Production output is available the next morning; the player sells on the following day.

## 6. Open questions (proceeding on the assumption in bold unless told otherwise)
1. Bankruptcy definition (resolved, DECISIONS 12): the brief's "not enough inventory to make lemonade" can never trigger because ice melts (rule 13) before the check, and an "any inventory is a grace" rule let players sit at $0 with a little stock forever. **Upkeep is always owed; short cash sells stock at bid; if that is still not enough the game ends.**
2. Are the tier names and numbers in DESIGN §6 acceptable? **Yes; tune only if time remains.**
3. PWA scope: installable app plus cached app shell, with no offline play. **Yes; the API is never cached and actions are disabled offline.**
4. Is username in a header acceptable as "auth"? **Yes; documented as a known limitation.** Usernames are 5 to 40 ASCII characters and case-insensitive (DECISIONS 15).

Resolved: all buildings of a facility type upgrade together (rule 10); PWA added in PLAN slice 7 (rule 28).

## 7. Out of scope / Future work
See "Future work" in PLAN.md.
