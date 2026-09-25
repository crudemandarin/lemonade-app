# Handoff: balance pass 2 (results so far and what is left)

For the agent doing the next balance pass. This records everything learned in pass 1 (`HANDOFF-BALANCE.md`, phases 0 and 1) so nothing has to be re-derived. Read `HANDOFF.md` (repo, stack, rules) and `HANDOFF-P0.md` sections 0 and 1 (ground rules, commands) first. It is a snapshot: check `git log` and `git status`, because other agents share the working tree. **Never `git add -A`; stage explicit paths.** Test first, domain stays pure, append to `DECISIONS.md` after entry 33, update `PLAN.md` (slice 14), and ask the user when a requirement is ambiguous.

Everything below is on branch `feature/slices-9-13` (not pushed, not deployed). Relevant commits: `7e67cb2` (phase 0: bots and baseline), `629ed46` (market depth backend), `ec485a3` (UI, docs, layout fixes).

## 1. Where things stand

| Phase (from `HANDOFF-BALANCE.md`) | State |
|---|---|
| 0: exploit bots and baseline | **Done** |
| 1: market depth and price impact | **Done**, knobs tuned by sweep |
| 2: perishable lemonade | Out of scope by the user's decision. Do not build it. |
| 3: retune | **Not started.** Blocked on the user's choice in section 5. |
| 4: UX, docs, verification | Mostly done for phase 1 (glossary, docs, real-stack check). The remainder belongs to whatever phase 3 changes. |

The user's original targets (section 2 of the first handoff) were approved, but pass 1 showed one is not reachable as written (section 4). I proceeded on the reading that success means "the price-blind volume strategy stops beating the careful player"; the user did not object but never explicitly confirmed it. Confirm it.

## 2. The exploit, measured (phase 0 baseline)

200 seeded games per bot, real domain calls, before any game change. Capital medians count only games still running at that day.

| Bot | What it does | Bankrupt | Day 60 | Day 90 | Everything maxed |
|---|---|---|---|---|---|
| Careful grower (existing) | steady batches, reinvests with a cash cushion | 13% | $29.9k | $308k | day 66 (87% within 120 days) |
| **Spammer** | ignores prices: sell all, buy a full batch, expand as soon as affordable | 20% | **$34.6k** | **$388k** | day 59 (82%) |
| Opportunist (= hoarder) | produces whenever the margin is positive, stocks up at dips, sells at once | 38% | $34.8k | $329k | day 65 (69%) |
| Thresholder | as above but skips days under a $10 margin and holds lemonade for a good bid | 64% | $19.2k | $229k | day 68 (47%) |
| Thresholder, literal handoff rule | buys only when ask <= 0.9 x base | 100% | dead | dead | never |

What this showed (all recorded in `numeric-tdd.md` section 9 and `DECISIONS.md` 31):
1. The exploit is **volume**, and needs no skill: a price-blind spammer out-earns the careful player.
2. **Waiting for good prices does not pay.** Upkeep is due every day whether or not anything is produced, so bots that skip days die more. The diagnosis in the first handoff (a free waiting option) was mostly wrong.
3. **Lemonade cannot be hoarded.** A warehouse holds exactly one day of production at every level (10/20/40/80 per building for both facility types), so the "hoarder" sells daily and produces identical results to the opportunist.
4. The literal buy rule (ask <= 0.9 x base) needs the price about 18% under base in lemon, sugar and cups at once, so that bot starves. It is kept in the report as `strict-thr`.

## 3. What was built (phase 1)

Domain (`lemonade-api/internal/domain/`), all pure:
- `impact.go`: the model. `Game.BuyPressure` / `SellPressure` (cases recently bought and sold, per resource, separate). The k-th case (1-based) costs the plain ask, or pays the plain bid, until `FreeDepth[r]` is used up (`pressure + k - free <= 0`), then moves `ImpactSlope` per further case, capped at `ImpactCap`; each unit rounded (ask up, bid down, bid floor $1). `QuoteBuy`, `QuoteSell` (with optional clamp to cash/space/stock), `MarginalAsk`, `MarginalBid`, `FreeDepthLeft`, `TradeQuote` (`Average`, `Slippage`).
- `EndDay` calls `forgetPressure` right after `g.Day++` (pressure *= 1 - `Recovery`, zeroed below 0.01).
- `Buy`, `Sell`, `BuyClamped` (buys the most cash allows counting impact), and `sellStockToCover` (forced sales price each case with impact, so raising the same cash takes more cases) all use it. `NetWorth`, `UnrealizedGain`, `AvgCost` and cost basis are unchanged: **stock is valued at the plain bid and ignores impact (user's decision)**, so a player can pad the score by ending a run holding lots of stock. Revisit if it shows on the leaderboard.
- Config knobs (`DefaultConfig()` in `config.go`): `FreeDepth` (map per resource) **80**, `ImpactSlope` **0.003**, `Recovery` **0.5**, `ImpactCap` **0.6**.

Storage and API:
- `gameRow.BuyPressure` / `SellPressure` JSONB (`buy_pressure`, `sell_pressure`); NULL on older rows loads as zero (Postgres test covers an old-shape row and a round trip).
- The game view's `bid`/`ask` are now the **marginal price of the next single case**. New per-resource fields: `buyDepthLeft`, `sellDepthLeft`, `buyImpactPercent`, `sellImpactPercent`, and `trade` (a ladder of server-priced quotes for the bulk amounts `1`, `10`, `50`, `100`, `all`, clamped to cash, space and stock, each with `qty`, `total`, `averagePrice`, `slippagePercent`).
- `GET /api/game/quote?resource=&side=buy|sell&qty=[&clamp=true]` prices any size without trading (qty 1 to 100000). **No UI uses it yet**: the market panel reads the ladder instead. Keep it or drop it deliberately.

Frontend:
- Market panel buy/sell labels read the ladder (`Buy 10 · $128`); the old client-side price arithmetic is gone. A button `title` explains average price and slippage; a warning icon (words in its `aria-label`) marks a price the player has moved. Glossary entries "Price impact" and "Market depth".
- Contract checked with `tsc --strict` against a real response. Mirrored in `core/api.models.ts` and `core/testing/fixtures.ts` (`tradeLadder()` helper).

Tests added: `impact_test.go` (free-depth exactness, cap, recovery, clamp, quote-equals-cost, forced sales, old saves, and a 400-case property test: buy N then sell N never profits, pressure never negative, cost with impact never below plain, determinism), API tests for the ladder, impact after heavy trading, and the quote endpoint, plus the Postgres pressure tests.

## 4. Results after phase 1 (defaults 80 / 0.3% / 50% / 60%)

Same harness, 200 seeds, 120-day horizon for the new bots:

| Bot | Bankrupt | Day 30 | Day 45 | Day 60 | Day 90 | Notes |
|---|---|---|---|---|---|---|
| Careful grower | 21% (over 120 d) | $3.8k | $10.2k | $10.5k | $14.6k | fills level 1 by day 34, first upgrade day 62, **no bot reaches level 3 or "everything maxed"** |
| Sloppy | 56% (45 d) | | | | | guard rail band 25% to 90%: passes |
| Spammer | **100%** (median day 35) | $14 | $6 | $3 | dead | price-blind; **never beats a skilled bot (0% of seeds)** |
| Thresholder | 20% | $3.1k | $9.1k | $9.1k | $16.8k | |
| Opportunist (= hoarder) | 22% | $3.0k | $10.0k | $8.8k | $15.6k | |

Against the approved targets:
- Spammer bankrupt or under $1,000 at day 90 in at least 25% of seeds: **100%** (met, strongly).
- Best skilled bot at about $6-10k at day 60: **about $9k** (met).
- Spammer never beats the best skilled bot: **0%** (met). The "skilled beats spammer by 1.5x to 5x" wording cannot be measured because the spammer is dead; the ratio is unbounded.
- Careful players mostly survive; guard-rail tests (`TestBalance*`) pass **unchanged**, no band widened.
- Early game unchanged: a level 1 or 2 business stays inside 80 free cases (peak volume per resource is about twice a day's production because of the half-a-night recovery).
- **"Everything maxed" median day 90+ with at least 50% maxing within 120 days: NOT met.** Nobody maxes. Capital flattens around $15k from day 60 to day 90, so the late game has little to work toward.

### How the knobs were chosen
The handoff's starting numbers (40 free cases, 1.5% per case) made **every** bot bankrupt, including the careful grower, so they were unusable. Sweep, all with recovery 0.5 unless noted, medians of survivors, 100 seeds:

| free / slope | careful dead by day 60 | careful day 60 | opportunist day 60 | sloppy dead by 45 | spammer dead |
|---|---|---|---|---|---|
| 60 / 0.5% | 2% | $17k | $18k | 47% | 100% |
| 100 / 0.5% | 0% | $19k | $14k | 48% | 99% |
| 200 / 0.5% | 0% | $46k | $44k | 49% | 98% |
| 40 / 0.3% | 8% | $9k | $12k | 54% | 100% |
| **80 / 0.3%** | **1%** | **$12k** | **$8.8k** | **48%** | **99%** |
| 80 / 0.4% | 3% | $19k | $12k | 47% | 100% |
| 80 / 0.2% | 6% | $8.6k | $8.9k | 49% | 95% |
| recovery 0.3 instead of 0.5 (grid of 40/60/80 x 0.2-0.4%) | much worse: 2% to 57% careful dead by day 60, and mostly above 10% | | | | 99-100% |

Free depth 200 keeps growth strong but a maxed L4 factory (800 cases a day) is far past it, so it still cannot max. Recovery 0.3 (slower forgetting) is markedly harsher.

### Why "maxed" is unreachable under linear impact
At 800 cases a day the k-th case sits hundreds of cases past any sensible free depth, so the marginal margin (about $20 a case) turns negative long before the last building. Any linear slope that makes the spammer lose also makes the top tier unprofitable. This is a design property, not a tuning miss.

## 5. Open decisions for the user (this is what blocks pass 2)

Ask these first; do not add a mechanic without an answer:
1. **Confirm the target reading:** success = the volume spammer no longer beats the careful player (met), and the "1.5x to 5x" wording is dropped.
2. **The late game.** Options, roughly in order of how much I would recommend them:
   - a. **Scale depth with warehouse level** (per-tier `FreeDepth`, for example 80 / 160 / 320 / 640). Bigger warehouses then mean deeper markets, so upgrading is what unlocks growth and maxing becomes reachable but slow. Smallest change; keeps the exploit closed at every tier because the spammer at each tier still exceeds its depth.
   - b. **Concave impact** (for example sqrt of excess) instead of linear, so heavy volume is punished less steeply.
   - c. **Lower big-tier upkeep** (the Factory costs $280 a day per building) so the flat late game still grows.
   - d. **Accept the ceiling**: pass 1's numbers become the design, and "everything maxed" is dropped as a goal. Then only docs and the `TestBalancePacing` band need attention.
3. **Average margin.** The first handoff wanted about $12-15 a case in phase 3 (today about $22, band in `TestBalanceMarketIsARealRisk` is $12 to $35). With impact biting, check whether the margin still needs lowering at all before touching base prices.
4. **Score padding** (stock valued at plain bid): leave as is, or value stock net of impact.

## 6. Method notes and gotchas

- **Run the report:** `cd lemonade-api && BALANCE_REPORT=1 go test ./internal/domain -run TestBalanceReport -v -count=1`. It prints every bot (survival, capital at days 5 to 90, pacing milestones) and the exploit summary. It takes a few seconds.
- **Bot files:** `internal/domain/balance_test.go` (existing bots, `report`, `exploitSummary`, the `TestBalance*` guard rails) and `internal/domain/exploitbots_test.go` (spammer, thresholder, strict thresholder, hoarder, opportunist, `growFast`, `profitableBatch`, `waiter`). Bots only make real domain calls, so any market change shows up in their results.
- **Capital medians are survivors only.** A bot with 60% bankrupt can show a healthy median at day 60. Always read the bankrupt column next to it. This hid a bug once.
- **Bots must play the new market sensibly.** After impact was added the careful grower and skilled bots first went 99% bankrupt: they bought full batches into rising prices and added capacity that could not pay for itself. They now use `profitableBatch` (size each batch to the last case whose marginal margin covers its cost, counting input asks today and lemonade bids tomorrow after half the sell pressure decays) and only expand when capacity is what limits profit (`capacityBinds`). The spammer is deliberately left price-blind. If a future change makes all bots die, suspect the bots before the game.
- **Bots also need a cash reserve and working-capital limits** (three days of upkeep, at most a quarter of spare cash per dip purchase, expansion only if cash after covers two days of new upkeep plus part of a batch). Without them they bankrupt themselves at factory scale, which is a bot fault.
- **A temporary sweep test was used and deleted.** Recreate it if needed: build a `Config` with `FreeDepth` set for all five resources, plus `ImpactSlope`/`Recovery`, run `grower`, `spammer`, `opportunist` over 100 seeds and print bankrupt share, day 60/90 medians, maxed share, and how often the spammer beats the grower at day 60. Use `capAt` and `median` from `balance_test.go`.
- **Exact-number tests:** changing prices, costs or upkeep breaks a few exact figures asserted in API and domain tests and in README, DESIGN, SPEC and `numeric-tdd.md` section 9. Change knobs in `DefaultConfig()` only; impact knobs are not asserted exactly except in `impact_test.go`, which builds its own config with `impactCfg`.
- **Determinism:** pressure is float64 and decays deterministically; no new randomness. Keep it that way (`TestImpactIsDeterministic`).
- **Do not:** add hard daily caps, widen the spread, add random inventory-destroying events, change bankruptcy rules or forced-sale order, add shared-market rival pressure, or build perishable lemonade (all from `HANDOFF-BALANCE.md` section 4).
- **Layout fixes made on the way** (late-game phone overflows): facilities panel buttons now stack under the capacity text, the stats strip is a 2x2 grid on phones, the best-score chip row wraps. Re-check 390 px after any change that lengthens labels or figures.

## 7. Suggested plan for pass 2

1. Get the answers in section 5.
2. If option a (depth scales with tier): make `FreeDepth` a per-warehouse-level table (config, `DefaultConfig()`, `impact.go` reads the current `WarehouseLevel`), keep the API shape, update `impact_test.go`, run the sweep for slope and the four tier depths, and target: careful bots reach level 3 and a meaningful share max within 120 days, spammer still fails at every tier, day-60 skilled median still inside $6-10k. Log a before and after table in `DECISIONS.md`.
3. Re-run all bots and the guard rails; widen a band only with the user's approval and log it.
4. Update the numbers in `numeric-tdd.md` section 9, `HANDOFF.md`, and the glossary text ("The first 80 cases..." in `shared/help/glossary.ts` mirrors `FreeDepth`; update both together).
5. Real-stack check: drive a factory-scale game by script (setting state directly in the dev database works: `UPDATE games SET capital=..., warehouse_level=4, warehouse_qty=..., inventory=...`), confirm big trades cost visibly more per case, the early game is unchanged, and screenshot the market panel at 1280 and 390 px.
6. Flag "Log candidate:" for `docs/claude/ai-usage-log.md`: the approach of measuring the exploit as numbers with bots, finding that the diagnosis was half wrong (waiting does not pay, lemonade cannot be hoarded), fixing, and re-measuring.

## 8. Out of scope here

Difficulty (cut), rival, perishable lemonade, real authentication, and the unused `GET /api/game/quote` client (decide whether to wire it into a bulk-bar hover or remove it).
