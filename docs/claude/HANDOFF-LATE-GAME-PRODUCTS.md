# Handoff: late game Products track (data-driven commodities, recipes, contracts)

Read `HANDOFF-LATE-GAME-ROADMAP.md` first (waves, shared contracts, branch `late/products`). Then read `HANDOFF.md`, `HANDOFF-P0.md` sections 0 and 1, `LATE-GAME-DESIGN.md` sections 0, 6.5, 8 and 9, and `LATE-GAME-CONTENT.md` sections G, H, I and J. Check `git log` and `git status`. Stage explicit paths only. Test first.

**User decisions already made:** fresh fruit and herbs are perishable, but lemons keep. Storage is pooled by class. Prices are global, with one price per commodity (design section 6.2).

**Ask the user first:** open questions 16 (launch set), 17 (unlock method), 18 (byproducts), 30 (compact money), 31 (timeline downsampling).

**Concurrency:**
- **Stage A:** wave 1, and it may only run alongside the Goals track. Every other track waits for it to merge.
- **Stage B:** wave 3, alongside Empire B. It needs Stage A and Upgrades A.
- **Stage C:** wave 4, alongside Upgrades B.

## Stage A: data-driven commodities, with no new content (design phase 4)

**Goal:** the five resources become catalog data, and the game plays **identically**.

- **Catalog.** `internal/domain/content/commodities.go`: `CommodityDef{Key, Name, Category, StorageClass, BasePrice, ShelfLifeDays (0 means keeps, -1 means melts nightly), Input bool, Product bool, Order int}`. For now it lists exactly lemon, sugar, ice, cup and lemonade.
- **Replace the fixed list.**
  - Replace `Resources`, `Inputs` and `Resource.Valid()` in `resource.go` with lookups into the catalog held on `Config` (`cfg.Commodities`, `cfg.CommodityByKey`). Keep the `Resource` string type as the key.
  - **The recipe becomes data too:** `Recipe{Key, Output, OutputQty, Inputs map[Resource]int}` with one row (lemonade). `produce()`, `PreviewEndDay`, cost basis and the bots read it, not the `Inputs` slice.
- **Fixed arrays become maps.** `TimelinePoint.Stock [5]int` (`timeline.go`) and `PricePoint.Prices [5]int` (`pricelog.go`) become `map[Resource]int`. JSON back-compat: a custom `UnmarshalJSON` reads the old 5-element arrays in catalog order (lemon, sugar, ice, cup, lemonade), and a test loads a stored old-shape timeline and price log.
- **Frontend.**
  - `Resource` in `core/api.models.ts` becomes `string`.
  - The view carries the catalog (`commodities: [{key, name, category, storageClass, isProduct, order}]`), or a static `GET /api/catalog` cached by the store. `core/resources.ts` (names, colours, icons) becomes a lookup with a fallback for unknown keys (a neutral colour, a generic icon).
  - Charts and the market panel iterate the catalog, not a hard-coded list.
  - Timeline colours for new commodities: extend the palette with the dataviz validator (as for the existing five) when stage B adds them.
- **The proof.** Add a test that plays the existing bots with fixed seeds and compares their full outcome (day-by-day capital and stock) to a golden file recorded **before** the refactor. It must be byte-identical. Also re-run the `tsc --strict` contract check against a real response.
- **Out of scope for A:** any new commodity, storage class or perishable rule. If you are tempted, stop.

## Stage B: recipes, new commodities, storage classes, perishables (design phase 5)

- **Commodities and recipes.** Add the launch set that the user picks (default: lime, mint, honey, black tea, strawberry, and flour, butter and eggs for bakery). Add the matching recipes from content section H: limeade, mint lemonade, honey lemonade, Arnold Palmer, strawberry lemonade, and lemon bars. Each needs `Requires` (era or tier, and upgrade keys such as `oven`), which reads the owned upgrades from Upgrades A.
- **Production plan.** `Game.ProductionPlan []PlanRow{Recipe, Target (0 means as much as possible)}`. `produce()` fills it in order, limited by inputs, space and capacity. The default plan for old saves is `[{lemonade, 0}]`, which is today's behavior. `PreviewEndDay` returns a projection per recipe and its limiting factor. The existing preview-equals-actual property test is extended to multiple recipes. Add `POST /api/game/production-plan`.
- **Per-product markets.** Each product gets its own price walk, depth and events, all using the existing code. Product depth is `freeDepth(g, cfg, product)`, multiplied by the product's "depth vs lemonade" ratio. The Empire track makes this per territory later through the same helper.
- **Storage classes (user decision).**
  - Classes are dry, cold, frozen and finished.
  - Warehouses become one building count **per class** at the shared level. Existing per-resource warehouses map by the class of each resource (lemon to cold; sugar and cup to dry; ice to frozen; lemonade to finished). Capacity per class is the count times the tier size.
  - **Migration:** sum each old game's per-resource building counts into its class. Capacity must never drop below stored stock, and a test covers that.
  - The facilities panel groups by class.
  - **Check with the user** whether the per-class building cap stays 10 or grows (four classes instead of five resources changes total capacity). Re-run `TestEachLevelPaysMoreThanTheLast` and the balance report.
- **Perishables (user decision).**
  - Fresh inputs have `ShelfLifeDays`. Stock is tracked in age buckets for those commodities only, and production uses the oldest first. At end of day, stock older than its shelf life spoils in the new spoilage step (roadmap section 3, step 4). Cost basis falls with it.
  - `DayReport.Spoiled` gets a field per commodity, shown in the day report. The header projection shows "N will spoil".
  - The `cold_room` upgrade adds 2 days, read from Upgrades A's effect hooks.
  - **Lemons keep** (they are grandfathered).
  - This is not an event and not random, so it does not conflict with the "no inventory-destroying random events" rule. Say so in DECISIONS.
- **Events.** Add the content section I rows for the launched commodities (strawberry season, mint frost, bee decline, bumper crop, food blog, influencer). Conflicts are derived automatically (the P0 rule).
- **UI.**
  - The market panel gets category tabs (Ingredients, Products, All) once there are more than 8 rows. Locked commodities are hidden, and constant row height is kept.
  - A new **Production** page (from the nav menu or a card link on the game page) shows the plan rows, reorder buttons (accessible up and down buttons, not drag only), and the projection per recipe.
  - Compact money (`$1.2M`) comes in if question 30 is yes.
- **Achievements.** Add the recipe rows in content section K (first recipe, five products, all recipes, strawberry 1k, hedged, byproduct if 18 is yes) to the Goals table.
- **Balance.** Add a **diversified bot** (adds recipes when capacity is idle or a product's margin beats lemonade). Target: it beats the single-product careful grower by a modest margin, 10 to 40% at day 90. Perishables must not raise careful-grower bankruptcy above 9%.

## Stage C: contracts (design phase 7, part)

- **Content.** Content section J: `ContractDef{Key, Client, Product, Qty, DeadlineDays, PricePerCase or PremiumPct, Era}`. Offers come from a seeded roll (with its own salt), and at most one is open at a time. An offer lasts 3 days.
- **Actions.** `POST /api/game/contracts/:key/accept`. Deliveries come from stock through `POST /api/game/contracts/:key/deliver {qty}`, or automatically at end of day from finished goods, if the user prefers. Delivery is paid at the contract price, and does not use market depth.
- **Penalty.** A failed contract costs 20% of its value, capped at 5% of net worth. The penalty is paid like upkeep. It can trigger forced sales, but **check with the user** that a contract penalty may contribute to bankruptcy. The default is that it can, since it is a choice the player took.
- **UI.** A contracts card on the game page shows the progress bar and days left.
- **Achievements.** Add contract rows if the user wants them.
- **Tests.**
  - The offer roll is deterministic and does not change market prices for a seed.
  - Delivery accounting is correct, and so is the penalty cap.
  - Old saves have no contract.

## Docs (every stage)
- `DECISIONS.md`, numbered at merge.
- `numeric-tdd.md`: data model, API and section 9.
- README "Tuning the game": commodities, recipes, shelf lives, storage classes and contracts.
- The glossary: recipe, production plan, storage class, spoilage, contract.
- `PLAN.md`.
