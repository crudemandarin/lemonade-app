# Handoff B: personal and global high scores, record history

Covers table items **12** (personal high score and record history) and **13** (global high scores). **Prerequisite: handoff A must be merged** (`HANDOFF-A-economy-and-runs.md`): it creates the `runs` and `day_reports` tables, `Effects.Finished`, `NetWorth`, the score formula, the give-up flow, and the reports endpoints. Verify each exists before starting (`git log`, `runs` in `store/postgres.go`). Read `HANDOFF.md` and `HANDOFF-P0.md` §0 (ground rules) and §1 (commands). Stage explicit paths only.

Add this as slice 13 in `PLAN.md`; DTO changes mirror into `core/api.models.ts` and `core/testing/fixtures.ts`.

## Difficulty is not in this handoff

The difficulty slider (table item 9) is a separate slice, not scheduled yet. `runs.difficulty` already exists with default 3 (Normal). Build boards that **accept** a `difficulty` filter now (default 3, validated 1 to 5), and hide the filter control in the UI while only one level exists in the data. When difficulty ships, the filter appears with no backend change. Recommended and assumed: **separate boards per difficulty**. If the user decides on one board instead, that needs a difficulty multiplier; ask before doing that.

## API

| Route | Purpose |
|---|---|
| `GET /api/scores?difficulty=3&limit=20` | Global board. One row per user (their **best** run), ranked by score, ties broken by earlier `created_at`. Row: rank, username, score, days, netWorth, difficulty, rivalResult (nullable), createdAt. Default limit 20, max 100. Also returns `me`: the caller's own best row and rank even when outside the top N (null if no runs). |
| `GET /api/runs?difficulty=` | The caller's record history, newest first, with an `isBest` flag. |
| `GET /api/runs/:id` | One of the caller's runs in full: score breakdown, stats, timeline, price log, difficulty, rival result, and the report index (the same light list as `GET /api/game/reports`). **Owner only**: another user's id is 404. Global board rows are summary-only and not clickable (decision to confirm: making others' runs viewable is a privacy and scope change). |
| game view | adds `best {score, days}` for the caller at the current difficulty (single indexed query). |

Reports for a finished run come from handoff A's `GET /api/game/reports?runId=` and `/reports/:day?runId=`. The 401 handling, `X-Username` header, and `{error, message}` shape are unchanged.

## Frontend

- **Routes:** `/scores` (tabs **Global | Mine**, difficulty filter hidden until difficulty exists, personal best highlighted, your row shown even when outside top N via `me`) and `/runs/:id` behind `authGuard`. Add "Scores" to `nav-bar`.
- **Run detail** reuses existing pieces: the stats summary and `timeline-charts` at `size="large"` with capital, stock, **and the price chart** (from `priceLog`), plus the report stepper component extracted in handoff A. Clicking a personal record row goes here, so timelines and everything are visible, as the user asked. Show score breakdown (net worth + days × per-day weight) and how the run ended (bankrupt or gave up).
- **Game page:** a "Best: X" chip in the header (from `best`), and on the result screen a "New personal best" callout when the finished run is your top score.
- Table markup with real `<table>` semantics; mobile at 390 px collapses to stacked rows with no horizontal page overflow. Loading, empty ("no finished runs yet"), and error states for both tabs. Sentence-case copy, no emoji, one primary button per screen.
- New `ScoresService` in `core/` calling `ApiService`; keep `GameStore` the only owner of game state, boards live in their own page state.

## Rules and gotchas

- Usernames are public on the global board and auth is username-only (anyone can type anyone's name). Add this to the documented known limitations in README and `numeric-tdd.md`.
- Only finished runs (`bankrupt`, `gave_up`) appear. An active run never does.
- Queries use the handoff A indexes: `runs(difficulty, score desc)` and `runs(user_id, created_at desc)`. Best-per-user ranking: use a window function or `DISTINCT ON (user_id)`; add a test with equal scores and multiple runs per user against real Postgres.
- The in-memory fake in `store/memory.go` must implement the same ranking and tie-break so handler tests match real behaviour (the store contract test runs against both).
- Runs from before this update have no rival result (null); render a dash.

## Tests

- Store contract: ranking, tie-break, best-per-user, `me` rank outside top N, per-difficulty separation, owner-only fetch.
- Handlers: limit clamping, invalid difficulty 400, another user's run 404, unauthenticated 401.
- Frontend: scores page tabs, personal-best highlight, empty state, run detail renders charts and stepper from a fixture, "New personal best" callout logic, nav entry.
- Real-stack check: finish several runs (bankrupt and give up) as two usernames, confirm boards, ranks, and that clicking a personal record shows all charts at 1280 and 390 px.

## Docs

`DECISIONS.md` entries (one row per user on the board, owner-only run detail, difficulty filter forward-compat), `numeric-tdd.md` API and data model, README known limitations, `PLAN.md` tick. Coordinate on `docs/DEMO.md` (another agent's uncommitted edits).

## Out of scope

Difficulty slider, rival (a nullable `rivalResult` is only displayed), glossary (handoff C), viewing others' runs, real authentication.
