package content

// Upgrade categories, in the order the Upgrades page lists them.
const (
	UpFreshness  = "freshness"
	UpProduction = "production"
	UpBrand      = "brand"
	UpIntel      = "intelligence"
	UpResilience = "resilience"
	UpSupply     = "supply"
	UpFinance    = "finance"
	UpQoL        = "convenience"
)

// UpgradeCategories is the display order of the categories.
var UpgradeCategories = []string{UpFreshness, UpProduction, UpBrand, UpIntel, UpResilience, UpSupply, UpFinance, UpQoL}

// Effect kinds: the closed set of typed modifiers an upgrade can carry. Each kind
// has one hook in the domain (effects.go) and one test; a new upgrade is one row.
// Percentages are whole percent points (15 means 15%).
const (
	// EffIceKeep: keep up to Value cases of ice one extra night (the largest owned counts).
	EffIceKeep = "ice_keep_cases"
	// EffEventDamp: scale the strength of event Target for the player. Factor Value
	// multiplies the event's distance from 1: 0.5 halves it, 1.5 makes it half as strong again, 0 ignores it.
	EffEventDamp = "event_damp"
	// EffEventFloor: a negative price event on a product is never worse than Value (a multiplier).
	EffEventFloor = "event_floor"
	// EffForecast: see events Value days ahead. Target is "weather" or "all".
	EffForecast = "forecast_days"
	// EffDepthBonus: Value extra cases of free depth for commodity Target.
	EffDepthBonus = "depth_bonus"
	// EffDepthBonusPct: Value percent more free depth for commodity Target.
	EffDepthBonusPct = "depth_bonus_pct"
	// EffInputDepthPct: Value percent more free depth for every input commodity.
	EffInputDepthPct = "input_depth_pct"
	// EffInputDiscountPct: input asks are Value percent lower.
	EffInputDiscountPct = "input_discount_pct"
	// EffUpkeepDiscountPct: facility upkeep is Value percent lower. Target is
	// "all", "production" or "warehouse".
	EffUpkeepDiscountPct = "upkeep_discount_pct"
	// EffYield: recipe Target makes Value percent more output (fractions carry over).
	EffYield = "yield_bonus"
	// EffUseDiscount: recipes use Value percent less of commodity Target.
	EffUseDiscount = "use_discount"
	// EffStoragePct: Value percent more capacity for storage class Target.
	EffStoragePct = "storage_bonus_pct"
	// EffShelfLife: perishables of storage class Target keep Value extra days.
	EffShelfLife = "shelf_life_days"
	// EffMake: each night, top up commodity Target by up to Value cases, at Aux dollars a case.
	EffMake = "make"
	// EffUnlock: unlocks feature or recipe Target for other systems to check.
	EffUnlock = "unlock"
	// EffQoL: turns on a convenience feature (Target) in the app or the day report.
	EffQoL = "qol"
)

// EffectDef is one typed modifier.
type EffectDef struct {
	Kind   string
	Target string
	Value  float64
	Aux    float64
}

// Requires gates an upgrade. Zero means no requirement.
type Requires struct {
	// Era is the era the player must have reached (see domain.Era).
	Era             int
	WarehouseLevel  int
	ProductionLevel int
	// Upgrades are keys that must already be owned.
	Upgrades []string
}

// UpgradeDef is one row of the upgrade table. Upgrades are bought once and cannot be sold.
type UpgradeDef struct {
	Key      string
	Name     string
	Category string
	Cost     int
	// Upkeep is dollars a day, paid with the buildings' upkeep.
	Upkeep   int
	Requires Requires
	Effects  []EffectDef
	// Text is what the upgrade does, in plain words, for the Upgrades page.
	Text string
}

// Upgrades is the table (late game content catalog, section E). Territory upgrades
// (signs, billboards, delivery fleet, rival intel) are added by the Empire track,
// managers by Upgrades stage B.
var Upgrades = []UpgradeDef{
	// Freshness and storage
	{Key: "freezer_1", Name: "Chest freezer", Category: UpFreshness, Cost: 1200, Upkeep: 5, Requires: Requires{Era: 1},
		Effects: []EffectDef{{Kind: EffIceKeep, Value: 20}}, Text: "Keep up to 20 cases of ice one extra night."},
	{Key: "freezer_2", Name: "Walk-in freezer", Category: UpFreshness, Cost: 8000, Upkeep: 20, Requires: Requires{Era: 2, Upgrades: []string{"freezer_1"}},
		Effects: []EffectDef{{Kind: EffIceKeep, Value: 150}}, Text: "Keep up to 150 cases of ice one extra night."},
	{Key: "freezer_3", Name: "Cold storage wing", Category: UpFreshness, Cost: 60000, Upkeep: 120, Requires: Requires{Era: 3, Upgrades: []string{"freezer_2"}},
		Effects: []EffectDef{{Kind: EffIceKeep, Value: 1200}}, Text: "Keep up to 1,200 cases of ice one extra night."},
	{Key: "cold_room", Name: "Cold room", Category: UpFreshness, Cost: 6000, Upkeep: 15, Requires: Requires{Era: 2},
		Effects: []EffectDef{{Kind: EffShelfLife, Target: StorageCold, Value: 2}}, Text: "Fresh fruit and herbs keep 2 extra days."},
	{Key: "bulk_racking", Name: "Bulk racking", Category: UpFreshness, Cost: 2000, Requires: Requires{Era: 1},
		Effects: []EffectDef{{Kind: EffStoragePct, Target: StorageDry, Value: 15}}, Text: "15% more room for dry goods (sugar and cups)."},

	// Production
	{Key: "citrus_press", Name: "Citrus press", Category: UpProduction, Cost: 6000, Requires: Requires{Era: 1},
		Effects: []EffectDef{{Kind: EffYield, Target: "lemonade", Value: 10}}, Text: "Every 10 batches of lemonade make 1 extra case."},
	{Key: "sugar_dissolver", Name: "Syrup station", Category: UpProduction, Cost: 10000, Requires: Requires{Era: 2},
		Effects: []EffectDef{{Kind: EffUseDiscount, Target: "sugar", Value: 10}}, Text: "Recipes use 10% less sugar."},
	{Key: "automation_line", Name: "Automation line", Category: UpProduction, Cost: 120000, Requires: Requires{Era: 3},
		Effects: []EffectDef{{Kind: EffUpkeepDiscountPct, Target: "production", Value: 15}}, Text: "Production upkeep 15% lower."},
	{Key: "oven", Name: "Commercial oven", Category: UpProduction, Cost: 12000, Upkeep: 15, Requires: Requires{Era: 2},
		Effects: []EffectDef{{Kind: EffUnlock, Target: "bakery"}}, Text: "Unlocks the bakery recipes."},
	{Key: "carbonator", Name: "Carbonation rig", Category: UpProduction, Cost: 50000, Upkeep: 30, Requires: Requires{Era: 3, ProductionLevel: 3},
		Effects: []EffectDef{{Kind: EffUnlock, Target: "sparkling"}}, Text: "Unlocks the sparkling recipes (needs a Bottling Plant or better)."},
	{Key: "zester", Name: "Zester", Category: UpProduction, Cost: 9000, Requires: Requires{Era: 2},
		Effects: []EffectDef{{Kind: EffUnlock, Target: "byproduct_lemon_peel"}}, Text: "Every lemon used also yields a lemon peel."},

	// Brand
	{Key: "painted_stand", Name: "Painted stand", Category: UpBrand, Cost: 1500, Requires: Requires{Era: 1},
		Effects: []EffectDef{{Kind: EffDepthBonus, Target: "lemonade", Value: 10}}, Text: "10 more cases of lemonade sell at the plain price."},
	{Key: "mascot", Name: "Mascot: Lemmy the Lemon", Category: UpBrand, Cost: 20000, Requires: Requires{Era: 2},
		Effects: []EffectDef{{Kind: EffEventDamp, Target: "holiday", Value: 1.5}}, Text: "Holidays lift lemonade prices half as much again."},

	// Intelligence
	{Key: "weather_radio", Name: "Weather radio", Category: UpIntel, Cost: 1000, Requires: Requires{Era: 1},
		Effects: []EffectDef{{Kind: EffForecast, Target: "weather", Value: 1}}, Text: "See tomorrow's weather event a day ahead."},
	{Key: "farmers_almanac", Name: "Farmer's almanac", Category: UpIntel, Cost: 8000, Requires: Requires{Era: 2},
		Effects: []EffectDef{{Kind: EffForecast, Target: "all", Value: 3}}, Text: "See every market event 3 days ahead."},
	{Key: "market_analyst", Name: "Market analyst", Category: UpIntel, Cost: 15000, Upkeep: 25, Requires: Requires{Era: 2},
		Effects: []EffectDef{{Kind: EffQoL, Target: "moving_average"}}, Text: "A 7-day average on every price row."},

	// Event resilience
	{Key: "awnings", Name: "Awnings and umbrellas", Category: UpResilience, Cost: 1500, Requires: Requires{Era: 1},
		Effects: []EffectDef{{Kind: EffEventDamp, Target: "rainy_week", Value: 0.5}}, Text: "Rainy Week hurts lemonade half as much."},
	{Key: "ice_machine", Name: "Ice machine", Category: UpResilience, Cost: 10000, Upkeep: 15, Requires: Requires{Era: 2},
		Effects: []EffectDef{{Kind: EffMake, Target: "ice", Value: 50, Aux: 4}}, Text: "Makes up to 50 cases of ice a night at $4 each, whatever the market says."},
	{Key: "generator", Name: "Backup generator", Category: UpResilience, Cost: 40000, Upkeep: 20, Requires: Requires{Era: 3},
		// Power outage is a market event added by a later track; the damp is inert until then.
		Effects: []EffectDef{{Kind: EffEventDamp, Target: "power_outage", Value: 0}}, Text: "Power outages do not affect you."},
	{Key: "orchard_lease", Name: "Orchard lease", Category: UpResilience, Cost: 100000, Upkeep: 100, Requires: Requires{Era: 3},
		Effects: []EffectDef{{Kind: EffEventDamp, Target: "lemon_blight", Value: 0.5}, {Kind: EffDepthBonusPct, Target: "lemon", Value: 20}},
		Text:    "Lemon Blight hurts half as much, and lemons trade 20% deeper."},
	{Key: "insurance", Name: "Business insurance", Category: UpResilience, Cost: 80000, Upkeep: 150, Requires: Requires{Era: 3},
		Effects: []EffectDef{{Kind: EffEventFloor, Value: 0.9}}, Text: "A bad event never cuts a product price below 90%."},

	// Supply
	{Key: "supplier_contract_1", Name: "Supplier contract", Category: UpSupply, Cost: 10000, Requires: Requires{Era: 2},
		Effects: []EffectDef{{Kind: EffInputDiscountPct, Value: 3}}, Text: "Input prices 3% lower."},
	{Key: "supplier_contract_2", Name: "Bulk supplier network", Category: UpSupply, Cost: 100000, Requires: Requires{Era: 3, Upgrades: []string{"supplier_contract_1"}},
		Effects: []EffectDef{{Kind: EffInputDepthPct, Value: 30}, {Kind: EffInputDiscountPct, Value: 3}}, Text: "Inputs trade 30% deeper and cost 3% less again."},

	// Finance
	{Key: "bookkeeper", Name: "Bookkeeper", Category: UpFinance, Cost: 2000, Upkeep: 5, Requires: Requires{Era: 1},
		Effects: []EffectDef{{Kind: EffQoL, Target: "pnl"}}, Text: "A profit and loss breakdown in the day report."},
	{Key: "accountant", Name: "Accountant", Category: UpFinance, Cost: 20000, Upkeep: 30, Requires: Requires{Era: 2},
		Effects: []EffectDef{{Kind: EffUpkeepDiscountPct, Target: "all", Value: 5}}, Text: "Building upkeep 5% lower."},

	// Convenience
	{Key: "order_book", Name: "Order book", Category: UpQoL, Cost: 500, Requires: Requires{Era: 1},
		Effects: []EffectDef{{Kind: EffQoL, Target: "repeat_trades"}}, Text: "A button to repeat yesterday's trades."},
	{Key: "price_alerts", Name: "Price alerts", Category: UpQoL, Cost: 800, Requires: Requires{Era: 1},
		Effects: []EffectDef{{Kind: EffQoL, Target: "price_alerts"}}, Text: "Mark a price row when it crosses a level you set."},
}
