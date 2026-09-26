package domain

import (
	"reflect"
	"strings"
	"testing"

	"lemonade-api/internal/domain/content"
)

func TestEveryAchievementPredicateIsValid(t *testing.T) {
	cfg := DefaultConfig()
	for _, d := range content.Achievements {
		if err := ValidatePredicate(d.Check, cfg); err != nil {
			t.Errorf("%s: %v", d.Key, err)
		}
	}
}

func TestValidatePredicateRejectsBadRows(t *testing.T) {
	cfg := DefaultConfig()
	bad := []content.Predicate{
		{Kind: "made_up"},
		content.NetWorthAtLeast(0),
		content.StatAtLeast("nope", 1),
		content.StockAtLeast("caviar", 5),
		content.SoldDuringEvent("meteor", "lemonade", 1),
		content.ProfitDuringEvent("meteor"),
		content.FacilityMaxed("castle"),
		content.AllOf(),
		content.AllOf(content.DayAtLeast(3), content.NetWorthAtLeast(-1)),
		content.ClosingCashBetween(9, 1),
		content.Comeback(100, 100),
		content.BoughtInputAtPercent(0),
		content.SoldAtPercentOfBase("lemonade", 5000),
	}
	for _, p := range bad {
		if ValidatePredicate(p, cfg) == nil {
			t.Errorf("%+v should be invalid", p)
		}
	}
}

// holdsFor evaluates one predicate on a before/after pair.
func holdsFor(p content.Predicate, before, after Game, ctx AchievementContext) bool {
	defs := []content.AchievementDef{{Key: "k", Check: p}}
	return len(Evaluate(defs, before, after, DefaultConfig(), ctx)) == 1
}

func intp(n int) *int { return &n }

func TestPredicatesAtTheirBoundaries(t *testing.T) {
	cfg := DefaultConfig()
	fresh := NewGame(cfg, 42) // net worth $1,500: $1,000 cash + $500 of buildings

	edit := func(f func(g *Game)) Game {
		g := fresh.Clone()
		f(&g)
		return g
	}
	endedDay := func(f func(g *Game)) Game { // the day advanced
		return edit(func(g *Game) { g.Day = fresh.Day + 1; f(g) })
	}
	maxWarehouses := func(g *Game) {
		g.WarehouseLevel = cfg.MaxLevel
		for _, r := range cfg.Resources() {
			g.WarehouseQty[r] = cfg.MaxQuantity
		}
	}
	maxProduction := func(g *Game) { g.ProductionLevel, g.ProductionQty = cfg.MaxLevel, cfg.MaxQuantity }
	fill := func(g *Game) {
		for _, r := range cfg.Resources() {
			g.Inventory[r] = Capacity(*g, cfg, r)
		}
	}
	rainy := edit(func(g *Game) { g.Events = []ActiveEvent{{Key: "rainy_week", DaysLeft: 2}} })
	onlyIce := edit(func(g *Game) { g.Inventory[Ice] = 4 })

	tests := []struct {
		name          string
		p             content.Predicate
		before, after Game
		ctx           AchievementContext
		want          bool
	}{
		{"net worth at", content.NetWorthAtLeast(1500), fresh, fresh, AchievementContext{}, true},
		{"net worth under", content.NetWorthAtLeast(1501), fresh, fresh, AchievementContext{}, false},
		{"day at", content.DayAtLeast(7), fresh, edit(func(g *Game) { g.Day = 7 }), AchievementContext{}, true},
		{"day under", content.DayAtLeast(7), fresh, edit(func(g *Game) { g.Day = 6 }), AchievementContext{}, false},
		{"day at most", content.DayAtMost(15), fresh, edit(func(g *Game) { g.Day = 15 }), AchievementContext{}, true},
		{"day past", content.DayAtMost(15), fresh, edit(func(g *Game) { g.Day = 16 }), AchievementContext{}, false},
		{"stat at", content.StatAtLeast(content.StatProduced, 1000), fresh, edit(func(g *Game) { g.Stats.Produced = 1000 }), AchievementContext{}, true},
		{"stat under", content.StatAtLeast(content.StatProduced, 1000), fresh, edit(func(g *Game) { g.Stats.Produced = 999 }), AchievementContext{}, false},
		{"goal stat", content.StatAtLeast(content.StatLongestIdle, 5), fresh, edit(func(g *Game) { g.Goals.LongestIdle = 5 }), AchievementContext{}, true},
		{"cash, no stock", content.CashWithNoStock(10000), fresh, edit(func(g *Game) { g.Capital = 10000 }), AchievementContext{}, true},
		{"cash, one case", content.CashWithNoStock(10000), fresh, edit(func(g *Game) { g.Capital = 10000; g.Inventory[Cup] = 1 }), AchievementContext{}, false},
		{"cash short", content.CashWithNoStock(10000), fresh, edit(func(g *Game) { g.Capital = 9999 }), AchievementContext{}, false},
		{"stock total at", content.StockTotalAtLeast(1000), fresh, edit(func(g *Game) { g.Inventory[Lemon], g.Inventory[Cup] = 600, 400 }), AchievementContext{}, true},
		{"stock total under", content.StockTotalAtLeast(1000), fresh, edit(func(g *Game) { g.Inventory[Lemon], g.Inventory[Cup] = 600, 399 }), AchievementContext{}, false},
		{"stock of one", content.StockAtLeast("sugar", 500), fresh, edit(func(g *Game) { g.Inventory[Sugar] = 500 }), AchievementContext{}, true},
		{"stock of one under", content.StockAtLeast("sugar", 500), fresh, edit(func(g *Game) { g.Inventory[Sugar] = 499 }), AchievementContext{}, false},
		{"all full", content.AllWarehousesFull(), fresh, edit(fill), AchievementContext{}, true},
		{"all but one case", content.AllWarehousesFull(), fresh, edit(func(g *Game) { fill(g); g.Inventory[Ice]-- }), AchievementContext{}, false},
		{"production maxed", content.FacilityMaxed("production"), fresh, edit(maxProduction), AchievementContext{}, true},
		{"production one short", content.FacilityMaxed("production"), fresh, edit(func(g *Game) { maxProduction(g); g.ProductionQty-- }), AchievementContext{}, false},
		{"warehouses maxed", content.FacilityMaxed("warehouse"), fresh, edit(maxWarehouses), AchievementContext{}, true},
		{"one warehouse short", content.FacilityMaxed("warehouse"), fresh, edit(func(g *Game) { maxWarehouses(g); g.WarehouseQty[Cup]-- }), AchievementContext{}, false},
		{"all of", content.AllOf(content.FacilityMaxed("production"), content.FacilityMaxed("warehouse")), fresh, edit(func(g *Game) { maxProduction(g); maxWarehouses(g) }), AchievementContext{}, true},
		{"all of, one part", content.AllOf(content.FacilityMaxed("production"), content.FacilityMaxed("warehouse")), fresh, edit(maxProduction), AchievementContext{}, false},
		{"runs at", content.RunsFinishedAtLeast(5), fresh, fresh, AchievementContext{RunsFinished: 5}, true},
		{"runs under", content.RunsFinishedAtLeast(5), fresh, fresh, AchievementContext{RunsFinished: 4}, false},
		{"new best", content.NewPersonalBest(), fresh, fresh, AchievementContext{PreviousBest: intp(900), FinishedScore: intp(901)}, true},
		{"tied best", content.NewPersonalBest(), fresh, fresh, AchievementContext{PreviousBest: intp(900), FinishedScore: intp(900)}, false},
		{"first run is no new best", content.NewPersonalBest(), fresh, fresh, AchievementContext{FinishedScore: intp(900)}, false},
		{"rank 10", content.BoardRankAtMost(10), fresh, fresh, AchievementContext{BoardRank: 10}, true},
		{"rank 11", content.BoardRankAtMost(10), fresh, fresh, AchievementContext{BoardRank: 11}, false},
		{"unranked", content.BoardRankAtMost(10), fresh, fresh, AchievementContext{}, false},
		{"bought at 70%", content.BoughtInputAtPercent(70), fresh, edit(func(g *Game) { g.Goals.LowestInputBuyPercent = 70 }), AchievementContext{}, true},
		{"bought at 71%", content.BoughtInputAtPercent(70), fresh, edit(func(g *Game) { g.Goals.LowestInputBuyPercent = 71 }), AchievementContext{}, false},
		{"never bought", content.BoughtInputAtPercent(70), fresh, fresh, AchievementContext{}, false},
		{"sold at 150%", content.SoldAtPercentOfBase("lemonade", 150), fresh, edit(func(g *Game) { g.Goals.BestSellPercent = map[string]int{"lemonade": 150} }), AchievementContext{}, true},
		{"sold at 149%", content.SoldAtPercentOfBase("lemonade", 150), fresh, edit(func(g *Game) { g.Goals.BestSellPercent = map[string]int{"lemonade": 149, "lemon": 300} }), AchievementContext{}, false},
		{"sold at twice cost", content.SoldAtPercentOfCost("lemonade", 200), fresh, edit(func(g *Game) { g.Goals.BestCostPercent = map[string]int{"lemonade": 200} }), AchievementContext{}, true},
		{"sold under twice cost", content.SoldAtPercentOfCost("lemonade", 200), fresh, edit(func(g *Game) { g.Goals.BestCostPercent = map[string]int{"lemonade": 199} }), AchievementContext{}, false},
		{"heat sales at", content.SoldDuringEvent("heat_wave", "lemonade", 100), fresh, edit(func(g *Game) { g.Goals.EventSales = map[string]int{"heat_wave/lemonade": 100} }), AchievementContext{}, true},
		{"heat sales under", content.SoldDuringEvent("heat_wave", "lemonade", 100), fresh, edit(func(g *Game) { g.Goals.EventSales = map[string]int{"heat_wave/lemonade": 99, "holiday/lemonade": 500} }), AchievementContext{}, false},
		{"rain profit", content.ProfitDuringEvent("rainy_week"), rainy, endedDay(func(g *Game) { g.Goals.DaysClosed, g.Goals.LastDayProfit = 3, 1 }), AchievementContext{}, true},
		{"rain, flat day", content.ProfitDuringEvent("rainy_week"), rainy, endedDay(func(g *Game) { g.Goals.DaysClosed, g.Goals.LastDayProfit = 3, 0 }), AchievementContext{}, false},
		{"profit, no rain", content.ProfitDuringEvent("rainy_week"), fresh, endedDay(func(g *Game) { g.Goals.DaysClosed, g.Goals.LastDayProfit = 3, 50 }), AchievementContext{}, false},
		{"rain, no day ended", content.ProfitDuringEvent("rainy_week"), rainy, edit(func(g *Game) { g.Events = rainy.Events; g.Goals.DaysClosed, g.Goals.LastDayProfit = 3, 50 }), AchievementContext{}, false},
		{"every event", content.EveryEventSeen(), fresh, edit(func(g *Game) {
			for _, e := range cfg.Events {
				g.Goals.EventsSeen = append(g.Goals.EventsSeen, e.Key)
			}
		}), AchievementContext{}, true},
		{"all but one event", content.EveryEventSeen(), fresh, edit(func(g *Game) {
			for _, e := range cfg.Events[1:] {
				g.Goals.EventsSeen = append(g.Goals.EventsSeen, e.Key)
			}
		}), AchievementContext{}, false},
		{"no slippage at", content.SoldWithoutImpact("lemonade", 500), fresh, edit(func(g *Game) { g.Goals.Sold = map[string]int{"lemonade": 500} }), AchievementContext{}, true},
		{"no slippage, one impact", content.SoldWithoutImpact("lemonade", 500), fresh, edit(func(g *Game) { g.Goals.Sold = map[string]int{"lemonade": 900}; g.Goals.SalesWithImpact = 1 }), AchievementContext{}, false},
		{"no slippage under", content.SoldWithoutImpact("lemonade", 500), fresh, edit(func(g *Game) { g.Goals.Sold = map[string]int{"lemonade": 499} }), AchievementContext{}, false},
		{"close call $1", content.ClosingCashBetween(1, 9), fresh, endedDay(func(g *Game) { g.Capital = 1 }), AchievementContext{}, true},
		{"close call $9", content.ClosingCashBetween(1, 9), fresh, endedDay(func(g *Game) { g.Capital = 9 }), AchievementContext{}, true},
		{"close call $10", content.ClosingCashBetween(1, 9), fresh, endedDay(func(g *Game) { g.Capital = 10 }), AchievementContext{}, false},
		{"close call $0", content.ClosingCashBetween(1, 9), fresh, endedDay(func(g *Game) { g.Capital = 0 }), AchievementContext{}, false},
		{"$5 mid-day", content.ClosingCashBetween(1, 9), fresh, edit(func(g *Game) { g.Capital = 5 }), AchievementContext{}, false},
		{"comeback", content.Comeback(100, 10000), fresh, edit(func(g *Game) { g.Goals.DaysClosed, g.Goals.LowestClosingLiquid, g.Capital = 4, 99, 9500 }), AchievementContext{}, true},
		{"comeback from $100", content.Comeback(100, 10000), fresh, edit(func(g *Game) { g.Goals.DaysClosed, g.Goals.LowestClosingLiquid, g.Capital = 4, 100, 9500 }), AchievementContext{}, false},
		{"comeback not there yet", content.Comeback(100, 10000), fresh, edit(func(g *Game) { g.Goals.DaysClosed, g.Goals.LowestClosingLiquid, g.Capital = 4, 50, 9499 }), AchievementContext{}, false},
		{"comeback untracked", content.Comeback(100, 10000), fresh, edit(func(g *Game) { g.Capital = 9500 }), AchievementContext{}, false},
		{"bankrupt on ice", content.BankruptHoldingOnly("ice"), onlyIce, edit(func(g *Game) { g.Status = StatusBankrupt }), AchievementContext{}, true},
		{"bankrupt on ice and a lemon", content.BankruptHoldingOnly("ice"), edit(func(g *Game) { g.Inventory[Ice], g.Inventory[Lemon] = 4, 1 }), edit(func(g *Game) { g.Status = StatusBankrupt }), AchievementContext{}, false},
		{"only ice, still going", content.BankruptHoldingOnly("ice"), onlyIce, onlyIce, AchievementContext{}, false},
		{"bankrupt day 3", content.BankruptByDay(3), fresh, edit(func(g *Game) { g.Day, g.Status = 3, StatusBankrupt }), AchievementContext{}, true},
		{"bankrupt day 4", content.BankruptByDay(3), fresh, edit(func(g *Game) { g.Day, g.Status = 4, StatusBankrupt }), AchievementContext{}, false},
		{"alive day 2", content.BankruptByDay(3), fresh, edit(func(g *Game) { g.Day = 2 }), AchievementContext{}, false},
		{"gave up rich", content.GaveUpWithNetWorth(50000), fresh, edit(func(g *Game) { g.Status, g.Capital = StatusGaveUp, 49500 }), AchievementContext{}, true},
		{"gave up a dollar short", content.GaveUpWithNetWorth(50000), fresh, edit(func(g *Game) { g.Status, g.Capital = StatusGaveUp, 49499 }), AchievementContext{}, false},
		{"rich, still going", content.GaveUpWithNetWorth(50000), fresh, edit(func(g *Game) { g.Capital = 90000 }), AchievementContext{}, false},
	}
	for _, tt := range tests {
		if got := holdsFor(tt.p, tt.before, tt.after, tt.ctx); got != tt.want {
			t.Errorf("%s: holds = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestEvaluateReturnsEachKeyOnceAndSkipsUnlocked(t *testing.T) {
	cfg := DefaultConfig()
	g := NewGame(cfg, 1)
	g.Day, g.Capital = 30, 30000
	defs := []content.AchievementDef{
		{Key: "day_7", Check: content.DayAtLeast(7)},
		{Key: "day_7", Check: content.DayAtLeast(1)}, // a duplicate row cannot grant twice
		{Key: "day_30", Check: content.DayAtLeast(30)},
		{Key: "nw_10k", Check: content.NetWorthAtLeast(10000)},
		{Key: "day_60", Check: content.DayAtLeast(60)},
	}
	got := Evaluate(defs, g, g, cfg, AchievementContext{Unlocked: map[string]bool{"day_30": true}})
	if want := []string{"day_7", "nw_10k"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Evaluate = %v, want %v", got, want)
	}
}

func TestProgressIsCappedAndOnlyForMeasurableChecks(t *testing.T) {
	cfg := DefaultConfig()
	g := NewGame(cfg, 1)
	g.Stats.Produced = 6200
	if cur, target, ok := Progress(content.StatAtLeast(content.StatProduced, 10000), g, cfg, AchievementContext{}); !ok || cur != 6200 || target != 10000 {
		t.Fatalf("produced progress = %d/%d %v", cur, target, ok)
	}
	if cur, target, ok := Progress(content.NetWorthAtLeast(1000), g, cfg, AchievementContext{}); !ok || cur != 1000 || target != 1000 {
		t.Fatalf("capped progress = %d/%d %v", cur, target, ok)
	}
	if cur, target, ok := Progress(content.RunsFinishedAtLeast(5), g, cfg, AchievementContext{RunsFinished: 2}); !ok || cur != 2 || target != 5 {
		t.Fatalf("runs progress = %d/%d %v", cur, target, ok)
	}
	g.Goals.EventsSeen = []string{"holiday", "heat_wave"}
	if cur, target, ok := Progress(content.EveryEventSeen(), g, cfg, AchievementContext{}); !ok || cur != 2 || target != len(cfg.Events) {
		t.Fatalf("events progress = %d/%d %v", cur, target, ok)
	}
	g.Goals.Sold = map[string]int{"lemonade": 300}
	if _, _, ok := Progress(content.SoldWithoutImpact("lemonade", 500), g, cfg, AchievementContext{}); !ok {
		t.Fatal("no-slippage progress should show before any impact")
	}
	g.Goals.SalesWithImpact = 1
	if _, _, ok := Progress(content.SoldWithoutImpact("lemonade", 500), g, cfg, AchievementContext{}); ok {
		t.Fatal("no-slippage progress is meaningless once a sale paid impact")
	}
	for _, p := range []content.Predicate{content.Comeback(100, 10000), content.AllWarehousesFull(), content.NewPersonalBest(), content.ClosingCashBetween(1, 9)} {
		if _, _, ok := Progress(p, g, cfg, AchievementContext{}); ok {
			t.Errorf("%s should have no progress bar", p.Kind)
		}
	}
}

func TestTradesRecordGoalFacts(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 100000
	g.WarehouseQty[Sugar] = 10
	g.Events = []ActiveEvent{{Key: "heat_wave", Multipliers: map[Resource]float64{Lemonade: 1.4, Ice: 1.3}, DaysLeft: 1}}

	// Sugar asks $11 on a $10 base: 110%, rounded up.
	if err := Buy(&g, cfg, Sugar, 5); err != nil {
		t.Fatal(err)
	}
	if g.Goals.LowestInputBuyPercent != 110 || g.Goals.LastTradeDay != 1 || g.Goals.TradesWithImpact != 0 {
		t.Fatalf("after a plain buy: %+v", g.Goals)
	}
	// Past the free depth of 80 the buy pays impact.
	if err := Buy(&g, cfg, Sugar, 85); err != nil {
		t.Fatal(err)
	}
	if g.Goals.TradesWithImpact != 1 || g.Goals.SalesWithImpact != 0 {
		t.Fatalf("an impact buy: %+v", g.Goals)
	}
	// Buying lemonade is not buying an input.
	g.Market[Lemonade].Price = 40
	if err := Buy(&g, cfg, Lemonade, 1); err != nil {
		t.Fatal(err)
	}
	if g.Goals.LowestInputBuyPercent != 110 {
		t.Fatalf("lemonade counted as an input: %d", g.Goals.LowestInputBuyPercent)
	}

	// Lemonade made from $50 of inputs, sold in a heat wave at $113 (bid of 90 x 1.4 = 126).
	g.Market[Lemonade].Price = 90
	g.Inventory[Lemonade] = 3
	g.CostBasis[Lemonade] = 150
	if err := Sell(&g, cfg, Lemonade, 2); err != nil {
		t.Fatal(err)
	}
	bid := Quotes(g, cfg)[Lemonade].Bid
	if g.Goals.Sold["lemonade"] != 2 || g.Goals.EventSales["heat_wave/lemonade"] != 2 {
		t.Fatalf("sale counts: %+v", g.Goals)
	}
	if want := 2 * bid * 100 / (2 * 90); g.Goals.BestSellPercent["lemonade"] != want {
		t.Fatalf("sell percent = %d, want %d", g.Goals.BestSellPercent["lemonade"], want)
	}
	if want := 2 * bid * 100 / 100; g.Goals.BestCostPercent["lemonade"] != want {
		t.Fatalf("cost percent = %d, want %d", g.Goals.BestCostPercent["lemonade"], want)
	}

	// A failed trade records nothing.
	before := g.Goals.clone()
	if err := Sell(&g, cfg, Lemonade, 99); err == nil {
		t.Fatal("expected insufficient stock")
	}
	if !reflect.DeepEqual(before, g.Goals) {
		t.Fatal("a failed sale changed the goal facts")
	}
}

func TestBigSalesCountImpact(t *testing.T) {
	g, cfg := newTestGame()
	g.WarehouseQty[Lemonade] = 10
	g.Inventory[Lemonade] = 100
	if err := Sell(&g, cfg, Lemonade, 100); err != nil {
		t.Fatal(err)
	}
	if g.Goals.SalesWithImpact != 1 || g.Goals.TradesWithImpact != 1 {
		t.Fatalf("selling past the depth: %+v", g.Goals)
	}
}

func TestRecordDayFacts(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 5000
	day := func(trade bool) DayReport {
		t.Helper()
		if trade {
			_ = SellClamped(&g, cfg, Lemonade, 100) // room for tonight's batch
			for _, r := range cfg.Inputs() {
				if err := Buy(&g, cfg, r, 10); err != nil {
					t.Fatal(err)
				}
			}
		}
		before := g.Clone()
		report, err := EndDay(&g, cfg)
		if err != nil {
			t.Fatal(err)
		}
		RecordDayFacts(before, &g, cfg, report)
		return report
	}

	day(false)
	day(false)
	if g.Goals.DaysClosed != 2 || g.Goals.IdleStreak != 2 || g.Goals.LongestIdle != 2 || g.Goals.FullProductionStreak != 0 {
		t.Fatalf("two idle days: %+v", g.Goals)
	}
	r := day(true) // full production: 10 of 10
	if r.Produced != 10 || g.Goals.FullProductionStreak != 1 || g.Goals.IdleStreak != 0 || g.Goals.LongestIdle != 2 || g.Goals.MostProducedInADay != 10 {
		t.Fatalf("a busy day: produced %d, %+v", r.Produced, g.Goals)
	}
	day(true)
	if g.Goals.FullProductionStreak != 2 || g.Goals.LongestFullProduction != 2 {
		t.Fatalf("a second busy day: %+v", g.Goals)
	}
	if g.Goals.LastClosingNetWorth != NetWorth(g, cfg) {
		t.Fatalf("closing net worth %d, want %d", g.Goals.LastClosingNetWorth, NetWorth(g, cfg))
	}
	if g.Goals.LowestClosingLiquid <= 0 || g.Goals.LowestClosingLiquid > 5000 {
		t.Fatalf("lowest closing liquid = %d", g.Goals.LowestClosingLiquid)
	}

	// Profit is closing to closing net worth.
	before := g.Clone()
	opening := g.Goals.LastClosingNetWorth
	g.Capital += 5000 // as if a windfall came during the day
	report, _ := EndDay(&g, cfg)
	RecordDayFacts(before, &g, cfg, report)
	if want := NetWorth(g, cfg) - opening; g.Goals.LastDayProfit != want || g.Goals.BestDayProfit < want {
		t.Fatalf("day profit %d (best %d), want %d", g.Goals.LastDayProfit, g.Goals.BestDayProfit, want)
	}

	// Events active before or after the night count as seen.
	before = g.Clone()
	g.Events = []ActiveEvent{{Key: "holiday", DaysLeft: 1}}
	before.Events = []ActiveEvent{{Key: "cup_shortage", DaysLeft: 1}}
	RecordDayFacts(before, &g, cfg, DayReport{Day: g.Day})
	if !reflect.DeepEqual(g.Goals.EventsSeen, []string{"cup_shortage", "holiday"}) {
		t.Fatalf("events seen = %v", g.Goals.EventsSeen)
	}
}

func TestGoalFactsNeverChangeHowTheGamePlays(t *testing.T) {
	cfg := DefaultConfig()
	play := func(withFacts bool) Game {
		g := NewGame(cfg, 7)
		for d := 0; d < 40; d++ {
			for _, r := range cfg.Inputs() {
				_ = BuyClamped(&g, cfg, r, 10)
			}
			_ = SellClamped(&g, cfg, Lemonade, 100)
			before := g.Clone()
			report, err := EndDay(&g, cfg)
			if err != nil {
				break
			}
			if withFacts {
				RecordDayFacts(before, &g, cfg, report)
			}
		}
		g.Goals = GoalStats{}
		return g
	}
	if a, b := play(true), play(false); !reflect.DeepEqual(a, b) {
		t.Fatal("recording goal facts changed the game")
	}
}

func TestCloneCopiesGoalFacts(t *testing.T) {
	g, _ := newTestGame()
	g.Goals.Sold = map[string]int{"lemon": 1}
	g.Goals.EventsSeen = []string{"holiday"}
	c := g.Clone()
	c.Goals.Sold["lemon"] = 9
	c.Goals.EventsSeen[0] = "x"
	if g.Goals.Sold["lemon"] != 1 || g.Goals.EventsSeen[0] != "holiday" {
		t.Fatal("clone aliases the goal facts")
	}
}

func TestProvenByRuns(t *testing.T) {
	run := func(score, days int, endedBy string, stats Stats) RunRecord {
		return RunRecord{Score: score, NetWorth: score, Days: days, EndedBy: endedBy, Stats: stats}
	}
	runs := []RunRecord{
		run(1200, 2, EndedByBankrupt, Stats{}),                                                       // 0
		run(6000, 40, EndedByGaveUp, Stats{Produced: 1500, FacilitiesBought: 2, PeakCapital: 12000}), // 1
		run(5000, 9, EndedByGaveUp, Stats{Upgrades: 1}),                                              // 2
		run(11000, 12, EndedByGaveUp, Stats{FacilitiesSold: 1}),                                      // 3
		run(60000, 120, EndedByGaveUp, Stats{}),                                                      // 4
	}
	got := map[string]int{}
	for _, p := range ProvenByRuns(content.Achievements, runs, 7) {
		if _, dup := got[p.Key]; dup {
			t.Fatalf("%s granted twice", p.Key)
		}
		got[p.Key] = p.Run
	}
	want := map[string]int{
		"nw_5k": 1, "nw_10k": 1, "nw_25k": 4, // run 1's cash peak proves $10k
		"day_7": 1, "day_30": 1, "day_60": 4, "day_100": 4,
		"runs_5":   4,
		"new_best": 1,
		"top_10":   4, // the best run carries the board rank
		"made_1k":  1, "first_expand": 1, "first_upgrade": 2, "sold_building": 3,
		"day_one_loss": 0, "give_up_rich": 4,
		"speedrun": 3, // $11,000 at the end of day 12
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("proven:\n got %v\nwant %v", got, want)
	}
	if len(ProvenByRuns(content.Achievements, runs, 11)) != len(want)-1 {
		t.Fatal("rank 11 should not prove top 10")
	}
	if len(ProvenByRuns(content.Achievements, nil, 1)) != 0 {
		t.Fatal("no runs prove nothing")
	}
}

func TestHiddenAchievementsReadWellOnceUnlocked(t *testing.T) {
	for _, d := range content.Achievements {
		if d.Hidden && (len(d.Description) < 10 || !strings.HasSuffix(d.Description, ".")) {
			t.Errorf("%s: a hidden achievement needs a full description for after it unlocks", d.Key)
		}
	}
}

func TestUpgradeAchievementPredicates(t *testing.T) {
	fresh := NewGame(DefaultConfig(), 42)
	own := func(keys ...string) Game {
		g := fresh.Clone()
		for _, k := range keys {
			g.Upgrades[k] = 1
		}
		return g
	}
	var freshness, all []string
	for _, u := range content.Upgrades {
		all = append(all, u.Key)
		if u.Category == content.UpFreshness {
			freshness = append(freshness, u.Key)
		}
	}
	cases := []struct {
		name string
		p    content.Predicate
		g    Game
		want bool
	}{
		{"none owned", content.UpgradesOwned(1), own(), false},
		{"one owned", content.UpgradesOwned(1), own("painted_stand"), true},
		{"a freshness set short by one", content.UpgradeSetOwned(content.UpFreshness), own(freshness[1:]...), false},
		{"the whole freshness set", content.UpgradeSetOwned(content.UpFreshness), own(freshness...), true},
		{"freshness only is not all", content.UpgradeSetOwned(""), own(freshness...), false},
		{"every upgrade", content.UpgradeSetOwned(""), own(all...), true},
	}
	for _, c := range cases {
		if got := holdsFor(c.p, fresh, c.g, AchievementContext{}); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	if err := ValidatePredicate(content.UpgradeSetOwned("nope"), DefaultConfig()); err == nil {
		t.Error("an unknown category should be rejected")
	}
}
