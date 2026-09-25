# Handoff: late game Empire track (territories and rivals, then victory and economic cycles)

Read `HANDOFF-LATE-GAME-ROADMAP.md` first (waves, shared contracts, branch `late/empire`). Then read `HANDOFF.md`, `HANDOFF-P0.md` sections 0 and 1, `LATE-GAME-DESIGN.md` sections 0, 2, 5, 6.2, 6.3 and 6.8, and `LATE-GAME-CONTENT.md` sections A to E and I (cycles). Check `git log` and `git status`. Stage explicit paths only. Test first.

**User decisions already made:**
- Rivals **may move the player's share and run price wars**. This supersedes "rival race only" and "no shared-market rival pressure".
- There is a **victory condition** ("Global leader"), and the player can keep playing after it.
- The target length is **150 to 200 days** for a diligent player to finish.
- Prices stay global (one price per commodity). Territories add depth, not separate prices.

**Ask the user first:** open questions 6 (how adversarial; default 1 share point a day at most, with a floor), 7 (hostile takeovers; default yes at 1.6 times value), 8 (name tone; default light puns, as in the catalog), 9 (rivals fold and make merger offers, but no cross-territory growth), 11 (one global facility pool with caps per territory), 22 (acquisitions count in net worth at 50% of the price paid), 27 (economic cycles in).

**Concurrency:**
- **Stage A:** wave 2, alongside Upgrades A. It needs Products A merged. Build the territory mechanics first. After Upgrades A has merged its framework commit, add the territory upgrades (`roadside_sign`, `billboard`, `radio_spot`, `tv_campaign`, `global_brand`, `delivery_fleet`, `rival_intel`, `pr_team`) as rows with the effect types `PresenceBonus` and `HubUpkeepDiscount`.
- **Stage B:** wave 3, alongside Products B.

## Stage A: territories and rivals (design phase 3)

Build in this order, merging each into the integration branch: Neighborhood rivals and share, then City, then buyouts and campaigns, then rival events, then Region, Nation and World (content rows plus tiers 5 to 7).

### Reach, and how it connects to phase 0
Phase 0 made depth `FreeDepth x DepthByLevel[warehouse level]` (80/280/480/960) behind the helper `freeDepth(g, cfg, r)`. The recommended way to keep that feel:
- **Neighborhood depth** = the phase 0 value × (Neighborhood share ÷ 40%). The player starts at 40%, so an unchanged game (including every old save) plays exactly as it does today. Buying out Neighborhood rivals scales it up.
- **Every other territory** adds `demand(product, t) × share(t) × brandMultiplier`.
- Reach for a product is the sum. `freeDepth` returns it, so no call site changes.

Confirm this with the user. The alternative is dropping the warehouse-level term entirely, which would re-balance phase 0.

### Domain (pure)
- **Content.** `content/territories.go` (content section A), `content/rivals.go` (section C) and the rival event table (section D). Tiers 5 to 7 go into the tier tables (section B), with an `Era` gate on each level.
- **State.** `Game.Territories map[string]TerritoryState{Entered bool, Share float64, CampaignDaysLeft int, CampaignBonus float64}` and `Game.Rivals map[string]RivalState{Share, Valuation, Momentum, Status, Mood, TelegraphKey, TelegraphDay}`.
  - Old saves get the Neighborhood entered at 40%, with its three rivals at their catalog shares.
  - An invariant test checks that per territory, the player's share plus the active rivals' shares equals 100%.
- **Era** is the highest territory entered. It becomes the `Requires.Era` gate that Upgrades and Products use (fill in the field they left).
- **Building cap** is 10 per type per territory entered, with +20 for the World (question 11). Tiers unlock by era. Old saves at levels 3 or 4 are grandfathered.
- **Actions.**
  - `POST /api/game/territories/:key/enter` needs the previous territory and the cost, and gives the entry share.
  - `POST /api/game/rivals/:key/buyout {hostile?: bool}` pays valuation × premium and moves the share. The rival's buildings come as discounted buildings where the catalog says so.
  - `POST /api/game/territories/:key/campaign {level}` spends cash for a presence bonus lasting N days.
  - Errors are 409 with reasons: `territory_locked`, `rival_refuses` (with its condition), `insufficient_funds`.
- **Rivals tick** (step 10 of the end-of-day order, with its own salt):
  1. Valuation update: roughly share × demand × margin × multiple, plus momentum, with boom and recession applied once Stage B lands.
  2. Share contest: presence is brand upgrades + campaign + fill rate + a small price term, against the rival's strength. At most `MaxShareShiftPerDay` moves per day, and never below the player's floor.
  3. Personality rules from the catalog.
  4. Rival events: telegraphed one day ahead where the catalog says so. A price war is an **ordinary event** (a multiplier on the effective price), and it never touches the walk or inventory.
- **Hub upkeep.** It is added in step 5, with the synergy discount.
- **Net worth.** Add acquisitions at `ResaleRate` × the price paid (question 22). Entry costs and campaigns are not counted.
- **Salts.** A test checks that adding rivals does not change market prices for an existing seed.

### API and UI
- `GET /api/game/empire` returns territories, shares, rivals, telegraphs and costs. It is a separate endpoint to keep the game view small. The game view only gets `era`, `reach` per product and `nextGoal`.
- The **Empire strip** on the game page shows the era, "Selling X of Y depth · Z capacity", and the next goal.
- The **Empire page** in the nav menu is a territory ladder (a vertical list at 390 px). Each territory shows share bars (text labels, not colour alone), rival cards (personality, valuation, trend, telegraph) and buyout and campaign buttons (secondary). A confirmation shows the price and what you get.
- **Compact money formatting** (`$1.2M`) lands here if Products has not done it yet (question 30). Exact values go in tooltips and tables.
- The **market panel's** depth hint shows the breakdown per territory.
- **Glossary:** territory, share, reach, rival, personality, buyout, hostile takeover, presence, campaign, hub upkeep, era.

### Balance
- Add a **diligent bot** (the design's benchmark): it enters territories when payback is 25 days or less, buys the cheapest rival per share point when cash allows it with a cushion, runs campaigns only to defend a share under threat, and keeps capacity about equal to reach.
- Era targets from design section 5 (the diligent bot), with a band of 1.5× for the careful grower.
- The victory target is the same length: day 150 to 200 for diligent play (Stage B).
- Careful-grower bankruptcy stays at 9% or less, and diligent at 10% or less. The spammer still fails.
- No bot is flat for more than 15 days before era 5.
- A buyout never returns its cost within 10 days at base prices (no instant flips).
- Sweep the territory depths, entry costs, rival growth and valuation multiple, and the buyout premium, one at a time. Record every sweep in DECISIONS.
- Extend the harness to 200 days and run the sweeps in parallel.

### Tests
- Share conservation, the floor, the per-day cap, and telegraph-then-act (a rival does what it telegraphed).
- Buyout accounting.
- Determinism and salt isolation.
- Old saves play identically to phase 0 (a golden run).
- Era gates on tiers, upgrades and recipes.
- API errors and the owner scope.
- Frontend specs for the Empire page and strip.

### Achievements
Add the rival and territory rows from content section K (first buyout, Lucy, own Neighborhood, enter each territory, hostile, price war won).

## Stage B: victory condition and economic cycles (design phase 7, part)

- **Victory.**
  - When the player holds 50% or more of the World, or buys out `global_citrus`, the game records a **win**: `RunRecord.EndedBy = "won"`, `WonOnDay`, and the net worth then.
  - The game shows a victory screen (the result-screen layout, "Global leader on day N") with a choice to keep playing.
  - Keeping playing means the run continues and the win stays recorded. The run's final score is still net worth at the end. **Check with the user** whether a won run's leaderboard score freezes at the win, or keeps growing (default: it keeps growing, and the win day is shown).
  - Add a leaderboard "won" badge, and the `victory` and `underdog` achievements.
- **Economic cycles.**
  - Content section I cycle table. It is a long regime, at most one active at a time, with a 3% daily start chance (its own salt).
  - Effects: product depth ±15% and rival valuations (boom +20%, recession −25%). Inflation, if kept, drifts all prices +1% a day and raises upkeep +10%.
  - Show it in the events banner as a slow-moving line. The `economist` upgrade reveals the days left.
  - The spawn check is in step 9, after events.
  - Tests: only one regime at a time, determinism, and that no cycle changes existing seeds' market prices before it starts.
- **Balance.** Re-run the era targets with cycles on. A recession must not push careful-grower bankruptcy above 9%.

## Docs (both stages)
- `DECISIONS.md`:
  - superseding the rival decisions
  - the Neighborhood reach formula
  - share contest rules
  - buyout valuation
  - net worth of acquisitions
  - the victory rules
  - cycles
- `numeric-tdd.md`: data model, API and section 9 era tables.
- README "Tuning the game": territories, rivals and cycles.
- The glossary.
- `PLAN.md`.
- Add a "Log candidate:" note for the diligent bot as the benchmark.
