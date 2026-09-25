# Handoff A: economy state, facility resale, give up, day report history

Covers table items **7, 5, 6, then 10, 11** (recommended build order, one commit per item). Land the P0 slice (`HANDOFF-P0.md`) first: this work builds on its day report (all five prices always present) and header. Read `HANDOFF.md` and `HANDOFF-P0.md` §0 (ground rules), §1 (file map, commands) first. Snapshot: check `git log` and `git status`; other agents share the tree, so stage explicit paths only.

Add these as slices 10 and 11 in `PLAN.md` and tick as you go. Every DTO change is mirrored in `core/api.models.ts` and `core/testing/fixtures.ts` (verify with the `tsc --strict` check).

**Old saves must keep loading.** New `games` columns are added by `AutoMigrate`; a row from before the update loads with zero values, and the API/domain fills sensible defaults (see each item). Add a Postgres round-trip test with an old-shape row (`DATABASE_URL="postgres://admin:password@localhost:5432/sample?sslmode=disable" go test ./internal/store`), following how `toTimelineDTOs` handles a game saved before the timeline existed.

## Item 7: sell facilities (build first)

**Rule:** only building **quantity** can be sold. Levels and upgrades are never sold or downgraded.
- `POST /api/game/facilities/:kind/sell` (`{resource}` for warehouse), returns the game view.
- Proceeds = `floor(ResaleRate × BuildCost[level])`, `ResaleRate` = 0.5 in `Config` (`config.go`, document it in README "Tuning the game").
- Guards, all 409 with a clear message: keep at least 1 production building and 1 warehouse per resource (`min_facility`); remaining capacity must still hold current stock (`stock_exceeds_capacity`, message "sell N cases first"). No silent liquidation.
- Upkeep drops immediately. New `Stats` counter (`FacilitiesSold`, `FacilityProceeds`). Timeline records a point; add a new marker kind for "sold" in `timeline-layout.ts` (`markerPoints`, `describePoint`) that is distinguishable by shape, not colour (existing: hollow circle buy, filled sell, square built, diamond upgraded).
- View adds per-facility `sellValue` and `canSell` (with a reason code when false). `facilities-panel` gets a Sell button per building group with a confirm dialog (it loses money by design). One primary button per screen still holds: use a secondary style.
- New pure helper `FacilityResaleValue(g, cfg)` (all buildings × per-building resale) for net worth (item 10).
- **Exploit invariant test:** for every tier path, resale of a building < cash spent to get a building to that level (build at L1, then upgrade shares). Sanity math: Production L2: 500 + 1000 = 1,500 spent vs 750 resale. If a config change breaks it, the test must fail.

## Item 5: average purchase price

- New `Game.CostBasis map[Resource]int` (total dollars held), weighted average, not FIFO. Add to `Clone()` in `types.go`, `gameRow` JSONB `cost_basis` in `store/postgres.go`, and the `memory.go` fake.
- Buy: basis += cost. Sell: basis -= `round(basis × sold / held)`; when selling the last unit, basis becomes 0 exactly (no drift). Production consumes inputs at their average cost and adds lemonade basis equal to the sum of consumed input costs (so lemonade avg cost = real cost to make). Ice melt and `sellStockToCover` (forced sales) reduce basis the same way. Hook points: `Buy`, `Sell` in `actions.go`, `produce()` and melt in `endday.go`, `bankruptcy.go`.
- Invariants tested: basis never negative; stock 0 means basis 0; production conserves cost (inputs' basis removed equals lemonade basis added); rounding never creates or loses more than $1 per operation.
- **Old saves:** seed basis on load as `qty × current price` (approximation; document in DECISIONS).
- DTO: per-resource `avgCost` (0 when none held). Market row shows "avg $X" and the unrealized gain versus current **bid** (text with sign, plus colour as a second cue, never colour alone). Lemonade row shows cost to make.

## Item 6: all-commodities price chart

- Persist a daily `PriceLog []{day int, prices [5]int, events []string}` (effective prices after the market tick, as the player sees them, plus active event keys that day). Append a point at game start (day 1) and at the end of every `EndDay` after the tick. About 60 bytes/day; no cap needed. Column `price_log` JSONB. **Old saves:** seed with today's prices; the chart starts from the update day.
- DTO adds `priceLog`. `timeline-charts` gets a **third chart** on the shared x-axis: one line per commodity, fixed per-resource colours (already validated for light and dark with the dataviz palette validator; reuse them), legend toggles, shared crosshair, keyboard nav, touch scrub, and the "view as table" twin including the new chart.
- Lemonade ($90) dwarfs the others, so add a `$` / `% of base` toggle (default `$`; base prices come from the game view or `Config.BasePrice`). Event days get shaded bands, event name in the hover.
- Reuse `timeline-layout.ts` helpers (`placePoints`, `niceTicks`, `dayTicks`, `stepPath`, `nearestIndex`); add pure helpers there with unit tests rather than new plumbing. The chart also renders full size on the game-over report.
- Look at `docs/claude/ai-usage-log.md` on the two-chart decision: this adds a third chart, not a dual axis. Keep one series set per chart.

## Foundation for items 10 and 11 (build after 7, 5, 6)

1. **Run identity:** `Game.RunID` (uuid) created at new game; empty on old saves, generated on their first mutation.
2. **Atomic effects:** change `Repository.Mutate(ctx, userID, fn func(*domain.Game) error)` to `fn` returning `Effects{Report *DayReport, Finished *RunRecord}`; the store persists them in the **same transaction** (`SELECT … FOR UPDATE` unchanged) and the in-memory fake mirrors it. Update every caller.
3. **Tables (AutoMigrate):** `day_reports(run_id, day, payload JSONB, PK(run_id, day))` and `runs(id, user_id, run_id unique, difficulty int default 3, days, score, net_worth, capital, ended_by, rival_result null, timeline JSONB, stats JSONB, price_log JSONB, created_at)` with indexes `runs(difficulty, score desc)` and `runs(user_id, created_at desc)`. Do not add endpoints for scores here (handoff B).
4. **`NetWorth(g, cfg)`** (pure, `networth.go`) = capital + Σ stock × current bid + `FacilityResaleValue`. Shown in the header as a secondary figure; returned in the view.
5. **Score:** `score = netWorth + Config.ScorePerDay × daysSurvived`. **Confirm with the user before merging: proposed ScorePerDay = $50 and net worth (not raw capital).** Put `ScorePerDay` in `Config` and README.

## Item 10: give up

- `POST /api/game/give-up` sets status `gave_up`, produces the run record (Effects.Finished), returns the view. Verify `Buy`, `Sell`, `Expand`, `Upgrade`, `EndDay` reject anything not `active` with `ErrGameOver` (check they test `!= StatusActive`, not `== StatusBankrupt`).
- Bankruptcy also emits a run record, in the same `EndDay` transaction that flips status (the game ends on the day it is lost; day does not advance).
- `POST /api/game/new` while the run is active returns 409 `run_active`; the UI routes the player through Give up. (Behaviour change, log in DECISIONS.) Check where "New game" is currently offered (home page, game-over) and fix flows and specs.
- UI: secondary, destructive-styled "Give up" with a confirm dialog on the game page (one primary button per screen: End day). Result screen reuses `game-over` with title "You called it on day N", net worth breakdown (cash, stock, facilities), final score, and the full-size charts.
- Tests: give up records exactly one run, is idempotent-safe (second call returns 409 `game_over`), score matches formula, net worth counts stock at bid, all mutations after give up return `game_over`.

## Item 11: old day reports

- Every `EndDay` returns `Effects.Report`, stored in `day_reports`. **"Game time" means game day, not wall clock**: there is no real-time tick.
- `GET /api/game/reports` returns a light list (day, produced, capital before/after, new/expired events, bankrupt flag), `GET /api/game/reports/:day` returns one full `DayReport`. Both take an optional `runId` for a finished run of the same user (used by handoff B); default is the current run. Not loaded on normal mutations (the game row must not grow).
- UI: a "Past days" button on the game page opens a drawer with a day stepper (previous, next, jump to day) showing the report. Extract the report body from `day-report-modal` into a reusable component so both use it. History starts at deploy time for existing runs; say so in the empty state.
- Tests: append on every end-day, ordering, unknown day is 404, another user's run is 404, an old-save game with no rows returns an empty list.

## Acceptance for the whole handoff

`gofmt -l .`, `go vet ./...`, `go test ./...` (including the real-Postgres store test), balance tests unchanged (nothing here changes economics), frontend tests, lint, format check, build. Real-stack check: play a run by script, sell a building, watch avg cost change, view the third chart at 1280 and 390 px, give up, page back through past days. Docs: DECISIONS entries (weighted-average cost, resale ratio, run identity and atomic effects, `run_active`, score formula), `numeric-tdd.md` data model and API, README tuning table, PLAN ticks. `docs/DEMO.md` has another agent's edits; coordinate before touching.

## Out of scope (other handoffs)

Scores and record pages (handoff B), glossary (handoff C), difficulty, rival, upgrade items, level downgrade.
