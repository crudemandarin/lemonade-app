# Handoff: late game phase 0, unblock growth (balance only)

For the agent who implements this. Read in this order: `HANDOFF.md` (repo, stack, rules), `HANDOFF-P0.md` sections 0 and 1 (ground rules, commands), `HANDOFF-BALANCE-PASS2.md` (the market-depth results, bot methods and gotchas; **this phase is its "pass 2"**), then `LATE-GAME-DESIGN.md` sections 0, 2 and 6.1. This is a snapshot: check `git log` and `git status` first, because other agents share the tree. **Never `git add -A`; stage explicit paths.** Test first, keep the domain pure, append to `DECISIONS.md` after the last entry, add a slice to `PLAN.md`, and ask the user when something is ambiguous.

**Scope: numbers and one formula only.** No new UI screens, no territories, no rivals, no tiers 5 to 7, no new content. Those are later phases of `LATE-GAME-DESIGN.md`.

## 1. Why (the user has seen and accepted this)

Careful players flatten at about $15k and never reach L3. A steady-state model of the current rules (`LATE-GAME-DESIGN.md` section 2.2) shows why:
- With 50% overnight recovery, a player selling `q` cases a day starts each day with pressure `q`. So the market absorbs only about **54 cases a day**, a gross ceiling of about **$1,167 a day**, whatever the player builds.
- Best net profit falls with every production level (L1 $987, L2 $927, L3 $767, L4 $687 a day), and a maxed business pays $4,800 a day in upkeep.
- Tiers cost more per case as they grow (upkeep per case: Kitchen $2.00, Factory $3.50).

So upgrading is a trap, and nothing grows demand.

## 2. The three changes

### 2.1 Market depth grows with warehouse level (interim, until territories)

- Replace the flat `FreeDepth` with **base depth times a level multiplier**: keep `FreeDepth map[Resource]int` as the level-1 value (80) and add `DepthByLevel []float64` indexed by `WarehouseLevel - 1`. Starting values: `1, 2, 4, 8`, which gives 80, 160, 320 and 640.
- Add a helper `freeDepth(g, cfg, r) int` in `impact.go` and use it everywhere `cfg.FreeDepth[r]` is read today: `impactMove`, `FreeDepthLeft`, and anything that calls them.
- `FreeDepthLeft(cfg, r, pressure)` needs the game (for the level), so update its signature and its callers (`internal/api/dto.go` around line 301, and the bots).
- **Why warehouse level:** it is the cheapest proxy for "a bigger business reaches more customers." Phase 3 of the design replaces it with territory reach through the same `freeDepth` helper, so the call sites do not change again. Write a comment saying so.

### 2.2 Price impact scales with depth (the same feel at every size)

- Replace the fixed per-case slope with a shape relative to depth: `impact = min(ImpactCap, ImpactShape x excess / depth)`. Rename `ImpactSlope` to `ImpactShape` in `Config`, with **0.24**, which equals today's 0.3% per case at depth 80.
- **Level 1 must behave exactly as today.** Add a test that, for depth 80, every unit price from the new formula equals the old `0.003 x excess` formula across a range of pressures and quantities. Floating point: `0.24 * e / 80` and `0.003 * e` can differ in the last bit. The existing `snap()` (1e-6) should absorb this; if any case flips, compute it as `ImpactShape * excess / depth` consistently and fix the test expectation, not the rounding.
- Keep separate buy and sell pressure, 50% recovery and the 60% cap (open question 32 in the design doc, recommended "keep").

### 2.3 Economies of scale in the tier tables

Starting values (per building). These come from `LATE-GAME-CONTENT.md` table B, levels 1 to 4 only:

| Level | Production | Size | Build | Upkeep | Upgrade to next | Warehouse | Size | Build | Upkeep | Upgrade to next |
|---|---|---|---|---|---|---|---|---|---|---|
| 1 | Kitchen | 10 | $500 | $20 | $660 | Pantry | 10 | $100 | $2 | $130 |
| 2 | Food Truck | 25 | $1,100 | $40 | $1,320 | Garage | 25 | $220 | $4 | $270 |
| 3 | Bottling Plant | 60 | $2,200 | $75 | $2,700 | Barn | 60 | $450 | $8 | $540 |
| 4 | Lemonade Factory | 150 | $4,500 | $150 | 0 | Industrial Warehouse | 150 | $900 | $15 | 0 |

- Upkeep per case now **falls** with level (production $2.00, $1.60, $1.25, $1.00).
- Warehouse and production sizes stay equal per level, so a warehouse still holds one day of production (a property `HANDOFF-BALANCE-PASS2.md` relies on).
- Upgrade cost is about 60% of the next tier's build cost, times all buildings (the existing `Upgrade` rule is unchanged).
- Resale stays at 50% of build cost (`ResaleRate`). Rerun the **exploit invariant test** (resale of a building is less than cash spent to reach that level); it must still pass.
- **Old saves:** sizes only grow (10 to 10, 20 to 25, 40 to 60, 80 to 150), so no stored stock can exceed capacity after the change, and upkeep only falls. No migration. Add a test that loads an old-shape game at each level and checks capacity is at least the stored stock.

## 3. Targets (gate for "done")

Run `cd lemonade-api && BALANCE_REPORT=1 go test ./internal/domain -run TestBalanceReport -v -count=1` (200 seeds). Compare with today's baseline (section 2.1 of the design doc):

| Target | Today | Goal |
|---|---|---|
| Careful grower median capital at day 60 | $10.5k | **at least $15k** |
| Careful grower at day 90 | $14.6k | at least $40k and still rising (the day 60 to 90 gain is at least 1.5 times the day 30 to 60 gain) |
| Careful grower reaches L3 | never | median day **90 or earlier** |
| "Everything maxed" | never | **at least 50% of careful growers within 120 days**, median day 90 or later (not too fast) |
| Careful grower bankrupt | 21% | **13% or less** (the pre-impact level) |
| Spammer | 100% bankrupt | still bankrupt or under $1,000 at day 90 in at least 25% of seeds, and never beats a skilled bot at day 60 |
| Early game (days 1 to 20) | as today | careful grower medians within ±10% of today's at days 5, 10, 15 and 20 |
| Guard rails (`TestBalance*`) | pass | pass; widen a band only with the user's approval and log it in DECISIONS with before and after numbers |

Also add **`TestEachLevelPaysMoreThanTheLast`**. Port the steady-state model (below) into a test helper and assert that, at base prices, the best steady-state net profit per day with 10 buildings strictly increases from L1 to L4. This is the permanent guard against the plateau coming back.

### The steady-state model (recreate as a Go helper)

For each level: depth `D = freeDepth` at that level. For a daily volume `q` (1 up to capacity), the starting pressure is `q x (1 - Recovery) / Recovery`. Then, for each case `k = 1..q`, price the lemonade bid and each input ask with the real `unitBid` and `unitAsk` at pressure `start + k`. Gross is the sum of (bid minus input asks). Net is gross minus the upkeep of the buildings needed for `q` (production `ceil(q / size)`, warehouses the same for each of the 5 resources). The best `q` maximizes net. The planning session's Python version gave the numbers in the design doc (current rules: best `q` of 54 and gross $1,167 at every level).

## 4. Method

1. **Test first:** the level-1 equivalence test, the depth-by-level tests (a level-2 player has 160 free cases, and so on), the old-save capacity test, and `TestEachLevelPaysMoreThanTheLast` (it fails today, which is the point).
2. Implement 2.1 and 2.2 with today's tier table. Run the report and record the numbers.
3. Apply the tier table in 2.3. Run it again.
4. **Sweep**, one knob at a time, with a temporary sweep test (as in pass 2, deleted afterwards). Knobs: `DepthByLevel`, tier sizes and upkeep, `ImpactShape`. Record every row in DECISIONS.
5. **If the bots, not the game, are the problem** (for example they never upgrade because `capacityBinds` does not see the new depth), fix the bots and say so. Pass 2 found this twice.
6. Update every exact number: API and domain tests (upgrade costs, capacities, upkeep), `README.md` "Tuning the game", `docs/numeric-tdd.md` section 9 and its config table (lines around 259), `SPEC.md`, `DESIGN.md` numbers if they are quoted, and the glossary (`lemonade-web/src/app/shared/help/glossary.ts` around line 132 says "the first 80 cases"; rewrite it as "the first 80 cases at a Pantry, more with bigger warehouses"). Also update any facility or help copy that states tier sizes or costs.
7. **Small UI touch only:** the facilities panel's warehouse upgrade hint should mention the deeper market (for example "Market depth 80 → 160"), if the view already carries what is needed. If it needs a new DTO field, add `marketDepth` per resource (mirror it in `core/api.models.ts` and `core/testing/fixtures.ts`, check with `tsc --strict`). Nothing else.

## 5. Verification

- `gofmt -l .`, `go vet ./...`, `go test ./...`, the real-Postgres store test (`DATABASE_URL=... go test ./internal/store`), and the balance report.
- Frontend: `npx ng test --watch=false --browsers=ChromeHeadless`, `npm run lint`, `npm run format:check`, `npm run build`.
- **Real-stack check:** compose up. Drive a game by script to warehouse level 3 (setting state in the dev database works: `UPDATE games SET capital=..., warehouse_level=3, warehouse_qty=..., inventory=...`). Confirm that selling 300 lemonade shows no impact inside the new depth and that the depth numbers and upgrade costs in the UI match `Config`. Screenshot the market and facilities panels at 1280 and 390 px.

## 6. Docs and records

- `DECISIONS.md`:
  - depth by warehouse level as an interim for territories
  - the impact shape relative to depth
  - flipped tier economics
  - the before and after tables
  - any band widened
- `PLAN.md`: a "Late game phase 0" slice, ticked. Future work: phases 1 to 7 point to `LATE-GAME-DESIGN.md`.
- Update `HANDOFF-BALANCE-PASS2.md` section 5 to say that question 2 was answered with a mix of a and c, delivered here.
- Flag "Log candidate:" for the steady-state model turned into a permanent test.

## 7. Do not

Add territories, rivals, recipes, upgrades, tiers 5 to 7, hard daily caps, a wider spread, perishable lemonade, or random inventory-destroying events. Do not change bankruptcy rules or forced-sale order, and do not change prices or event tables unless a sweep shows it is needed and the user approves.

If the targets in section 3 cannot all be met, stop and report the closest configuration and its numbers to the user. Do not add a mechanic.
