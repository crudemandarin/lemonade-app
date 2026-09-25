# Handoff: late game Goals track (achievements, time-capped leaderboards)

Read `HANDOFF-LATE-GAME-ROADMAP.md` first (waves, shared contracts, branch `late/goals`). Then read `HANDOFF.md`, `HANDOFF-P0.md` sections 0 and 1, `LATE-GAME-DESIGN.md` sections 0, 6.7 and 6.8, and `LATE-GAME-CONTENT.md` section K. Check `git log` and `git status`. Stage explicit paths only. Test first.

**Concurrency:** wave 1, alongside Products A. Stage B can run in any later wave. This track never touches `EndDay` or the commodity type.

**Ask the user first:** open questions 19, 20, 21 and 23 (defaults: cosmetic only; retroactive only where stored data proves it; a small badge count on the leaderboard row; add a "best at day 100" board and keep all-time).

## Stage A: achievements (design phase 1)

### Scope
The framework, plus the achievements marked ★ in content section K (about 45: they need no new systems). The rest (rivals, recipes, upgrades) are added by the tracks that build those systems, as rows in the same table.

### Domain (pure)
- `internal/domain/content/achievements.go`: `AchievementDef{Key, Name, Description, Category, Tier (bronze, silver, gold), Hidden bool, Check Predicate}`.
- `internal/domain/achievements.go`:
  - `Predicate` is a small closed set of typed checks, not free functions per row. Examples: `NetWorthAtLeast(n)`, `DayAtLeast(n)`, `StatAtLeast(stat, n)`, `ProducedInOneDay(n)`, `AllWarehousesFull`, `StockTotalAtLeast(n)`, `SoldDuringEvent(event, product, n)`, `ProfitDuringEvent(event)`, `EveryEventSeen`, `PriceRatioTraded(side, ratio)`, `RunsFinishedAtLeast(n)`, `ClosingCash(min, max)`, `Comeback(low, high)`.
  - `Evaluate(before, after Game, ctx AchievementContext) []string` returns keys that became true. `AchievementContext` holds cross-run data (finished runs count, personal best, leaderboard rank) that the API layer loads.
- Some checks need facts the game does not keep yet: every event seen, the lowest cash this run for "comeback", consecutive days at full production, days without trading, and whether any sale this run paid impact. Add these as **run-scoped counters in `Stats`** (they are persisted already), updated where the facts happen (`EndDay`, `Buy`/`Sell`). They must be cheap and deterministic.

### Storage and API
- New table `achievements (user_id, key, run_id, unlocked_at, PRIMARY KEY (user_id, key))`, created by `AutoMigrate`, plus the memory fake and a store contract test. It is idempotent, so unlocking twice is a no-op.
- Evaluate **after every successful mutation** in the handler layer, inside the same transaction as the save: extend `domain.Effects` with `Unlocked []string`, and have the store insert them there. Return newly unlocked keys in the mutation response as `unlocked: [{key, name, tier}]`, for toasts. This is additive to the view DTO.
- `GET /api/achievements` returns every definition with unlocked state and date, and progress where it is measurable (for example 6,200 of 10,000 produced). Hidden entries return `name: null` and `description: null` until unlocked. This needs `requireUser`. Usernames and Google-linked accounts both work, because it is keyed by `user_id`.
- Leaderboard rows gain `achievements` (the count) if question 21 is yes.
- **Retroactive (question 20):** a one-off idempotent backfill that runs at startup (or as a command) and grants what stored `runs` prove (net worth, days, runs count, personal best). Document what it cannot prove.

### Frontend
- A toast service shows one line per unlock ("Achievement unlocked: Five figures"), stacked, auto-dismissed, with no emoji, announced through `aria-live`.
- `/achievements` page is in the nav menu. It groups achievements by category with a tier label (text, not colour alone) and progress bars, and shows hidden ones as "???". It is one column at 390 px.
- The run detail page (`/runs/:id`) lists the achievements unlocked in that run.

### Tests
- Every predicate: a table test, including the boundary values.
- A content validation test: unique keys, every predicate valid, and hidden entries have a description for after unlock.
- `Evaluate` returns each key only once.
- The transaction test: a failed mutation grants nothing.
- The backfill is idempotent.
- API tests: the list, hidden masking, and the `unlocked` field on mutations.
- Frontend specs: the toast, the page, and progress.
- The balance report is unchanged (achievements do not affect play).

## Stage B: time-capped leaderboards (design phase 7, part)

- Record `net_worth_day_100` on the run (null if the run ended earlier). Set it when day 100 is reached, in the same transaction, through `Effects`. Consider also day 30 and day 60 if the user wants shorter boards.
- `GET /api/scores?board=all_time|day_100` with the same shape. The `/scores` page gets a board toggle.
- Old runs have no snapshot. Leave them off the day-100 board, and say so in the empty state.
- Tests: the snapshot is taken exactly once, runs that end before day 100 are excluded, and the ranking and tie-break match the all-time rules (decision 30).

## Docs
`DECISIONS.md` entries (numbered at merge; see the roadmap): cosmetic achievements, the predicate set, the backfill scope, and the day-100 board. Also update `numeric-tdd.md` (data model, API), README (features), the glossary (achievements), and `PLAN.md`. Add a "Log candidate:" note if a technique stands out.
