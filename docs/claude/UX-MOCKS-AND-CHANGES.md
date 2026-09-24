# Lemonade Tycoon: change notes and UX mocks

Addendum to `SPEC.md`, `DESIGN.md`, and `PLAN.md`. **Where this file conflicts with those, this file wins.** It contains only what changed after the first draft, plus the UX mocks. Refer to rules by name, not number (numbering changed between drafts).

---

## Part 1: What changed

### 1.1 Quick map: old draft → now

| If your copy says | It now says |
|---|---|
| Money in cents, or cent-level prices (cup $0.15, etc.) | **Whole dollars everywhere.** Market unit is a "case"; prices are per case (see 1.2) |
| Lemonade sold with price-setting or demand simulation | Sold at **market price** only (commodity), with a bid/ask spread |
| Six facility kinds (`warehouse_lemon`, ..., `production`), each with its own level | **Two facility types**: Warehouse and Production. One shared level per type (see 1.3) |
| Facility `quantity` and per-facility level | Warehouse has a building `count` **per resource**; Production has one `count`; level is **per type** |
| Bankruptcy = capital 0 (or capital 0 and can't make lemonade) | Bankruptcy = **capital 0 AND total inventory 0** at end of day (see 1.4) |
| PWA assumed provided by scaffold | **Not provided.** Added in slice 7 (see 1.5) |
| Slice 7 = sparklines | Slice 7 = PWA; sparklines are slice 8 (stretch) |

### 1.2 Money and market

- All money (capital, prices, costs, upkeep) is **whole dollars** in the domain, API, DB, and UI. Type: `int`.
- One market **unit = one case**. One lemonade unit is made from one case each of lemon, sugar, ice, and cup (fixed recipe).
- Base prices per case: lemon $20, sugar $10, ice $10, cup $10, lemonade $100. Starting capital: **$1,000**.
- **Spread 10%.** Ask (buy) = `ceil(price × 1.1)`. Bid (sell) = `max(1, floor(price × 0.9))`. This keeps bid below ask and blocks same-day buy/sell arbitrage.
- Price walk keeps a **float** internally: `p' = p + 0.2·(base − p) + p·σ·N(0,1)`, σ = 0.08, clamped to [0.25×, 4×] base. The quoted **effective price** = `max(1, round(walked × active event multipliers))`. Event multipliers stack multiplicatively and never modify the walked price.
- Randomness is deterministic: `rand.New(rand.NewSource(seed ^ int64(day)))` inside `EndDay`. No `time.Now()` in the domain.
- Selling is at unlimited depth (no price impact). Any resource can be sold at bid, including raw inputs.

### 1.3 Facilities (replaces the old facility model)

**Two facility types.** Each type has **one `level` (1-4) shared by all its buildings**.

- **Warehouse** (resource storage): one warehouse **per resource** (lemon, sugar, ice, cup, lemonade). Each resource has its own building `count`. A warehouse holds only its own resource; capacity is never shared across resources.
- **Production** (lemonade production): one `count`.

Everything starts at **level 1**: one warehouse building per resource, one Production building.

**Actions**

- **Expand (buy one facility):** adds one building at the type's current level for that level's build cost. For Warehouse the player picks the resource; for Production there is one target. Fails at max count or if capital is short.
- **Upgrade (entire type):** raises the type's level by 1, so **every building of that type moves up together** (all five warehouses at once, or all Production buildings). Cost = `per-building upgrade cost × total buildings of that type`. Fails at max level or if capital is short.
- Example: a fresh game's 5 Pantries upgrade to 5 Garages for 5 × $100 = $500; each capacity goes 10 → 20. One resource with 2 Pantries (20) becomes 2 Garages (40).

**Formulas**

- Warehouse capacity for resource `r` = `count[r] × size(warehouse level)`.
- Production per day = `count × rate(production level)`.
- Upkeep per day = `total buildings × upkeep(level)` per type (Warehouse counts buildings across all five resources).

**Tiers** (size/rate, costs, and upkeep are per building)

| | L1 | L2 | L3 | L4 |
|---|---|---|---|---|
| **Warehouse name** | Pantry | Garage | Barn | Industrial Warehouse |
| Size (cases) | 10 | 20 | 40 | 80 |
| Expand (build) cost | $100 | $300 | $800 | $2,000 |
| Upgrade cost to next | $100 | $250 | $600 | n/a |
| Upkeep per day | $1 | $3 | $8 | $20 |
| **Production name** | Kitchen | Food Truck | Bottling Plant | Lemonade Factory |
| Rate (lemonade/day) | 10 | 20 | 40 | 80 |
| Expand (build) cost | $500 | $1,500 | $4,000 | $10,000 |
| Upgrade cost to next | $1,000 | $2,500 | $6,000 | n/a |
| Upkeep per day | $10 | $25 | $60 | $140 |

Limits: max level 4; max buildings **10 per warehouse (per resource)** and **10 for Production**. All numbers live in one config struct.

Starting upkeep: 5 warehouse buildings × $1 + 1 Production × $10 = **$15/day**. Sanity check: one batch of 10 costs about $550 at ask and sells for about $92 each at bid, so roughly $350 profit per day at the start. Not tuned for fun.

**Data model**

```
games.facilities JSONB:
{ "warehouse":  { "level": 1, "counts": {"lemon":1,"sugar":1,"ice":1,"cup":1,"lemonade":1} },
  "production": { "level": 1, "count": 1 } }
```

Tier names (Pantry, Garage, ...) are derived from type + level via config; never stored. Money columns are integers (`capital`).

**Domain surface (pure Go)**

```go
ExpandWarehouse(g *Game, r Resource) error   // that resource's count + 1
ExpandProduction(g *Game) error              // production count + 1
Upgrade(g *Game, t FacilityType) error       // type level + 1 for every building
Capacity(g Game, r Resource) int             // count[r] * size(warehouse level)
IsBankrupt(g Game) bool                      // capital == 0 && total inventory == 0
Quotes(g Game) map[Resource]Quote            // effective price, bid, ask (whole dollars)
```

**API changes**

| Method and path | Purpose |
|---|---|
| `POST /api/game/facilities/warehouse/expand` `{resource}` | Add one warehouse building for that resource |
| `POST /api/game/facilities/production/expand` | Add one Production building |
| `POST /api/game/facilities/:type/upgrade` | Upgrade the whole type (`warehouse` or `production`) |

The game view returns, per type: tier name, level, upkeep, upgrade cost, and the number of buildings it covers. It also returns, per warehouse resource and for Production: count, capacity or rate, and expand cost.

### 1.4 End of day and bankruptcy

Order, as one atomic transition:

1. **Produce:** `min(production count × rate, stock of each input, free lemonade warehouse space)`.
2. **Melt:** ice → 0.
3. **Upkeep:** deduct; capital clamps at 0 (unpaid upkeep is forgiven).
4. **Advance:** day + 1, market tick, events start/expire.
5. **Bankruptcy check:** game over iff **capital == 0 AND total inventory == 0**.

Notes:

- Any leftover inventory is a grace: it can be sold at bid, so the player can raise cash.
- I did **not** use "not enough inventory to make lemonade" because ice melts in step 2, so that test could never pass.
- The check runs **only at end of day**, so spending down to $0 mid-day is fine.
- Alternatives if you want them later: lemonade-only grace, or inventory value at bid must cover a day's upkeep.
- After game over all actions are rejected; the player starts a new game.

### 1.5 PWA (not in the scaffold)

- Add with `ng add @angular/pwa`: manifest, lemon-themed icons, `ngsw-config.json`, service worker **registered in production builds only**.
- The service worker prefetches the **app shell** and defines **no data group for `/api/**`**, so API responses are never cached.
- An `online.service.ts` signal (from `navigator.onLine` and `online`/`offline` events) drives an **offline banner** and **disables all action buttons** offline. No offline play.
- Verify installability with **Lighthouse on the deployed HTTPS URL**; service workers do not run under `ng serve`.
- Tests: `online.service` reacts to events; action buttons disabled offline. Manual Lighthouse result noted in the README.

### 1.6 Plan changes

Order and timing (cumulative, about 2h10m plus a stretch):

| Slice | Change |
|---|---|
| 0: Verify scaffold (10m) | Also confirm PWA is **absent** |
| 1: Login and persisted game (30m) | Dashboard shows both facility types with tier name, level, and per-resource building counts |
| 2: Buy and sell (45m) | Whole-dollar bid/ask with ceil/floor rounding; tests include a $1 price |
| 3: End day, bankruptcy (65m) | Bankruptcy truth table: capital 0 × any inventory (ice already melted). **Playable-loop milestone; deploy and check** |
| 4: Market simulation (80m) | Float walk, whole-dollar quotes |
| 5: Facilities (105m) | **Rewritten:** `ExpandWarehouse`, `ExpandProduction`, `Upgrade(type)`; two-card panel (see 2.4); tests below |
| 6: Random events (115m) | unchanged |
| 7: **PWA** (130m) | **New** (see 1.5) |
| 8: Sparklines and polish (140m, stretch) | Moved from slice 7 |

Slice 5 acceptance:

- Upgrading Warehouses raises all five capacities at once and costs per-building × total buildings.
- Expanding one resource's warehouse changes only that resource.
- Upgrading Production raises the rate for all its buildings.
- Max level, max count, and insufficient funds are enforced.
- Upkeep in the day report reflects level × total buildings per type.

Slice 5 tests: expand warehouse (only that resource changes, at max count, insufficient funds); expand production; upgrade warehouse (affects every resource, cost scales with total buildings, at max level, insufficient funds); upgrade production; capacity across levels and counts; upkeep sum; end-day uses the upgraded rate; facilities-panel component emits.

Hold points (unchanged): first Reviewer pass near 2:30, second plus fresh-clone check near 3:30.

### 1.7 Assumptions and open questions

- Production output is available the next morning; the player sells on the following day.
- Facility images are static placeholders: 4 warehouse-tier and 4 production-tier assets, chosen by level.
- Username sent in an `X-Username` header is the "auth". Intentionally not secure; documented as a known limitation.
- **Open:** the brief said "three facility types" but only Warehouse and Production were named. **Modeled as two.** If a third exists, add it as another type with its own level, counts, and tier table.
- **Open:** the grace rule is "any inventory" (see 1.4). Change if you prefer a stricter version.

---

## Part 2: UX mocks

Text wireframes of five screens. The interactive versions were shown in chat; these are the export-friendly equivalents. Copy is sentence case, with no terminal punctuation on labels and buttons. There are no emoji; icons come from a single icon set. Buy, Sell, Expand, and Upgrade buttons stay enabled and show the server's error message on failure (except when offline, see 1.5). **One primary button per screen.**

### 2.1 Routes and shared nav

| Route | Screen | Guard |
|---|---|---|
| `/` | Home | none. Signed in: "Play game" reads "Continue game" |
| `/signin` | Sign in | none |
| `/game` | Game page (also shows game over when `status = bankrupt`) | redirects to `/signin` without a stored username |

Shared top nav on **every** page. One input: the username, or none.

```
Signed out
+------------------------------------------------------------------------+
| [logo] Lemonade Tycoon                                   [ Sign in ]   |
+------------------------------------------------------------------------+

Signed in
+------------------------------------------------------------------------+
| [logo] Lemonade Tycoon                       (user) lemonjoe [Log out] |
+------------------------------------------------------------------------+
```

"Log out" clears the stored username (there is no session) and routes to `/`. "Sign in" routes to `/signin`.

### 2.2 Home (signed out)

```
+------------------------------------------------------------------------+
| [nav, signed out]                                                      |
+------------------------------------------------------------------------+
|                                                                        |
|                             [lemon icon]                               |
|                          Lemonade Tycoon                               |
|      Buy low, sell high, and grow your stand into a factory.           |
|                                                                        |
|                   [ Play game (primary) ]  [ Sign in ]                 |
|                                                                        |
+------------------------------------------------------------------------+
```

"Play game" while signed out goes to `/signin`. When signed in, it becomes "Continue game" and goes to `/game`.

### 2.3 Sign in

```
+------------------------------------------------------------------------+
| [nav, signed out]                                                      |
+------------------------------------------------------------------------+
|                    Username                                            |
|                    [ lemonjoe                    ]                     |
|                    [ Continue (primary)          ]                     |
|                    New name? We'll start a game for it.                |
+------------------------------------------------------------------------+
```

Empty username shows an inline error under the field (for example "Enter a username"). `POST /api/login` creates or gets the user, then routes to `/game`.

### 2.4 Game page

```
+------------------------------------------------------------------------+
| [nav, signed in]                                                       |
+------------------------------------------------------------------------+
| Day         Capital        Upkeep per day                              |
| 4           $1,240         $15                        [ End day -> ]   |
+------------------------------------------------------------------------+
| (sun) Heat wave: lemonade x1.4 and ice x1.3 for 2 more days            |  events banner
+------------------------------------------------------------------------+
| Market and inventory                                                   |
| Resource  Stock         Price        Buy at ask       Sell at bid      |
| Lemon     [####------]  $20 ^ ~~~    [1] [Buy $22]    [1] [Sell $18]   |
|           4 / 10                                                       |
| Sugar     [####------]  $11 v ~~~    [1] [Buy $13]    [1] [Sell $9]    |
|           4 / 10                                                       |
| Ice       [----------]  $13 ^ ~~~    [1] [Buy $15]    [1] [Sell $11]   |
|           0 / 10                                                       |
| Cups      [####------]  $10 ^ ~~~    [1] [Buy $11]    [1] [Sell $9]    |
|           4 / 10                                                       |
| Lemonade  [##########]  $140 ^ ~~~   [1] [Buy $154]   [1] [Sell $126]  |
|           10 / 10                                                      |
+------------------------------------------------------------------------+
| Facilities                                                             |
| +---------------------------------------------+ +--------------------+ |
| | [img] Pantry                                | | [img] Kitchen      | |
| |       Warehouses, level 1 of 4              | |       Production,  | |
| |       Upkeep $5 per day                     | |       level 1 of 4 | |
| | [ Upgrade all warehouses to Garage: $500 ]  | | 1 building         | |
| | Covers 5 buildings at $100 each. Each       | | Makes 10 lemonade  | |
| | building goes from 10 to 20 cases.          | | per day            | |
| |---------------------------------------------| | Upkeep $10 per day | |
| | Lemon     x1  Holds 10 cases  [Expand $100] | | [ Expand: $500 ]   | |
| | Sugar     x1  Holds 10 cases  [Expand $100] | | [ Upgrade to Food  | |
| | Ice       x1  Holds 10 cases  [Expand $100] | |   Truck: $1,000 ]  | |
| | Cups      x1  Holds 10 cases  [Expand $100] | | Covers 1 building  | |
| | Lemonade  x1  Holds 10 cases  [Expand $100] | | at $1,000 each.    | |
| +---------------------------------------------+ +--------------------+ |
+------------------------------------------------------------------------+
```

Notes:

- **Primary button:** only "End day".
- **Stock bar:** stock over capacity, where capacity = `count × size(level)`.
- **Price cell:** effective whole-dollar price, a trend arrow versus yesterday, and a 14-day sparkline (sparkline is slice 8; show only the arrow before then).
- **Bid and ask:** the Buy button shows the ask, the Sell button shows the bid, so a quantity's cost is visible before clicking.
- **Upgrade all warehouses:** one button for the whole Warehouse type, showing its cost and how many buildings it covers. At max level it reads "Max level" and does nothing.
- **Expand per warehouse row:** that resource only; Production has its own Expand and Upgrade.
- **Facility image:** placeholder box; real art keyed by tier.

### 2.5 Day report (modal over the game page)

```
+------------------------------------+
| Day 4 report                       |
|                                    |
| Lemonade produced        +10       |
| Ice melted               2 cases   |
| Upkeep paid              -$15      |
| Lemonade price           $100 -> $140
| New event                Heat wave |
| ---------------------------------- |
| Capital           $1,240 -> $1,225 |
|                                    |
| [ Start day 5 (primary) ]          |
+------------------------------------+
```

Content comes straight from the `DayReport` returned by `POST /api/game/end-day`. Dismissing shows the updated game page.

### 2.6 Game over

```
+------------------------------------------------------------------------+
| [nav, signed in]                                                       |
+------------------------------------------------------------------------+
|                          [sad face icon]                               |
|                       Bankrupt on day 12                               |
|         You ran out of cash and stock. Final capital: $0.              |
|                                                                        |
|                       [ New game (primary) ]                           |
+------------------------------------------------------------------------+
```

Shown at `/game` when `status = bankrupt`. "New game" calls `POST /api/game/new`.

### 2.7 Component and data map

| Component | Data (from the single game view) | Emits |
|---|---|---|
| `nav-bar` | username or none | sign in, log out |
| `stats-strip` | day, capital, total upkeep | end day |
| `events-banner` | active events (name, affected resources and multipliers, days left) | none |
| `market-panel` | per resource: stock, capacity, effective price, trend, bid, ask, history | buy(resource, qty), sell(resource, qty) |
| `facilities-panel` | per type: tier name, level, upkeep, upgrade cost, buildings covered; per warehouse resource and Production: count, capacity or rate, expand cost | expandWarehouse(resource), expandProduction, upgrade(type) |
| `day-report-modal` | `DayReport` | dismiss |
| `game-over` | status, final day, final capital | new game |
| `offline-banner` | `online.service` signal | none (disables action buttons) |

Only `game.store` talks to the API. Every mutating call returns the updated game view, so the UI never needs a follow-up fetch.

### 2.8 Open UX questions

1. Should signed-in users auto-redirect from `/` to `/game`? **Assumed no; keep the explicit "Continue game" button.**
2. Mobile layout: the market rows are wide, so on a phone each resource should stack as a card. **Assumed later, in the stretch slice; the PWA is not phone-first.**
