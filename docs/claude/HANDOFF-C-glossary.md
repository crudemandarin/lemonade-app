# Handoff C: user help glossary (and the documentation sweep)

Covers table item **15**. Build this **last**: a glossary must describe what actually shipped, so start it only after handoffs P0, A and B are merged. Check `git log` and the code to see which features exist (difficulty slider and rival are separate, not-yet-scheduled slices: include their sections only if they have shipped). Read `HANDOFF.md` and `HANDOFF-P0.md` §0 (ground rules) and §1 (commands). Stage explicit paths only.

Add this as slice 15 in `PLAN.md`.

## What to build

- **`/help` page** behind no auth requirement if the nav allows it (check `authGuard` and `app.routes.ts`; the glossary is static, so it may be public), linked from `nav-bar` with a "Help" entry. Sections with in-page navigation: Basics, Market, Facilities, Day and events, Scoring, plus Difficulty and Rival only if shipped.
- **Content in one typed constant** (`pages/help/glossary.ts`): `{id, term, section, definition, seeAlso[]}`. A spec asserts every term renders, every `seeAlso` and anchor resolves, and ids are unique. This is the guard against the glossary rotting.
- **Deep links:** small "?" links next to key labels (Bid, Ask, Upkeep, Avg cost, Net worth, Projection line in the header, Events) navigate to `/help#<id>`. Keyboard focusable, with an accessible name ("What is bid?"). Keep them unobtrusive at 390 px.
- **Search or filter box** (type to filter terms) only if it stays under about 40 lines; otherwise skip and list alphabetically within sections.
- Copy is sentence case, plain language, and short (two to four sentences per term). Explain the game, not the code.

## Terms to cover (verify each against the shipped behaviour, not this list)

- **Basics:** the recipe (1 lemon + 1 sugar + 1 ice + 1 cup makes 1 lemonade), what a day is (turn based, advances only on End day, no real-time clock), capital, net worth, how a run ends (bankrupt or give up).
- **Market:** price versus effective price (events), bid and ask and the 10% spread, why buying then selling immediately loses money, price walk and mean reversion in plain words, average purchase price and unrealized gain, bulk trade bar (All, 10, 50, 100; clamping), the daily price chart and its `$` / `% of base` toggle.
- **Facilities:** warehouse (capacity by resource) and production (cases per day), tiers and the shared level, expand versus upgrade, upkeep, selling buildings (proceeds at half of build cost, kept minimums, stock must still fit), max buildings.
- **Day and events:** end-of-day order (produce, ice melts, upkeep, advance and market tick), ice melting, the header projection ("Makes N lemonade", "M ice will melt", limiting factor), each event and what it does, why conflicting events never overlap, day report and past days.
- **Scoring:** insolvency and forced sales (sold at the bid: lemonade first, then lemon, sugar, cup), give up, score formula (net worth plus per-day weight), personal best versus global board, one row per user.
- **Only if shipped:** difficulty levels and what each changes; the rival and how the race works.

**Numbers must not contradict the game.** Prefer reading values from the game view or a small typed constants module shared with the UI (spread, resale rate, per-day score weight) instead of hard-coding them in prose. If a number has to appear in text, add a comment pointing to the `Config` field it mirrors, and note it in the README "Tuning the game" section as something to update.

## Documentation sweep (also this handoff)

Finish the docs for the whole update so the repo is coherent:
- `README.md`: features, "Tuning the game" (`ResaleRate`, `ScorePerDay`, event conflict rule), known limitations (public usernames on the board, weak auth), new API routes.
- `docs/numeric-tdd.md`: data model (new columns, `runs`, `day_reports`), API section, balance notes, the "Out (future work)" list (upgrade items and level downgrade moved there; leaderboard removed since it shipped).
- `docs/claude/SPEC.md` rules touched, `docs/claude/DECISIONS.md` (verify every deviation from the earlier handoffs is logged, numbered after entry 20), `docs/claude/PLAN.md` ticks and Future work (add: upgrade items, level downgrade, insolvency selling facilities, rival shared-market pressure).
- `docs/DEMO.md`: it has another agent's uncommitted edits. Coordinate with the user before touching; add the new features to the walkthrough, file index, and "what's next".
- `docs/claude/ai-usage-log.md`: flag "Log candidate:" moments (for example, the derived event-conflict test that covers future events, and the preview-equals-actual property test) but only append when the user confirms.

## Tests and acceptance

Frontend specs for the glossary constant integrity, page rendering, anchor scroll and focus, "?" link presence on the labeled elements, and nav entry. `npm run lint`, `npm run format:check`, `npm run build`, backend `go test ./...` unaffected. Real-stack check at 1280 and 390 px (no horizontal overflow, in-page navigation works with keyboard, deep link from a market label lands on the term). Grep the docs for stale claims (for example, "no leaderboard", "no selling facilities").

## Out of scope

New game features. If writing a definition reveals unclear or inconsistent behaviour, report it to the user instead of changing rules.
