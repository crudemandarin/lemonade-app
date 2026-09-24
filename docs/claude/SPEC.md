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
- **Facilities:** five **Warehouses** (one per resource) and one **Production** facility. Each has a `level` (1-4) and a `quantity` (number of buildings).
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
3. A new game starts on Day 1 with starting capital, empty inventory, every facility at **level 1, quantity 1**, and initial market prices.

### Money
4. All money (capital, prices, costs, upkeep) is whole dollars. There are no cents in the domain, API, or UI. Quotes are rounded to whole dollars (rule 7).

### Player actions (during a day)
5. **Buy:** the player buys `qty` cases of a resource at the current **ask** price. Fails if capital is insufficient or the resource's warehouse lacks free space. No partial fills.
6. **Sell:** the player sells `qty` cases of any resource at the current **bid** price. Fails if inventory is insufficient. Lemonade is sold this way (no price-setting, no demand simulation).
7. Bid = floor(price × (1 − spread)), min $1. Ask = ceil(price × (1 + spread)). This rounding keeps bid < ask and prevents same-day arbitrage.
8. **Capacity:** a warehouse's capacity = `quantity × size(level)`. Production capacity per day = `quantity × rate(level)`.
9. **Expand** (increase quantity): adds one building at the facility's current level, for the build cost of that level. Capacity rises by one building's size. Fails if capital is insufficient or quantity is at the max.
10. **Upgrade** (increase level): raises **all** buildings of that facility to the next level (buildings of one facility always share one level, so there are no mixed tiers), for `upgrade_cost(level) × quantity`. Example: 2 Pantries (10 each = 20) upgraded become 2 Garages (20 each = 40). Fails if at max level or capital is insufficient.
11. Actions are unlimited within a day and take effect immediately. All actions are rejected once the game is over.

### End of day (single atomic transition)
12. **Production:** the Production facility converts inputs into lemonade. Recipe is fixed: 1 lemon + 1 sugar + 1 ice + 1 cup → 1 lemonade. Amount produced = min(daily production capacity, stock of each input, free lemonade warehouse space). Production is automatic.
13. **Melt:** all remaining ice is set to 0. Other resources carry forward.
14. **Upkeep:** for each facility, `quantity × upkeep(level)` is deducted from capital. Capital never goes below 0 (clamped).
15. **Advance:** the day counter increments, the market ticks (rule 18), and events may start or expire (rule 20).
16. **Bankruptcy:** the game is over if, after steps 12-15, **capital is 0 AND total inventory is 0** (no cases of any resource left). Any inventory is a grace: every resource can be sold at bid (rule 6), so the player can still raise cash. Ice is always 0 at check time because it melts in step 13. The check runs only at end of day, so spending down to $0 mid-day is allowed.
17. The player receives a **day report**: production, melted ice, upkeep paid, price changes, new/active events, and capital before/after.

### Market and events
18. Each resource has a base price. Each day the walked price does a mean-reverting random walk toward its base price, bounded to [0.25×, 4×] base. Walk state may be fractional internally; every quote shown or charged is a whole dollar (min $1).
19. Market evolution is deterministic given the game's seed and day number (so tests and replays are reproducible).
20. Each day there is a chance a new event starts. An event has a name, description, duration in days (1-3), and price multipliers on one or more resources. Active event multipliers stack multiplicatively on the walked price to give the effective price.
21. Events are visible to the player as soon as they are active (same day prices reflect them). No forecasting. Each active event also shows a looping, low-detail animated background (at most two at once; see EVENT-BACKDROPS.md); it is decorative only.

### API and access
22. Every game endpoint requires an identified user (username in a request header). This is intentionally not secure, per the brief's "no password" login.
23. Errors return a stable JSON shape `{ "error": "<code>", "message": "..." }` with sensible HTTP status codes. The UI displays the message.

### UI
24. Dashboard shows: day, capital, inventory vs. warehouse capacity per resource, current bid/ask per resource, active events, and facilities (tier name, image, level, quantity, capacity/rate, upkeep, next expand and upgrade costs).
25. Player can buy, sell, expand, upgrade, and end the day from the dashboard. After each end-of-day, the day report is displayed.
26. Game-over screen shows final day reached and final capital, with a "New game" button.
27. Each resource shows a small price-history sparkline (last 14 days). *(Stretch; see PLAN slice 8.)*

### PWA
28. The frontend is an installable PWA: web app manifest, icons, and a service worker that caches the app shell (HTML, JS, CSS, icons). Offline, the shell loads and shows an offline notice. API responses are never cached, and buy, sell, expand, upgrade, and end-day are disabled while offline. The service worker is enabled in production builds only.

## 4. Starting values
Tunable, all in one config file (see DESIGN §6). Starting capital $1,000; base prices per case: lemon $20, sugar $10, ice $10, cup $10, lemonade $100; spread 10%; sizes, rates, costs, and upkeep per tier in DESIGN.

## 5. Assumptions
- Warehouse tier sizes (10 / 20 / 40 / 80) and Production rates (10 / 20 / 40 / 80 lemonade per day) double per level.
- Each facility tracks a single `level` shared by all its buildings (confirmed): upgrade is all-or-nothing, matching the "2 pantries → 2 garages" example.
- The scaffold does **not** include PWA support. It is added in PLAN slice 7 (via `ng add @angular/pwa` or equivalent).
- Quantity is capped at 10 buildings per facility; level at 4. Both tunable.
- Ice melts at end of day, **after** production, so ice bought today is used tonight.
- Selling is at unlimited depth (no price impact from the player's trades).
- "Sell resources" includes selling raw inputs back at the bid price.
- The game is endless; there is no win condition. Score is capital (and day reached).
- Facility images are static assets: one per tier (4 warehouse, 4 production); placeholders/emoji-style SVGs are acceptable.
- Production output is available the next morning; the player sells on the following day.

## 6. Open questions (proceeding on the assumption in bold unless told otherwise)
1. Grace definition: bankruptcy needs capital 0 **and no inventory at all**. I did not use the brief's "not enough inventory to make lemonade" because ice melts (rule 13) before the check, so that test could never pass. **Keep "any inventory is grace". Alternatives: lemonade-only grace, or inventory value at bid must cover a day's upkeep.**
2. Are the tier names and numbers in DESIGN §6 acceptable? **Yes; tune only if time remains.**
3. PWA scope: installable app plus cached app shell, with no offline play. **Yes; the API is never cached and actions are disabled offline.**
4. Is username in a header acceptable as "auth"? **Yes; documented as a known limitation.**

Resolved: all buildings of a facility upgrade together (rule 10); the scaffold has no PWA support (rule 28, PLAN slice 7).

## 7. Out of scope / Future work
See "Future work" in PLAN.md.
