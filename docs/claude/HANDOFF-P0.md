# Handoff: P0 items for the major update (slice 9)

For the agent implementing the four P0 items. Read `docs/claude/HANDOFF.md` first (repo state, stack, rules), then this. It is a snapshot: check `git log` and `git status` before trusting details, because other agents share this working tree.

## 0. Ground rules (from `docs/claude/CLAUDE.md`)

- Test first: write failing tests for the acceptance criteria, implement, run everything, commit small imperative messages ending with `Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>`.
- Domain logic is pure (no I/O, no clock); handlers and UI stay thin; the frontend never computes game outcomes.
- **Shared working tree. Never `git add -A`.** Stage explicit paths only. At last check `deploy/**` and `docs/DEMO.md` had uncommitted edits that are not yours; leave them.
- Ambiguity: ask the user, do not silently guess. Log real deviations as new entries in `docs/claude/DECISIONS.md` (append after entry 20). Flag "Log candidate:" moments for `ai-usage-log.md`.
- UI: sentence-case labels with no terminal punctuation, one primary button per screen, whole-dollar money, mobile-friendly (check 390 px, no horizontal overflow), meaning never carried by colour alone.
- Scope: only the four items below. Do not start difficulty, runs, rival, or anything in slices 10+.
- Add the four items as slice 9 (9a to 9d) in `docs/claude/PLAN.md` and tick as you go.

## 1. Where things live (verify these)

Backend (`lemonade-api/`): `internal/domain/{endday,events,config,quotes,actions,types,facilities}.go`; `internal/api/{game,dto,errors}.go`; tests `internal/domain/{domain_test,balance_test}.go`, `internal/api/game_test.go`. Frontend (`lemonade-web/src/app/`): `core/{api.models,api.service,game.store}.ts`, `core/testing/fixtures.ts`, `pages/game/game.component.*`, `pages/game/components/{stats-strip,market-panel,day-report-modal,events-banner}`.

Contract rule: every Go DTO change must be mirrored in `core/api.models.ts` **and** `core/testing/fixtures.ts`. Verify by compiling real API responses against `api.models.ts` with `tsc --strict` (a past technique worth reusing).

Run and test: backend `cd lemonade-api && gofmt -l . && go vet ./... && go test ./...`; frontend `cd lemonade-web && npx ng test --watch=false --browsers=ChromeHeadless && npm run lint && npm run format:check && npm run build`. Full stack: `docker compose up -d --build` (web :4200, api :8080; if the API logs "no such host db", run `docker compose up -d` again).

## 2. P0-1 (9a): lemonade price always on the day report

**Now:** `EndDay` in `internal/domain/endday.go` builds `DayReport.PriceChanges []PriceChange{Resource, Before, After}` and only includes resources whose *effective* price changed. `day-report-modal.component.ts` finds lemonade with `report().priceChanges.find(c => c.resource === 'lemonade')` and renders "Lemonade price $before → $after"; the row is missing when unchanged.

**Change:**
- `EndDay` always emits all five resources in the `Resources` display order (`Before == After` allowed).
- The modal shows lemonade pinned first, then the others; unchanged rows are dimmed with the text "no change" (not colour alone). Rows that changed keep the current arrow/delta styling.
- The JSON shape does not change (`priceChanges` array), so `api.models.ts` needs no field change; update fixtures and any spec that assumed only changed rows.

**Acceptance:** a test where prices are unchanged yields five entries; a test with mixed changes yields five entries with correct before/after; modal spec renders "no change" for an unchanged lemonade price; existing timeline/report assertions still pass. Note: `DayReport` is not persisted yet (it lives in the frontend `GameStore._report` signal); do not add persistence here.

## 3. P0-2 (9b): conflicting events never overlap

**Now:** `Config.Events []EventDef{Key, Name, Description, Duration, Multipliers map[Resource]float64, Excludes []string}`. `eligibleEvents` (in `internal/domain/events.go`) excludes already-active events and pairs blocked by `Excludes`, checked both directions. Only `heat_wave` and `rainy_week` are linked. **Gap:** `holiday` (lemonade ×1.35) can overlap `rainy_week` (×0.75).

**Change:**
- Add a pure `conflicts(a, b EventDef) bool`: true if for any resource one multiplier is > 1 and the other is < 1. `eligibleEvents` treats explicit `Excludes` OR derived conflicts as blocking, both directions, against every currently active event.
- Keep `Excludes` as an explicit override (do not delete the field).
- Spawn logic otherwise unchanged (25% chance, at most one spawn per day, seeded `rng` from `seed ^ day`). Do not change how many random numbers are drawn on the no-conflict path, so existing seeded tests and balance bands stay stable.
- Events already overlapping in saved games simply expire; no migration.

**Acceptance:** a table test asserting `conflicts` for every pair in `DefaultConfig().Events` (expect exactly heat/rain and holiday/rain among current events, plus a synthetic pair); a test that with `rainy_week` active `holiday` is never eligible and vice versa, over many seeds; a test that non-conflicting events (heat wave and holiday) can still co-occur. Add a comment or README note that new events are checked automatically. Update the events line in `README.md` "Tuning the game" and `SPEC.md` if it states the exclusion rule.

## 4. P0-3 (9c): header projection

**Now:** `stats-strip` shows Day, Capital, Upkeep per day, and the End day button. The user wants it to also show how many lemonade will be produced and how much ice will melt at end of day. Existing rules: `produce()` makes `min(ProductionCapacity, stock of each input, free lemonade space)` (recipe 1 lemon + 1 sugar + 1 ice + 1 cup); ice is set to 0 at end of day (`IceMelted = Inventory[Ice]`), and the order is produce, then melt.

**Change:**
- Extract a small pure helper (e.g. `produceQty(g, cfg) (qty int, limitedBy string)`) used by both `produce()` and a new `PreviewEndDay(g, cfg) Projection{LemonadeToProduce, IceToMelt, LimitedBy}`. One source of truth so preview and reality cannot drift. Ice consumed by production is used before the rest melts, so `IceToMelt = Inventory[Ice] - LemonadeToProduce`.
- `LimitedBy` is one of `production`, `lemon`, `sugar`, `ice`, `cup`, `space`, or empty when nothing limits output (also empty if nothing would be produced because capacity is 0).
- DTO: add `projection {lemonadeToProduce, iceToMelt, limitedBy}` to the game view (`dto.go`, `api.models.ts`, `fixtures.ts`). Computed on every view, so it stays correct after any buy, sell, or expand.
- UI: `stats-strip` adds "Makes N lemonade" and "M ice will melt" (hide the ice figure at 0 or show 0 dimmed; decide, and keep it compact at 390 px). Tooltip or `title` names the limiting factor in words, e.g. "Limited by sugar". Also add an `aria-label` reading.

**Acceptance:** a property-style test (many random inventories/levels) asserting the preview equals the actual `DayReport.Produced` and `IceMelted` from `EndDay` on a clone of the same game; edge cases: zero stock, full lemonade warehouse, each input as the bottleneck, ice larger than production; handler test that the view includes `projection`; strip spec for both figures and the tooltip text. Preview must not mutate the game.

## 5. P0-4 (9d): bulk trade bar

**Now:** each `market-panel` row has its own quantity/buy/sell controls; `POST /api/game/buy` and `/sell` take `{resource, qty}` and error with `insufficient_funds`, `insufficient_stock`, or `capacity_exceeded` (409) when the full quantity cannot be done.

**Assumed reading (confirmed by the user as the working assumption; if in doubt, ask):** a bar at the top of the markets panel selects the amount: `All`, `10`, `50`, `100`, labelled as Buy/Sell pairs ("Buy 10 / Sell 10", and so on). Each commodity row then has only Buy and Sell buttons that use the selected amount. It is NOT a set of buttons that act on every commodity at once.

**Change:**
- Backend: optional boolean `clamp` on buy and sell requests. With `clamp`, Buy trades `min(qty, affordable at ask, free warehouse space)` and Sell trades `min(qty, stock)`. If the clamped amount is 0, return the normal error for the binding limit. Without `clamp`, behaviour is unchanged. "All" is sent as a very large qty with `clamp: true` (or a `qty` cap constant), never computed in the UI.
- Response is still the updated game view; also return how many were actually traded if easy (optional; if added, mirror in `api.models.ts`).
- Frontend: a quantity selector component in `market-panel`, rows simplified to Buy and Sell, buttons disabled per row when the action is impossible (no stock, no cash), selection persisted in `localStorage` (wrap access in try/catch), sensible default (10). Works at 390 px (segmented control wraps or scrolls without horizontal page overflow), keyboard accessible (`role="radiogroup"`), and the offline banner still disables actions.
- Timeline: same-day repeated trades of the same resource already merge, so bulk trades should not bloat it.

**Acceptance:** domain tests for clamped buy (limited by cash, by space, by both, zero result error) and clamped sell; unchanged behaviour without `clamp`; handler test for the flag; `game.store` spec passes `clamp`; market-panel spec for selector state, persistence, and row buttons. Update `README`/`numeric-tdd.md` API section for the new field.

## 6. Cross-cutting checklist

- Fixtures and `api.models.ts` updated together; `tsc --strict` check against a real response.
- `gofmt`, `go vet`, all Go tests green; frontend tests, lint, format check, build green.
- Real-stack check: compose up, play a few days by script or browser, confirm the day report shows all five prices, the projection matches the actual end-of-day numbers, and bulk buy/sell clamps correctly; screenshot at 1280 and 390 px.
- Balance tests (`TestBalance*`) must still pass unchanged: none of the P0 items should alter economics.
- Docs: `PLAN.md` slice 9 ticked, `DECISIONS.md` entries for (a) derived event conflicts, (b) `clamp` semantics, (c) projection sharing `produceQty`; brief mentions in `README.md`.
- Commit one item per commit, explicit paths only.

## 7. Out of scope but coming (do not build)

Slices 10 to 15: cost basis, all-commodity price chart, facility resale (buildings only; no level downgrade, no upgrade items), difficulty, runs, give up, scores, rival, glossary. Full plan: `/Users/nyko/.claude/plans/here-are-notes-for-dreamy-mango.md` (copy into `docs/claude/` if the next agent cannot see it).
