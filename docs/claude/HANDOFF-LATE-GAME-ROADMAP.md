# Late game roadmap: phases 1 to 7 grouped into tracks

Read this before any late-game track handoff. The design is `LATE-GAME-DESIGN.md` (decisions in its section 0, open questions in section 11), and the content is `LATE-GAME-CONTENT.md`. Phase 0 is **done** (`6fee996`, decision 36): depth grows with warehouse level (80/280/480/960), impact is relative to depth, and big tiers are cheaper per case. The careful grower now reaches $50.5k at day 90, maxes out around day 93 (64% within 120 days), and 9% go bankrupt.

## 1. Tracks

The seven phases group into four tracks. Each track has one handoff and one owner (agent) at a time, and its stages land in order.

| Track | Handoff | Stages (design phases) | Needs before it starts |
|---|---|---|---|
| **Goals** | `HANDOFF-LATE-GAME-GOALS.md` | A: achievements (phase 1). B: time-capped leaderboards (phase 7, part) | nothing (phase 0 done) |
| **Products** | `HANDOFF-LATE-GAME-PRODUCTS.md` | A: data-driven commodities, no new content (phase 4). B: recipes, new commodities, storage classes, perishables (phase 5). C: contracts (phase 7, part) | A: nothing. B: Products A **and** Upgrades A (freezer, oven, carbonator, cold room are upgrades). C: Products B |
| **Upgrades** | `HANDOFF-LATE-GAME-UPGRADES.md` | A: upgrade framework and the first about 20 upgrades (phase 2). B: managers and fast-forward (phase 6) | A: Products A merged (so effects are keyed by catalog commodities, not the old enum). B: Products B (the plant manager runs the production plan) |
| **Empire** | `HANDOFF-LATE-GAME-EMPIRE.md` | A: territories and rivals (phase 3). B: victory condition and economic cycles (phase 7, part) | A: Products A merged. B: Empire A |

Why these groupings: each track owns one set of domain files and one set of screens, so a single agent keeps its context. Stages that depend on each other stay in the same track. Phase 7 is split up by what each piece depends on.

## 2. Waves: what can run at the same time

| Wave | Run concurrently | Why it is safe | Gate to the next wave |
|---|---|---|---|
| **1** | **Goals A** and **Products A** | Achievements live in a new table, a new predicate file, a new page and one hook after each mutation. They read stats and net worth, not the commodity type. Products A is a mechanical refactor. The only overlap is `dto.go`, `api.models.ts` and `fixtures.ts` (additive on the Goals side), which is a textual merge, not a logic conflict. | Products A merged, with byte-identical outcomes for existing seeds |
| **2** | **Upgrades A** and **Empire A** (and Goals B if not done) | Different mechanics and different files (`upgrades.go` and `territories.go`/`rivals.go`, and separate pages). Coupling rule: Upgrades A builds the effect framework and every upgrade **not** tied to territories. Upgrades that touch presence, hubs or rivals (`roadside_sign`, `billboard`, `radio_spot`, `tv_campaign`, `global_brand`, `delivery_fleet`, `rival_intel`, `pr_team`) are added by **Empire A** using the framework, once Upgrades A has merged its framework commit. Empire A lands the territory mechanics first, then these rows. | Both merged; balance report re-run on the integration branch |
| **3** | **Products B** and **Empire B** | Recipes touch production, storage and the market panel. Victory and cycles touch rivals, events and the result screen. | Products B merged |
| **4** | **Upgrades B** and **Products C** | Managers and fast-forward touch the start of end-of-day and a settings UI. Contracts touch a new contracts module and a panel. The overlap is `EndDay` step order only (see section 3). | Epic done |

**Do not run Products A alongside anything except Goals.** It touches nearly every file on both sides, so any other track would rebase through all of it. Everything else in a wave may run fully in parallel.

## 3. Shared contracts every track follows

**Branches.** One branch per track (`late/goals`, `late/products`, `late/upgrades`, `late/empire`), cut from and merged back into an integration branch (`late/main`, cut from `feature/slices-9-13` or wherever phase 0 lives; check `git log`). Merge small and often. Rebase before merging, and run everything after rebasing. Shared working trees: **never `git add -A`**, stage explicit paths, and never commit another agent's files.

**End-of-day step order.** It is fixed now so that tracks insert steps without negotiating. Each step is a named function, owned by the track in brackets, and a no-op until that track lands:
1. managers act [Upgrades B]
2. produce by plan [Products B; today `produce`]
3. freezer rotation [Upgrades A]
4. ice melt, fresh-goods spoilage [Products B]
5. settle upkeep: facilities, hubs [Empire A], upgrades and managers [Upgrades]
6. record timeline and report, then the bankruptcy check (unchanged)
7. advance the day, forget pressure (unchanged)
8. events, then economic cycles [Empire B]
9. market tick for every commodity [Products A]
10. rivals tick [Empire A]
11. contract deadlines [Products C]
12. price log (unchanged)

(Products A swapped 8 and 9 from the first draft: the event roll and the walk share one random stream, events first, and existing seeds must replay exactly. The step functions are in `endday.go`.)

Achievements are evaluated **after** the whole mutation, in the API layer [Goals A], not inside `EndDay`. Add the skeleton of this order (empty named functions with comments) in whichever track merges first.

**Game state.** Each track adds its own fields to `domain.Game`, `Clone()`, `gameRow` (JSONB) and `store/memory.go`, with old-save defaults and a round-trip Postgres test (`DATABASE_URL=... go test ./internal/store`) against an old-shape row. Do not reorganize fields another track added.

**Config.** Content tables go in `internal/domain/content/` as one file per table (for example `upgrades.go`, `rivals.go`, `recipes.go`, `achievements.go`), with one validation test per table (unique keys, references resolve, numbers in sane ranges). Knobs stay in `DefaultConfig()`.

**Randomness.** Each system that rolls gets its own salt (`rand(seed ^ day ^ salt)`), declared as constants in one file (`internal/domain/salts.go`). Adding a system must never change market prices for an existing seed. Each track adds a test for that.

**API contract.** Every DTO change is mirrored in `lemonade-web/src/app/core/api.models.ts` and `core/testing/fixtures.ts` in the same commit, and checked with `tsc --strict` against a real response. The game view is getting large, so new heavy data gets its own endpoint (`/api/game/rivals`, `/api/game/upgrades`, `/api/achievements`) instead of growing `GET /api/game`.

**Docs.** `DECISIONS.md` numbering collides when tracks run concurrently. Write entries under a heading like `## N. <title>` with `N` left as `TBD`, and number them when merging into the integration branch. Each track owns its own `PLAN.md` slice block. The glossary (`shared/help/glossary.ts`) gets each track's terms, and its numbers must mirror `Config`.

**Balance.** Every track re-runs `BALANCE_REPORT=1 go test ./internal/domain -run TestBalanceReport -v -count=1` before merging. No existing guard rail band is widened without the user's approval. `TestEachLevelPaysMoreThanTheLast` must keep passing. The careful grower is the control that ignores new systems. It must keep growing (it may grow more slowly than bots that use them).

**UI.** Mobile first (390 px, no horizontal scroll), one primary button per screen (End day on the game page), sentence case, whole dollars with compact formatting for large values (`$1.2M`, added by whichever track first shows a value above $1M; it is likely Empire), and new nav entries go in the existing nav bar menu.

## 4. Open questions each track must settle with the user first

From `LATE-GAME-DESIGN.md` section 11. The recommendation there is the default, but **ask before building**:
- **Goals:** 19 (cosmetic or rewards), 20 (retroactive), 21 (badge on the leaderboard), 23 (time-capped boards).
- **Products:** 10 is settled (global prices), plus 16 (launch set), 17 (unlock method), 18 (byproducts), 30 (compact money), 31 (timeline downsampling).
- **Upgrades:** 12 (upgrades cannot be sold), 13 (upkeep), 26 (automation trades for the player), 29 (loans: `credit_line` stays out unless approved).
- **Empire:** 6 (how adversarial), 7 (hostile takeovers), 8 (name tone), 9 (rivals fold and merge), 11 (facility pool), 22 (net worth of acquisitions), 27 (economic cycles).

## 5. User answers (2026-09-25): every recommendation accepted

The user accepted every recommendation in `LATE-GAME-DESIGN.md` section 11 and every default in the track handoffs. **Do not ask again; build to these.**
- **Goals:** 19 cosmetic only (plus badges). 20 retroactive only where stored runs prove it. 21 achievement count badge on leaderboard rows. 23 add a "best net worth by day 100" board, keep all-time.
- **Products:** 16 launch set lime, mint, honey, black tea, strawberry, plus flour, butter, eggs (bakery); recipes limeade, mint lemonade, honey lemonade, Arnold Palmer, strawberry lemonade, lemon bars. 17 recipes unlock by era-gated purchase. 18 byproducts yes, as an upgrade. 30 compact money (`$1.2M`) yes, exact values in tooltips. 31 downsample the timeline to daily points beyond 30 days. Per-class storage building cap stays 10. Contract penalties may contribute to bankruptcy.
- **Upgrades:** 12 upgrades cannot be sold. 13 managers and marketing carry upkeep, physical upgrades mostly do not. 26 managers trade for the player, opt-in and rule-based, player sets every threshold. 29 no loans (`credit_line` stays out).
- **Empire:** 6 rivals may take share, at most 1 point a day, floor at half the entry share, never enough to bankrupt on their own. 7 hostile takeovers yes, at 1.6 times valuation. 8 light-pun names (as in the catalog). 9 rivals fold and make merger offers, no cross-territory growth. 11 one global facility pool, cap 10 per type per territory entered, +20 for World. 22 acquisitions count in net worth at `ResaleRate` (0.5) times price paid. 27 economic cycles in, seasons out. Neighborhood reach = phase 0 depth × (share ÷ 40%). A won run keeps playing and its score keeps growing; the win day is shown.

## 6. Execution setup (lead agent)

- Integration branch `late/main`, worktree `../lemonade-late/main`. Track worktrees `../lemonade-late/<track>` on `late/<track>`. The lead merges tracks into `late/main`; track agents never merge into it themselves and never touch `main` or `feature/slices-9-13`.
- Postgres for store tests: container `late-pg` on port 55432, one database per track: `DATABASE_URL=postgres://late:late@localhost:55432/late_<track>?sslmode=disable`.
- Track agents do not run `docker compose` (ports clash); the lead runs the real-stack check at each wave gate.
