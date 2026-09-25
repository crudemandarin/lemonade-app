# Handoff: balance pass (fixing the buy-everything, sell-everything exploit)

For the agent who implements this. Read `HANDOFF.md` first (repo, stack, rules), and `HANDOFF-P0.md` §0 (ground rules) and §1 (commands). It is a snapshot: check `git log` and `git status` first, because other agents edit this working tree and slices from the other handoffs (P0, A, B, C) may have landed or be in flight. **Never `git add -A`; stage explicit paths.** Test first, domain stays pure, log deviations in `DECISIONS.md` (append after the last entry), update `PLAN.md`, and ask the user when a requirement is ambiguous.

**Status: design only. Nothing here is implemented.** The user has signed off on the decisions in §5. Numbers below are starting points to be tuned with simulation.

## 1. The problem

Players can max out inputs, produce, sell every lemonade, and repeat daily for a near-guaranteed win. This was observed by the user; the diagnosis below comes from reading the code and running `BALANCE_REPORT=1 go test ./internal/domain -run TestBalanceReport -v` (bots, not real players, so confirm with the new exploit bots in phase 0).

**Why it works**
1. **Fat margin.** Average lemonade margin is about $22 per case after buying all four inputs at ask and selling at bid (median $20, p5 −$7, p95 $59; unprofitable on about 11% of days). Inputs cost about $55 at ask against about $81 lemonade bid, roughly 40% return. The 10% spread (about 18–20% round trip) is small next to the recipe markup.
2. **Unlimited liquidity.** Buy and sell have no depth and no price impact (`Quotes` in `quotes.go`, `Buy` and `Sell` in `actions.go`). Selling 80 cases gets the same bid as 1, so profit = margin × production rate.
3. **Compounding.** Profit buys production capacity, raising volume at a constant margin. Current report: the "grower" bot goes $2.4k (day 30) to $15.9k (day 45) to $29.9k (day 60); "everything maxed" at a median of day 65. A Factory costs $280/day upkeep but earns about 80 × $22 ≈ $1,760/day.
4. **Free waiting.** Only ice perishes. Lemons, sugar, cups and lemonade keep at no holding cost, and prices mean-revert to a known base (`walk` in `market.go`: revert 0.15, σ 0.12). A player can buy below base, sell above base, and skip unprofitable days: a free option with no downside. Per-case daily volatility (about ±$11) is small next to the margin.

## 2. Goals and guard rails (fun first)

Goals: volume and hoarding become real decisions, the best strategy still rewards skill (timing, events, expansion order), and reaching "maxed" takes longer than about day 65 without feeling like a grind.

Must not change: early game feel (a starter player producing about 10 cases a day never notices a new mechanic), whole-dollar prices, one-primary-button UI, server as source of truth (the frontend previews via API, never computes outcomes), determinism from `seed ^ day` (no new unseeded randomness), and Normal difficulty balance until the user signs off on new numbers.

Targets at Normal (**approved by the user as starting targets; revisit with the user if the phase 0 baseline shows they are unreachable, and log any band you widen in DECISIONS**):
- Max-volume "spammer" bot: bankrupt or under $X at day 90 in at least 25% of seeds; never beats the best skilled bot.
- Best skilled bot ("thresholder"): median day-60 capital roughly $6–10k (today about $30k), and beats the spammer by 1.5x to 5x (skill matters, is not mandatory).
- "Everything maxed" median moves from about day 65 to about day 90+, with at least 50% still maxing within 120 days.
- Existing guard rails keep passing (careful players mostly survive; careless, sloppy, idle lose; market is a real risk). Read the thresholds in `balance_test.go` first; widen bands only with the user's approval and log it.

## 3. Work plan (phases with gates)

### Phase 0: exploit bots and baseline (no game change, commit alone)

In `internal/domain/balance_test.go` (existing pieces: `player`, `owner`, `grower`, `sloppy`, `careless`, `idle`, `report`, `marginStats`, `shareDead`, the `TestBalance*` tests) add bots that drive real domain calls (`Buy`, `Sell`, `Expand`, `Upgrade`, `EndDay`), like the current ones:
- **spammer:** every day buy inputs to fill production, sell all lemonade next day regardless of price, expand whenever affordable.
- **thresholder:** buy inputs only when ask is at most 0.9 × base, sell lemonade only when bid is at least 0.95 × base (else hold if warehouse space allows), skip days with negative margin, expand aggressively. This is the free-option player.
- **hoarder:** fill warehouses during dips, sell only on spikes (at least 1.2 × base, e.g. events).

Extend `TestBalanceReport` to print all bots (survival, day-30/45/60/90 capital, maxed day). Record the baseline table in the handoff PR and in `numeric-tdd.md` §9. Gate: show the user the baseline before phase 1 (the targets in §2 are already approved; this is to confirm the exploit is real).

### Phase 1: market depth and price impact (fix A, main fix)

Model, all in `internal/domain`:
- Per resource, count units the player **bought** and **sold** (separate pressures, so buying only raises ask and selling only lowers bid; this prevents any round-trip arbitrage). Store as `Game.BuyPressure`, `Game.SellPressure` (`map[Resource]float64`), JSONB columns in `store/postgres.go` (`gameRow`), copied in `Clone()` in `types.go` and in the `memory.go` fake. Old saves load as zero.
- Unit price for the k-th unit today: `ask × (1 + slope × max(0, pressure + k − FreeDepth))` for buys, `bid × (1 − same)` for sells, each unit rounded (ask up, bid down, floor $1, and **impact cap** so a move never exceeds ±`ImpactCap`, e.g. 60%). Total cost is the sum over units (loop or closed form, integer dollars).
- Overnight recovery in `EndDay`: `pressure *= (1 − Recovery)`.
- Config knobs (in `DefaultConfig()`, documented in README "Tuning the game"): `FreeDepth` per resource (start: 40 cases for each), `ImpactSlope` (start 0.015 per case beyond free), `Recovery` (start 0.5/day), `ImpactCap` (0.6). Choose `FreeDepth` so players at production levels 1–2 (10–20 cases a day) never hit it; the pressure appears only at Factory scale.
- Touch points: `Buy`, `Sell` (impact-aware totals), the P0 `clamp` logic (bulk "buy N" must clamp by impact-aware affordability and space), `sellStockToCover` in `bankruptcy.go` (forced sales now sell more cases to raise the same cash; still deterministic), cost basis if it has landed (uses actual cost paid), `NetWorth` if it has landed (value stock at plain bid, document that it ignores impact), the rival (uses the same domain calls, so it inherits impact on its own copy of the state; its pressure must be its own, not shared with the player).
- **Invariants (property tests):** buy N then sell N the same day never profits; pressure is never negative and recovers toward 0; cost with impact ≥ cost without; a trade within free depth costs exactly ask (or bid) times quantity (existing tests keep passing unchanged); results are deterministic; clamped bulk buys never overspend.
- API/UI (thin): quote DTO `bid`/`ask` become the **marginal price of the next unit**, plus `freeDepthLeft` per side. Add `GET /api/game/quote?resource=&side=&qty=` returning total, average price and slippage percent, used by the bulk bar hover or confirmation text ("50 lemonade: avg $78, slippage 4%"). Mirror DTOs in `core/api.models.ts` and `core/testing/fixtures.ts` (verify with `tsc --strict` against a real response). Market row shows a subtle "price impact" hint when pressure is above zero (text plus icon, not colour alone).
- Impact applies to **buying inputs as well as selling lemonade** (all five resources).
- Gate: run all bots. If the targets in §2 are not met, tune phase 3 knobs first; if still not met, report to the user with the numbers rather than adding mechanics on your own.

### Phase 2: perishable lemonade (fix B): NOT in scope

The user decided against perishable lemonade. Do not build it. If the targets are still unmet after phases 1 and 3, report to the user; do not add spoilage as a fallback. (Kept as a Future work note in `PLAN.md`.)

### Phase 3: retune (fix C)

Only after A. With bots: adjust `FreeDepth`, `ImpactSlope`, `Recovery`, average margin (target about $12–15 per case, via base prices), and big-tier upkeep. Rules: change one knob at a time, rerun the report, keep a table of before and after in DECISIONS. Update exact figures asserted in domain and API tests and in README, DESIGN, SPEC, `numeric-tdd.md` §9 (the earlier handoff notes several exact-number tests break when prices or costs change). If the difficulty slice (`Config.ForDifficulty`) exists, add depth and recovery scaling per level and rerun bots per level; otherwise leave a note for that slice.

### Phase 4: UX, docs, verification

- UI copy and glossary: add "market depth" and "price impact" to `/help` if handoff C has landed; otherwise leave a Future-work note in `PLAN.md`.
- Docs: `README.md` "Tuning the game" and API list, `numeric-tdd.md` (data model, API, §9 balance tables), `DECISIONS.md` (depth model, separate buy and sell pressure, marginal-price quotes, targets and results), `PLAN.md`. `docs/DEMO.md` has another agent's edits, so coordinate with the user before touching. Flag "Log candidate:" for the exploit-bot approach (measure the exploit as numbers, then fix, then re-measure).
- Verification: `gofmt -l . && go vet ./... && go test ./...` (including the real-Postgres store test with an old-shape row), balance report for all bots, frontend tests, lint, format check, build, then a real-stack run: drive a factory-scale game by script and confirm selling 80 cases visibly costs more per case than selling 10, and the small-scale early game is unchanged. Check 1280 and 390 px.

## 4. Things not to do

No hard daily caps (they feel arbitrary), no wider spread (punishes small players), no random inventory-destroying events, no changes to bankruptcy rules or forced-sale order, no shared-market rival pressure (the rival stays race-only), and no unrelated refactors. If you find the diagnosis wrong in phase 0 (for example the spammer does not win), stop and report to the user with the numbers before building.

## 5. Decisions (from the user)

1. **Targets and bands:** the §2 targets are approved. Widening an existing balance band is allowed if needed, but log it in DECISIONS with before and after numbers.
2. **Perishable lemonade:** no. Phase 2 is out of scope.
3. **Impact on inputs:** yes. Price impact applies to buying inputs and selling lemonade (and selling raw inputs).
4. **Valuation:** net worth and score count stock at the **plain bid**, ignoring price impact. Document this simplification (README and glossary) and note in DECISIONS that a player can pad the score by ending a run holding a large stock; if that shows up on the leaderboard, revisit.
