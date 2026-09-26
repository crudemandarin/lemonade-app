package content

// Achievements are cosmetic goals (late game Goals A). Each row is plain data: its
// condition is a Predicate, one of a small closed set of typed checks that the domain
// evaluates (domain/achievements.go). Later tracks add rows for their systems here.

// Achievement tiers.
const (
	TierBronze = "bronze"
	TierSilver = "silver"
	TierGold   = "gold"
)

// AchievementCategory is a group on the achievements page.
type AchievementCategory struct {
	Key  string
	Name string
}

// AchievementCategories are the groups in display order.
var AchievementCategories = []AchievementCategory{
	{Key: "wealth", Name: "Wealth"},
	{Key: "survival", Name: "Survival and runs"},
	{Key: "production", Name: "Production and capacity"},
	{Key: "facilities", Name: "Facilities and upgrades"},
	{Key: "trading", Name: "Trading and market"},
	{Key: "empire", Name: "Rivals and territories"},
	{Key: "oddities", Name: "Oddities"},
}

// Predicate kinds: the closed set of checks. Params live in the Predicate fields
// named in each comment.
const (
	KindNetWorthAtLeast      = "net_worth_at_least"       // N
	KindDayAtLeast           = "day_at_least"             // N
	KindDayAtMost            = "day_at_most"              // N (only useful inside AllOf)
	KindStatAtLeast          = "stat_at_least"            // Stat, N
	KindCashWithNoStock      = "cash_with_no_stock"       // N
	KindStockTotalAtLeast    = "stock_total_at_least"     // N
	KindStockAtLeast         = "stock_at_least"           // Commodity, N
	KindAllWarehousesFull    = "all_warehouses_full"      //
	KindFacilityMaxed        = "facility_maxed"           // Facility
	KindAllOf                = "all_of"                   // All
	KindRunsFinishedAtLeast  = "runs_finished_at_least"   // N
	KindNewPersonalBest      = "new_personal_best"        //
	KindBoardRankAtMost      = "board_rank_at_most"       // N
	KindBoughtInputAtPercent = "bought_input_at_percent"  // N: paid at most N% of base
	KindSoldAtPercentOfBase  = "sold_at_percent_of_base"  // Commodity, N: got at least N% of base
	KindSoldAtPercentOfCost  = "sold_at_percent_of_cost"  // Commodity, N: got at least N% of average cost
	KindSoldDuringEvent      = "sold_during_event"        // Event, Commodity, N
	KindProfitDuringEvent    = "profit_during_event"      // Event
	KindEveryEventSeen       = "every_event_seen"         //
	KindSoldWithoutImpact    = "sold_without_impact"      // Commodity, N
	KindClosingCashBetween   = "closing_cash_between"     // N (min), M (max)
	KindComeback             = "comeback"                 // N (low), M (high)
	KindBankruptHoldingOnly  = "bankrupt_holding_only"    // Commodity
	KindBankruptByDay        = "bankrupt_by_day"          // N
	KindGaveUpWithNetWorth   = "gave_up_with_net_worth"   // N
	KindUpgradesOwned        = "upgrades_owned"           // N: at least N catalog upgrades (territory ones excluded)
	KindRivalsBoughtAtLeast  = "rivals_bought_at_least"   // N
	KindRivalBought          = "rival_bought"             // Key
	KindHostileBuyout        = "hostile_buyout"           //
	KindTerritoryEntered     = "territory_entered"        // Key
	KindTerritoryShare       = "territory_share_at_least" // Key, N (percent)
	KindUpgradeSetOwned      = "upgrade_set_owned"        // Key (a category), or "" for every non-territory upgrade
)

// Stats a StatAtLeast check can read.
const (
	StatProduced              = "produced"
	StatFacilitiesBought      = "facilities_bought"
	StatUpgrades              = "upgrades"
	StatFacilitiesSold        = "facilities_sold"
	StatMostProducedInADay    = "most_produced_in_a_day"
	StatLongestFullProduction = "longest_full_production"
	StatLongestIdle           = "longest_idle"
	StatTradesWithImpact      = "trades_with_impact"
	StatBestDayProfit         = "best_day_profit"
	StatPriceWarsWon          = "price_wars_won"
)

// Predicate is one typed check. Build it with the constructors below; the domain
// validates every row (content test and domain.ValidatePredicate).
type Predicate struct {
	Kind      string
	N, M      int
	Stat      string
	Commodity string
	Event     string
	Facility  string
	// Key names a territory, rival or upgrade category, whichever the kind checks.
	Key string
	All []Predicate
}

func NetWorthAtLeast(n int) Predicate   { return Predicate{Kind: KindNetWorthAtLeast, N: n} }
func DayAtLeast(n int) Predicate        { return Predicate{Kind: KindDayAtLeast, N: n} }
func DayAtMost(n int) Predicate         { return Predicate{Kind: KindDayAtMost, N: n} }
func CashWithNoStock(n int) Predicate   { return Predicate{Kind: KindCashWithNoStock, N: n} }
func StockTotalAtLeast(n int) Predicate { return Predicate{Kind: KindStockTotalAtLeast, N: n} }
func AllWarehousesFull() Predicate      { return Predicate{Kind: KindAllWarehousesFull} }
func EveryEventSeen() Predicate         { return Predicate{Kind: KindEveryEventSeen} }
func NewPersonalBest() Predicate        { return Predicate{Kind: KindNewPersonalBest} }
func BoardRankAtMost(n int) Predicate   { return Predicate{Kind: KindBoardRankAtMost, N: n} }
func BankruptByDay(n int) Predicate     { return Predicate{Kind: KindBankruptByDay, N: n} }
func AllOf(p ...Predicate) Predicate    { return Predicate{Kind: KindAllOf, All: p} }

func StatAtLeast(stat string, n int) Predicate {
	return Predicate{Kind: KindStatAtLeast, Stat: stat, N: n}
}
func StockAtLeast(commodity string, n int) Predicate {
	return Predicate{Kind: KindStockAtLeast, Commodity: commodity, N: n}
}
func RivalsBoughtAtLeast(n int) Predicate { return Predicate{Kind: KindRivalsBoughtAtLeast, N: n} }
func RivalBought(key string) Predicate    { return Predicate{Kind: KindRivalBought, Key: key} }
func HostileBuyout() Predicate            { return Predicate{Kind: KindHostileBuyout} }
func TerritoryEntered(key string) Predicate {
	return Predicate{Kind: KindTerritoryEntered, Key: key}
}
func TerritoryShareAtLeast(key string, percent int) Predicate {
	return Predicate{Kind: KindTerritoryShare, Key: key, N: percent}
}
func UpgradesOwned(n int) Predicate { return Predicate{Kind: KindUpgradesOwned, N: n} }

// UpgradeSetOwned is every upgrade of a category, or every non-territory upgrade for "".
func UpgradeSetOwned(category string) Predicate {
	return Predicate{Kind: KindUpgradeSetOwned, Key: category}
}
func FacilityMaxed(facility string) Predicate {
	return Predicate{Kind: KindFacilityMaxed, Facility: facility}
}
func RunsFinishedAtLeast(n int) Predicate {
	return Predicate{Kind: KindRunsFinishedAtLeast, N: n}
}
func BoughtInputAtPercent(pct int) Predicate {
	return Predicate{Kind: KindBoughtInputAtPercent, N: pct}
}
func SoldAtPercentOfBase(commodity string, pct int) Predicate {
	return Predicate{Kind: KindSoldAtPercentOfBase, Commodity: commodity, N: pct}
}
func SoldAtPercentOfCost(commodity string, pct int) Predicate {
	return Predicate{Kind: KindSoldAtPercentOfCost, Commodity: commodity, N: pct}
}
func SoldDuringEvent(event, commodity string, n int) Predicate {
	return Predicate{Kind: KindSoldDuringEvent, Event: event, Commodity: commodity, N: n}
}
func ProfitDuringEvent(event string) Predicate {
	return Predicate{Kind: KindProfitDuringEvent, Event: event}
}
func SoldWithoutImpact(commodity string, n int) Predicate {
	return Predicate{Kind: KindSoldWithoutImpact, Commodity: commodity, N: n}
}
func ClosingCashBetween(min, max int) Predicate {
	return Predicate{Kind: KindClosingCashBetween, N: min, M: max}
}
func Comeback(low, high int) Predicate {
	return Predicate{Kind: KindComeback, N: low, M: high}
}
func BankruptHoldingOnly(commodity string) Predicate {
	return Predicate{Kind: KindBankruptHoldingOnly, Commodity: commodity}
}
func GaveUpWithNetWorth(n int) Predicate {
	return Predicate{Kind: KindGaveUpWithNetWorth, N: n}
}

// AchievementDef is one achievement. Hidden ones show as "???" until unlocked, so
// their Description is what the player reads afterwards.
type AchievementDef struct {
	Key         string
	Name        string
	Description string
	Category    string
	Tier        string
	Hidden      bool
	Check       Predicate
}

// Achievements is the table: the rows marked ★ in LATE-GAME-CONTENT.md section K,
// which need no new game systems.
var Achievements = []AchievementDef{
	// Wealth
	{Key: "nw_5k", Name: "Pocket money", Description: "Reach a net worth of $5,000.", Category: "wealth", Tier: TierBronze, Check: NetWorthAtLeast(5_000)},
	{Key: "nw_10k", Name: "Five figures", Description: "Reach a net worth of $10,000.", Category: "wealth", Tier: TierBronze, Check: NetWorthAtLeast(10_000)},
	{Key: "nw_25k", Name: "Comfortable", Description: "Reach a net worth of $25,000.", Category: "wealth", Tier: TierBronze, Check: NetWorthAtLeast(25_000)},
	{Key: "nw_100k", Name: "Six figures", Description: "Reach a net worth of $100,000.", Category: "wealth", Tier: TierSilver, Check: NetWorthAtLeast(100_000)},
	{Key: "cash_pile", Name: "Rainy-day fund", Description: "Hold $10,000 in cash with no stock at all.", Category: "wealth", Tier: TierBronze, Check: CashWithNoStock(10_000)},
	{Key: "peak_day", Name: "Record day", Description: "Grow your net worth by $5,000 in a single day.", Category: "wealth", Tier: TierSilver, Check: StatAtLeast(StatBestDayProfit, 5_000)},

	// Survival and runs
	{Key: "day_7", Name: "First week", Description: "Reach day 7.", Category: "survival", Tier: TierBronze, Check: DayAtLeast(7)},
	{Key: "day_30", Name: "Month one", Description: "Reach day 30.", Category: "survival", Tier: TierBronze, Check: DayAtLeast(30)},
	{Key: "day_60", Name: "Seasoned", Description: "Reach day 60.", Category: "survival", Tier: TierSilver, Check: DayAtLeast(60)},
	{Key: "day_100", Name: "Centurion", Description: "Reach day 100.", Category: "survival", Tier: TierSilver, Check: DayAtLeast(100)},
	{Key: "runs_5", Name: "Serial entrepreneur", Description: "Finish 5 runs.", Category: "survival", Tier: TierBronze, Check: RunsFinishedAtLeast(5)},
	{Key: "runs_25", Name: "Can't stop squeezing", Description: "Finish 25 runs.", Category: "survival", Tier: TierSilver, Check: RunsFinishedAtLeast(25)},
	{Key: "new_best", Name: "Personal best", Description: "Finish a run with a better score than your previous best.", Category: "survival", Tier: TierBronze, Check: NewPersonalBest()},
	{Key: "top_10", Name: "Hall of fame", Description: "Finish a run that puts you in the top 10 of the global board.", Category: "survival", Tier: TierGold, Check: BoardRankAtMost(10)},

	// Production and capacity
	{Key: "made_1k", Name: "First thousand", Description: "Make 1,000 lemonade in one run.", Category: "production", Tier: TierBronze, Check: StatAtLeast(StatProduced, 1_000)},
	{Key: "made_10k", Name: "Ten thousand cups", Description: "Make 10,000 lemonade in one run.", Category: "production", Tier: TierSilver, Check: StatAtLeast(StatProduced, 10_000)},
	{Key: "big_batch", Name: "Big batch", Description: "Make 500 lemonade in one day.", Category: "production", Tier: TierSilver, Check: StatAtLeast(StatMostProducedInADay, 500)},
	{Key: "full_house", Name: "Full house", Description: "Fill every warehouse to 100% at the same time.", Category: "production", Tier: TierBronze, Check: AllWarehousesFull()},
	{Key: "stocked_1k", Name: "Stockpile", Description: "Hold 1,000 cases in total.", Category: "production", Tier: TierSilver, Check: StockTotalAtLeast(1_000)},
	{Key: "no_idle", Name: "Humming", Description: "Run production at full capacity 7 days in a row.", Category: "production", Tier: TierSilver, Check: StatAtLeast(StatLongestFullProduction, 7)},

	// Facilities and upgrades
	{Key: "first_expand", Name: "Growing", Description: "Build a building.", Category: "facilities", Tier: TierBronze, Check: StatAtLeast(StatFacilitiesBought, 1)},
	{Key: "first_upgrade_item", Name: "Gadget", Description: "Buy an upgrade.", Category: "facilities", Tier: TierBronze, Check: UpgradesOwned(1)},
	{Key: "freshness_set", Name: "Ice cold", Description: "Own every freshness upgrade.", Category: "facilities", Tier: TierSilver, Check: UpgradeSetOwned(UpFreshness)},
	{Key: "upgrades_all", Name: "Fully loaded", Description: "Own every upgrade.", Category: "facilities", Tier: TierGold, Check: UpgradeSetOwned("")},
	{Key: "first_upgrade", Name: "Moving up", Description: "Upgrade a facility.", Category: "facilities", Tier: TierBronze, Check: StatAtLeast(StatUpgrades, 1)},
	{Key: "max_production", Name: "Factory floor", Description: "Production at the top level with the most buildings allowed.", Category: "facilities", Tier: TierSilver, Check: FacilityMaxed("production")},
	{Key: "max_warehouse", Name: "Room to spare", Description: "Every warehouse at the top level with the most buildings allowed.", Category: "facilities", Tier: TierSilver, Check: FacilityMaxed("warehouse")},
	{Key: "all_maxed", Name: "Everything maxed", Description: "Max out production and every warehouse at once.", Category: "facilities", Tier: TierGold, Check: AllOf(FacilityMaxed("production"), FacilityMaxed("warehouse"))},
	{Key: "sold_building", Name: "Downsizing", Description: "Sell a building.", Category: "facilities", Tier: TierBronze, Check: StatAtLeast(StatFacilitiesSold, 1)},

	// Rivals and territories
	{Key: "first_buyout", Name: "Acquisition", Description: "Buy out a rival.", Category: "empire", Tier: TierBronze, Check: RivalsBoughtAtLeast(1)},
	{Key: "lucy_bought", Name: "Lemonade from a kid", Description: "Buy Lil' Lucy's stand.", Category: "empire", Tier: TierBronze, Check: RivalBought("lil_lucy")},
	{Key: "own_neighborhood", Name: "Block boss", Description: "Hold 100% of the Neighborhood.", Category: "empire", Tier: TierSilver, Check: TerritoryShareAtLeast("neighborhood", 100)},
	{Key: "enter_city", Name: "Big city", Description: "Enter Citrus City.", Category: "empire", Tier: TierBronze, Check: TerritoryEntered("city")},
	{Key: "own_city", Name: "Mayor of lemonade", Description: "Hold 50% of Citrus City.", Category: "empire", Tier: TierSilver, Check: TerritoryShareAtLeast("city", 50)},
	{Key: "enter_region", Name: "Regional", Description: "Enter the Sunbelt Region.", Category: "empire", Tier: TierSilver, Check: TerritoryEntered("region")},
	{Key: "enter_nation", Name: "National brand", Description: "Enter the Nation.", Category: "empire", Tier: TierSilver, Check: TerritoryEntered("nation")},
	{Key: "enter_world", Name: "Going global", Description: "Enter the World.", Category: "empire", Tier: TierGold, Check: TerritoryEntered("world")},
	{Key: "hostile", Name: "Hostile takeover", Description: "Complete a hostile buyout.", Category: "empire", Tier: TierSilver, Check: HostileBuyout()},
	{Key: "price_war_won", Name: "Price warrior", Description: "Come out of a price war with at least the share you had going in.", Category: "empire", Tier: TierSilver, Check: StatAtLeast(StatPriceWarsWon, 1)},

	// Trading and market
	{Key: "buy_low", Name: "Bargain hunter", Description: "Buy an ingredient at 70% of its base price or less.", Category: "trading", Tier: TierBronze, Check: BoughtInputAtPercent(70)},
	{Key: "sell_high", Name: "Top of the market", Description: "Sell lemonade at 150% of its base price or more.", Category: "trading", Tier: TierSilver, Check: SoldAtPercentOfBase("lemonade", 150)},
	{Key: "heat_seller", Name: "Hot seller", Description: "Sell 100 lemonade during Heat Waves in one run.", Category: "trading", Tier: TierBronze, Check: SoldDuringEvent("heat_wave", "lemonade", 100)},
	{Key: "rain_profit", Name: "Singing in the rain", Description: "End a Rainy Week day richer than you started it.", Category: "trading", Tier: TierSilver, Check: ProfitDuringEvent("rainy_week")},
	{Key: "every_event", Name: "Seen it all", Description: "See every market event in one run.", Category: "trading", Tier: TierSilver, Check: EveryEventSeen()},
	{Key: "no_slippage", Name: "Light touch", Description: "Sell 500 lemonade in one run without ever paying price impact on a sale.", Category: "trading", Tier: TierSilver, Check: SoldWithoutImpact("lemonade", 500)},
	{Key: "volume_trader", Name: "Market mover", Description: "Pay price impact for the first time: you traded past the market's depth.", Category: "trading", Tier: TierBronze, Check: StatAtLeast(StatTradesWithImpact, 1)},
	{Key: "avg_cost_win", Name: "Bought right", Description: "Sell lemonade for at least twice its average cost.", Category: "trading", Tier: TierSilver, Check: SoldAtPercentOfCost("lemonade", 200)},

	// Oddities
	{Key: "just_ice", Name: "Just ice", Description: "Go bankrupt holding nothing but ice.", Category: "oddities", Tier: TierBronze, Hidden: true, Check: BankruptHoldingOnly("ice")},
	{Key: "close_call", Name: "Close call", Description: "Survive an end of day with $1 to $9 left.", Category: "oddities", Tier: TierSilver, Check: ClosingCashBetween(1, 9)},
	{Key: "comeback", Name: "Lemons into lemonade", Description: "Close a day with under $100 in cash and stock, then reach a net worth of $10,000 in the same run.", Category: "oddities", Tier: TierGold, Check: Comeback(100, 10_000)},
	{Key: "speedrun", Name: "Speedrun", Description: "Reach a net worth of $10,000 by day 15.", Category: "oddities", Tier: TierSilver, Check: AllOf(NetWorthAtLeast(10_000), DayAtMost(15))},
	{Key: "patience", Name: "Patience", Description: "End 5 days in a row without trading.", Category: "oddities", Tier: TierBronze, Hidden: true, Check: StatAtLeast(StatLongestIdle, 5)},
	{Key: "give_up_rich", Name: "Quit while ahead", Description: "Give up with a net worth of $50,000 or more.", Category: "oddities", Tier: TierBronze, Hidden: true, Check: GaveUpWithNetWorth(50_000)},
	{Key: "day_one_loss", Name: "Tough crowd", Description: "Go bankrupt on day 1, 2 or 3.", Category: "oddities", Tier: TierBronze, Hidden: true, Check: BankruptByDay(3)},
	{Key: "sugar_rush", Name: "Sugar rush", Description: "Hold 500 sugar.", Category: "oddities", Tier: TierBronze, Check: StockAtLeast("sugar", 500)},
}
