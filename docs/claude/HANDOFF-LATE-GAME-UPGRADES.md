# Handoff: late game Upgrades track (upgrades, then managers and fast-forward)

Read `HANDOFF-LATE-GAME-ROADMAP.md` first (waves, shared contracts, branch `late/upgrades`). Then read `HANDOFF.md`, `HANDOFF-P0.md` sections 0 and 1, `LATE-GAME-DESIGN.md` sections 0, 6.4 and 6.6, and `LATE-GAME-CONTENT.md` sections E and F. Check `git log` and `git status`. Stage explicit paths only. Test first.

**Ask the user first:** open questions 12 (upgrades cannot be sold; the default is yes, per the earlier "only building quantity can be sold"), 13 (which upgrades carry upkeep), 26 (managers trade for the player, opt-in and rule-based), and 29 (loans: leave `credit_line` out unless approved).

**Concurrency:**
- **Stage A:** wave 2, alongside Empire A. It needs Products A merged. **Merge the framework commit first**, so Empire A can add its territory upgrades on top.
- **Stage B:** wave 4, alongside Products C. It needs Products B.

## Stage A: the upgrade framework and the first upgrades (design phase 2)

### Framework (pure domain)
- **Content.** `internal/domain/content/upgrades.go`: `UpgradeDef{Key, Name, Category, Cost, Upkeep, Requires{Era or WarehouseLevel/ProductionLevel, Upgrades []string}, Effects []Effect}`.
  - Until Empire A lands, "era" is not defined. Gate on facility level, or treat everything as era 1, and leave the era gate as a field Empire A will fill.
- **State.** `Game.Upgrades map[string]int`, which holds the key and a level (1 for single-level upgrades). Old saves get an empty map.
- **Effects** are a closed set of typed modifiers, and each type has **one hook** and one test. The domain reads them through small query helpers (for example `effects.IceKeep(g)` or `effects.InputDiscount(g, r)`), never by looking up upgrade keys.
  - The initial types are `IceKeepCases`, `EventDamp{event, factor}`, `ForecastDays`, `DepthBonus{commodity, cases or pct}`, `InputDepthBonusPct`, `InputDiscountPct`, `UpkeepDiscountPct{facility}`, `YieldBonus{recipe or commodity}`, `UseDiscount{commodity, pct}`, `StorageBonusPct{class}`, `Unlock{recipe or feature}`, `QoL{feature}`.
  - Empire A adds `PresenceBonus` and `HubUpkeepDiscount`. Stage B adds `Manager{kind}`.
- **Actions.** `POST /api/game/upgrades/:key/buy` checks the requirements, the cash and the next level. Errors are 409 `upgrade_locked` (with the reason), `upgrade_owned` and `insufficient_funds`. There is no sell endpoint if question 12 is yes. Stats count upgrades bought and spend.
- **Upkeep.** Upgrade upkeep is added to `TotalUpkeep` in step 5 of the end-of-day order (roadmap section 3), and it appears in the upkeep line of the report.
- **Net worth.** Upgrades are **not** counted in net worth (they cannot be sold). Record this in DECISIONS, because it makes buying one lower your score on the day you buy it.

### First upgrades (from content section E, minus the territory ones Empire A adds)
- **Freshness:** `freezer_1..3`, `cold_room` (its hook is used by Products B), `bulk_racking`.
- **Production:** `citrus_press`, `sugar_dissolver`, `automation_line`, `oven`, `carbonator`, `zester` (the unlock types are consumed by Products B).
- **Brand, no territory needed:** `painted_stand`, `mascot`.
- **Intelligence:** `weather_radio`, `farmers_almanac`, `market_analyst`.
- **Event resilience:** `awnings`, `ice_machine`, `generator`, `orchard_lease`, `insurance`.
- **Supply:** `supplier_contract_1..2`.
- **Finance:** `bookkeeper`, `accountant`.
- **QoL:** `order_book`, `price_alerts`.

### Worked mechanics that need care
- **Freezer.** Track ice in two age buckets (fresh, and one night old). Production uses the oldest first. At the freezer-rotation step, up to `IceKeepCases` of fresh ice move to "old", and the rest melts as today. Old ice that is not used by the next production melts. Cost basis moves with it. Add "Ice kept in freezer" to the day report and "N ice kept, M will melt" to the header projection. The preview-equals-actual test covers the freezer.
- **Forecast.** Pre-roll tomorrow's event with the same seeded draw `EndDay` will use (`seed ^ (day+1)` with the events salt). A test must prove the forecast always equals what then happens. Show it in the events banner ("Tomorrow: Heat Wave likely"). Events are then certain, so say "Tomorrow: Heat Wave".
- **Insurance and damping.** These change the effective multiplier the player sees and trades at, not the walk. A test checks that the event history is unchanged.
- **Yield bonus.** 1 lemon becomes 1.1 cases. Fractions carry over as a per-recipe float remainder on `Game`, so output stays whole cases and deterministic. There is a test for the remainder carry-over.
- **Event damping** of Rainy Week (awnings) interacts with derived event conflicts. Damping does not change which events can overlap.

### UI
- An **Upgrades** page in the nav menu shows each category as a card list with the cost, upkeep, effect in plain words, and the state (owned, available, or locked with its reason).
- Buy buttons are secondary (the page has no primary button), and a confirmation shows the upkeep.
- The game page shows small "owned" indicators where effects appear: the freezer line in the header, the forecast in the events banner, and the moving average on price rows.

### Balance
- Add an "upgrader" rule to the diligent bot: buy an upgrade when its payback at base prices is 20 days or less, and the cash cushion allows it.
- Targets:
  - The careful grower (with no upgrades) keeps phase 0 numbers within noise.
  - The upgrader beats it by 10 to 30% at day 90.
  - No single upgrade shortens "everything maxed" by more than 10 days (a sweep test knocks out one upgrade at a time).

### Achievements
Add the rows: `first_upgrade_item`, `freshness_set` and `upgrades_all`. `upgrades_all` excludes territory upgrades until Empire lands.

## Stage B: managers and fast-forward (design phase 6)

- **Managers are upgrades.** They have an `Effect{Manager{kind}}` and upkeep, and their settings live in `Game.ManagerSettings` (per kind: thresholds from content section F). `POST /api/game/managers/:kind/settings`. The CFO blocks manager spending below the reserve, but never the player's own actions.
- **Execution.** Managers run in step 1 of the end-of-day order, in this fixed order: CFO check, sales, purchasing, plant, logistics.
  - They **reuse the bots' decision helpers** (`profitableBatch`, `capacityBinds`, moved from test files into a non-test package file, such as `internal/domain/advisor.go`). The bots then call the same code.
  - Every manager action is a normal `Buy`, `Sell` or plan change, so impact, cost basis, the timeline and stats all apply. Record each action in the day report under "Managers did".
- **Fast-forward.** `POST /api/game/fast-forward {days (1 to 7), stopOn: [...]}` needs the `fast_forward` upgrade and the purchasing, sales and plant managers.
  - It runs up to N end-of-days in one transaction. It **stops early** on a new event, a rival telegraph (once Empire exists), a contract deadline, cash under the CFO reserve, or bankruptcy risk (projected upkeep above cash plus sellable stock at the bid).
  - It returns every day report and the final view, and the stored day reports get one row per day.
  - **Cap the request duration.** Seven days of `EndDay` is cheap, but check the real stack.
- **UI.**
  - Manager settings live inline on the Upgrades page, as one rule sentence per manager with number inputs.
  - A secondary "Fast-forward" button sits next to End day (End day stays the primary). Its dialog has the day count and stop conditions.
  - The result opens the past-days drawer at the first skipped day.
- **Tests.**
  - Each manager's rule respects its thresholds.
  - Managers never exceed the CFO reserve.
  - Fast-forward equals N single end-days with managers (a property test), and it stops correctly on each condition.
  - Determinism holds.
  - The "diligent bot = a player with every manager" check: the bot's policy and the managers produce the same actions on a sample of states.
- **Achievements.** Add rows if the user wants them (for example "Autopilot: fast-forward 7 days without stopping").

## Docs (both stages)
- `DECISIONS.md`: the effect set, freezer buckets, forecast pre-roll, upgrades not counted in net worth, managers reusing bot helpers, fast-forward stop rules.
- `numeric-tdd.md`.
- README "Tuning the game": upgrade table and manager rules.
- The glossary: upgrade, freezer, forecast, manager, fast-forward.
- `PLAN.md`.
