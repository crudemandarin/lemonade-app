# Lemonade Tycoon: Demo Walkthrough

A 5-minute demo script, then the decisions and trade-offs to defend, what I'd do next, and likely questions. Live app: https://lemonade.nyko.run (local: `docker compose up -d --build`, then http://localhost:4200).

Live App: https://lemonade.nyko.run

Code Repository: https://github.com/crudemandarin/lemonade-app

**Contents**
1. [Before you start](#1-before-you-start)
2. [The 5-minute script](#2-the-5-minute-script)
3. [Decisions and trade-offs](#3-decisions-and-trade-offs)
4. [How I used AI](#4-how-i-used-ai-60-second-talking-point)
5. [What I'd do next](#5-what-id-do-next)
6. [Likely questions](#6-likely-questions)
7. [Where everything is](#7-where-everything-is)

---

## 1. Before you start

- [ ] Open the live site in one window and a code editor with the repo in another. Have the phone or a narrow window ready for the mobile view.
- [ ] Sign in with a **fresh username** (5+ characters, e.g. `demo-<date>`) so Day 1 starts clean. Have a second, older account ready to show a long history in the charts.
- [ ] Run `cd lemonade-api && go test ./...` once so the results are cached and instant.
- [ ] Have these tabs open: [numeric-tdd.md](numeric-tdd.md) (architecture diagram), [ai-usage-log.md](claude/ai-usage-log.md), [DECISIONS.md](claude/DECISIONS.md).
- [ ] Skip the Cloud Run cold start: load the site and click through once before the demo.

---

## 2. The 5-minute script

Timings are cumulative. **Say** is what to say; **Do** is what to click.

### 0:00 The pitch (30s)
**Say:** "Lemonade Tycoon is a turn-based business game. Buy lemons, sugar, ice and cups at market prices, turn them into lemonade, sell it, grow your facilities, and don't go bankrupt. Angular PWA, Go/Gin API, Postgres, deployed on Google Cloud Run. I built it in a 4-hour box, working with Claude Code in separate designer, implementer and reviewer sessions. I'll show the product first, then how it's built."
**Do:** Show the home page.

### 0:30 Sign in and install (30s)
**Do:** Sign in with the fresh username. Point at the browser's Install button.
**Say:** "Username-only login, as the brief asked. Names are case-insensitive, 3 to 40 characters, and the header identifies you on every request, so it's not secure and I say so in the README. It's an installable PWA: the app shell is cached, but gameplay needs the network because **the server is the source of truth**. I chose no offline play over stale-state bugs."

### 1:00 The market (45s)
**Do:** Go to the game page. Point at prices, arrows, sparklines and the event banner and backdrop.
**Say:** "Five resources on a seeded, mean-reverting price walk with a clamp. Every price has a bid and an ask 10% either side, so buying and selling straight back loses about 18%, which blocks same-day arbitrage. Random events like a heat wave or lemon blight multiply the *effective* price without touching the underlying walk, and they stack. The weather backdrop is CSS-only and comes from a design doc I had Claude write before any code."

### 1:45 One full day (60s)
**Do:** Buy 10 of each input. Show a rejection (buy more than capacity, or with too little cash). Click **End day**. Read the day report.
**Say:** "Warehouses hold only their own resource, so capacity limits matter. Errors come back with a typed shape and change nothing. End day is one atomic step: produce `min(rate, each input, free space)`, ice melts, upkeep is paid, the market walks, events expire and spawn. Everything in the report is computed on the server; the UI only renders it."
**Do:** Sell the lemonade.

### 2:45 Facilities (30s)
**Do:** Expand a Pantry, then hover or click Upgrade. Point at the "+$x upkeep" note.
**Say:** "Two facility types with four tiers each. Level is shared per type, count is per resource. Upgrading costs per-building cost times total buildings, so growth gets expensive, and upkeep grows with it. Expansion is a trap unless you scale production and every warehouse together."

### 3:15 Charts (20s)
**Do:** Switch to the older account and show the two charts.
**Say:** "Capital and stock are two charts on a shared time axis, not one dual axis, because dollars in the thousands and cases in the tens would mislead. Shape and fill mark buy, sell, build and upgrade, so color isn't the only signal. Claude checked the palette with a validator for light, dark and color-blind separation."

### 3:35 Losing (25s)
**Do:** Show a game-over screen (a pre-lost account is fastest).
**Say:** "Upkeep is always owed. If cash is short, the server sells stock at the bid, lemonade first. If that still can't cover it, you're bankrupt. That rule came out of a simulation, not a guess. I'll come back to it."

### 4:00 Under the hood (60s)
**Do:** Show the architecture diagram in [numeric-tdd.md](numeric-tdd.md#cloud-deployment), then `lemonade-api/internal/domain/`. Run `go test ./...`.
**Say:** "Domain rules are pure Go: no I/O, no Gin, no SQL, and no clock. The seed comes in from the API layer, so the same seed and actions give the same game. Handlers are thin, and every mutation is load, domain call, save in one transaction with `SELECT ... FOR UPDATE`. A 20-goroutine test proves no lost updates. Terraform provisions Cloud Run, Cloud SQL and Secret Manager, and a push to main deploys through keyless Workload Identity."

### 5:00 Close (say this, then stop)
**Say:** "Working core early, then depth in two places: balance by simulation, and verification against the real stack. The trade-offs and the AI process are written up in `docs/`. Happy to go deeper anywhere."

> **If you have 10 minutes:** spend the extra time on section 4 (AI process) and pick two questions from section 6. If you're running over, cut the charts and facilities beats first.

---

## 3. Decisions and trade-offs

Full reasoning for each is in [DECISIONS.md](claude/DECISIONS.md); the technical view is [numeric-tdd.md §8](numeric-tdd.md#8-key-trade-offs-and-limits).

| # | Decision | What it buys | What it costs | Source |
|---|---|---|---|---|
| 1 | **Server is the source of truth**; every mutation returns the full game view | No client-side game logic to drift; simple UI; PWA with no stale state | Every action needs the network; no offline play | [DESIGN §2](claude/DESIGN.md), [DECISIONS 3](claude/DECISIONS.md) |
| 2 | **Pure domain layer** (no I/O, injected seed) | Fast, deterministic table tests; the simulator reuses real rules | A DTO mapping step between domain and API | [numeric-tdd §2](numeric-tdd.md#2-architecture) |
| 3 | **Bid/ask spread** with whole-dollar prices | Blocks arbitrage; no cents handling | Coarse prices for cheap goods (min $1) | [DECISIONS 9](claude/DECISIONS.md) |
| 4 | **Shared level per facility type**, count per resource | Simple upgrade rules and UI | No mixed tiers (a "Barn just for lemons") | [DECISIONS 7](claude/DECISIONS.md) |
| 5 | **Tuned balance by simulation**; upkeep is always owed (forced stock sales, then bankruptcy) | Real stakes; no "zombie" players at $0 holding one case; guard-rail tests keep the feel | Harsher than the first spec; the player can't choose what's sold | [DECISIONS 16-18](claude/DECISIONS.md), [numeric-tdd §9](numeric-tdd.md#9-balance-and-tuning) |
| 6 | **Header-only auth** (username), looked up on every request | Matches the brief; trivial to demo | Anyone can play as anyone; a session token would be the first change | [DECISIONS 19](claude/DECISIONS.md) |
| 7 | **JSONB for game internals**, one game row per user | No join tables; atomic saves | No SQL analytics over game state | [numeric-tdd §4](numeric-tdd.md#4-data-model) |
| 8 | **Frontend built ahead of the backend**; `api.models.ts` is the contract | Parallel work; UI verified against fixtures | Contract shifted once (upgrade costs, `/api` proxy), caught by a real-stack run | [DECISIONS 3, 11, 12](claude/DECISIONS.md) |
| 9 | **PWA caches the shell only** | No stale game data ever served | Install works; offline play doesn't | [DECISIONS 6, 14](claude/DECISIONS.md) |
| 10 | **Two charts, not a dual axis**; older days keep milestones only | Honest scales; bounded payload (400-point cap) | Old individual trades disappear from the chart | [DECISIONS 20](claude/DECISIONS.md) |

### The three to lead with if time is short
1. **Server-authoritative, pure domain.** It's why testing was cheap and why the balance work was possible.
2. **Balance by measurement.** Simulated players showed 54-69% of careless games ended as "zombies" who could never lose. Raising upkeep alone changed nothing for careful players. The fix was a rule change (upkeep is always owed) plus numbers. The `TestBalance*` tests now hold the difficulty in a known band.
3. **Scope discipline.** [PLAN.md](claude/PLAN.md) lists what I cut on purpose: mixed tiers, demand simulation, loans, real auth, e2e tests.

### Known limits (say them before they're asked)
- The header is the only credential.
- The 400-point timeline cap drops the oldest milestones in very long games (about day 350+).
- The brief mentioned three facility types but named two. I modeled two; a third is another type with its own level and tiers.
- No end-to-end browser tests; unit, component and API tests only.

---

## 4. How I used AI (60-second talking point)

> "I treated Claude Code as a small team with separate roles and file-based handoffs. A designer session produced the [SPEC](claude/SPEC.md), [DESIGN](claude/DESIGN.md) and [PLAN](claude/PLAN.md). Implementer sessions built one slice at a time, test first, following [CLAUDE.md](claude/CLAUDE.md), and logged every deviation in [DECISIONS.md](claude/DECISIONS.md). A separate reviewer session audited the code against the spec and gave me a numbered list I could triage in one line per item. The moments I'd point to are where the AI was wrong and the process caught it: a float rounding bug found by a test taken from the spec, a balance tool whose scary numbers turned out to be a bug in the simulated player, and a contract mismatch that only a real-stack run exposed. Every change had to come with evidence: a test, a screenshot, or a real run."

Five entries worth opening if asked, all in the [AI usage log](claude/ai-usage-log.md):

| Entry | Shows |
|---|---|
| [#9 Tuning balance by simulation](claude/ai-usage-log.md#9-tuning-game-balance-by-simulation-and-refusing-to-trust-the-first-scary-number) | Measuring instead of guessing; debugging the instrument before the product |
| [#10 Compiling real responses against the contract](claude/ai-usage-log.md#10-verifying-the-backend-matches-the-contract-by-compiling-real-responses-against-it) | Using the type checker as a contract test |
| [#12 Reviewer role](claude/ai-usage-log.md#12-reviewer-role-a-numbered-audit-i-could-triage-in-one-line-per-item) | Role separation; numbered findings; a stale-report failure I corrected |
| [#7 Test-first caught a float bug](claude/ai-usage-log.md#7-test-first-caught-an-ai-written-float-bug-the-ais-own-tests-were-wrong-too) | Spec-derived tests; deciding which side is wrong |
| [#2 Design first, then a gate](claude/ai-usage-log.md#2-design-pass-before-code-then-a-hard-gate-before-integration) | The model jumped ahead twice; how I constrained it |

The reusable prompts for each role are in [kickoff-prompts.md](claude/kickoff-prompts.md). The session's design conversation is in [design-session-notes.md](claude/design-session-notes.md).

---

## 5. What I'd do next

Ordered by value for effort. The complete cut list is under "Future work" in [PLAN.md](claude/PLAN.md).

**With another day**
1. **End-to-end tests** (Playwright): the sign-in, buy, end day, sell, bankrupt flow against a real Postgres. Today unit and API tests cover each layer, and the two integration bugs I hit came from the seams between them.
2. **Real authentication.** A session token or OAuth in place of the header, plus a leaderboard on `capital` and `day`.
3. **A CI pipeline.** Run `go test`, `ng test` and lint on every PR before the deploy job. Today the deploy workflow ships on push to main.
4. **Player-chosen liquidation.** Let the player pick what to sell when upkeep is short, instead of the fixed order, and add a one-day grace period.
5. **Offline-tolerant PWA.** An update-available prompt and a read-only cached game view.

**With a week**
6. **Demand simulation.** Player-set lemonade price, weather-driven customers, and price impact from your own trades, so the market reacts to you.
7. **Richer facilities.** Mixed tiers, selling or downgrading, specialty warehouses (a freezer for ice), a third facility type.
8. **Event forecasting** ("heat wave expected tomorrow"), bulk contracts, and loans with interest.
9. **Observability.** Structured logs and a readiness check that pings the database (today `/api/health` is liveness only; see [DECISIONS 2](claude/DECISIONS.md)).
10. **Balance for fun, not just solvency.** Playtest with real people; the simulation only tells me the game is winnable and losable.

---

## 6. Likely questions

**Why Go for the rules and not share logic with the frontend?**
One implementation means the client can't disagree with the server. Sharing would need a second runtime or duplicated code, and the client would still have to be trusted.

**How do you know the balance is good?**
I only know it's *bounded*: careful players go bankrupt about 1 in 8 within 90 days, sloppy about 2 in 3 within 45, idle around day 34. Whether it's *fun* needs real players. That's the last item under "next".

**What happens with two tabs or double clicks?**
Each mutation locks the game row with `SELECT ... FOR UPDATE`, so requests serialize. A test fires 20 concurrent mutations and checks none are lost. On the client, buttons are disabled while a request is busy.

**Why JSONB and not normalized tables?**
The game is always read and written whole, by one user. JSONB gives atomic saves and no join tables. The cost is no SQL analytics, which I don't need yet.

**Why is bankruptcy checked only at end of day?**
Spending to $0 mid-day is a legitimate move. The check happens where the money is owed: at upkeep.

**What would break first at scale?**
The per-request user lookup and the whole-game JSON payload (about 13 KB typical). A session token, a cache, and trimming the timeline payload would come first. Cloud SQL is `db-f1-micro`, so that would go up too.

**What did the AI get wrong?**
Rounding (`100 × 1.1 = 110.00000000000001`, which rounded the ask to 111), a balance simulator that sank its own player, an integration test that could have wiped a shared database, and docs that disagreed with the client on warehouse levels. Each is written up in [ai-usage-log.md](claude/ai-usage-log.md) and [DECISIONS.md](claude/DECISIONS.md).

**How do I run it?**
`cp .env.example .env && docker compose up -d --build`, then http://localhost:4200. Tests: `go test ./...` in `lemonade-api/`, and `npx ng test --watch=false --browsers=ChromeHeadless` in `lemonade-web/`. See the [README](../README.md).

---

## 7. Where everything is

| I want to show... | Open |
|---|---|
| The rules, numbered | [SPEC.md](claude/SPEC.md) |
| Architecture, API, data model, balance | [numeric-tdd.md](numeric-tdd.md) |
| The original design and slice plan | [DESIGN.md](claude/DESIGN.md), [PLAN.md](claude/PLAN.md) |
| Every deviation and trade-off (20 entries) | [DECISIONS.md](claude/DECISIONS.md) |
| The instructions Claude worked under | [CLAUDE.md](claude/CLAUDE.md) |
| **My prompts by role** | [kickoff-prompts.md](claude/kickoff-prompts.md) |
| **Curated prompts, results and lessons (13 entries)** | [ai-usage-log.md](claude/ai-usage-log.md) |
| Design session and UX mocks | [design-session-notes.md](claude/design-session-notes.md), [UX-MOCKS-AND-CHANGES.md](claude/UX-MOCKS-AND-CHANGES.md) |
| Event backdrop design | [EVENT-BACKDROPS.md](claude/EVENT-BACKDROPS.md) |
| Balance guard-rail tests | [balance_test.go](../lemonade-api/internal/domain/balance_test.go) |
| Game config (every knob) | [config.go](../lemonade-api/internal/domain/config.go) |
| Cloud deployment | [deploy/README.md](../deploy/README.md) |
