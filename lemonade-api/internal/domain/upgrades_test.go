package domain

import (
	"errors"
	"math"
	"math/rand"
	"reflect"
	"testing"

	"lemonade-api/internal/domain/content"
)

// own gives a game upgrades without paying, so a test can isolate one effect.
func own(g *Game, keys ...string) {
	for _, k := range keys {
		g.Upgrades[k] = 1
	}
}

func rich(g *Game) { g.Capital = 1 << 40 }

func TestBuyUpgradeChargesAndAddsUpkeep(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 5000
	before := TotalUpkeep(g, cfg)
	if err := BuyUpgrade(&g, cfg, "freezer_1"); err != nil {
		t.Fatal(err)
	}
	if g.Capital != 5000-1200 || !g.Owns("freezer_1") || g.UpgradeSpend != 1200 {
		t.Fatalf("capital %d, owned %v, spend %d", g.Capital, g.Upgrades, g.UpgradeSpend)
	}
	if got := TotalUpkeep(g, cfg); got != before+5 {
		t.Fatalf("upkeep %d, want %d", got, before+5)
	}
}

func TestBuyUpgradeErrors(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 500
	if err := BuyUpgrade(&g, cfg, "nope"); !errors.Is(err, ErrUnknownUpgrade) {
		t.Errorf("unknown: %v", err)
	}
	if err := BuyUpgrade(&g, cfg, "freezer_1"); !errors.Is(err, ErrInsufficientFunds) {
		t.Errorf("funds: %v", err)
	}
	g.Capital = 100_000
	var lock *UpgradeLockedError
	if err := BuyUpgrade(&g, cfg, "freezer_2"); !errors.Is(err, ErrUpgradeLocked) || !errors.As(err, &lock) || lock.Code != "era" {
		t.Errorf("era lock: %v", err)
	}
	if err := BuyUpgrade(&g, cfg, "bulk_racking"); err != nil {
		t.Fatal(err)
	}
	if err := BuyUpgrade(&g, cfg, "bulk_racking"); !errors.Is(err, ErrUpgradeOwned) {
		t.Errorf("owned: %v", err)
	}
	g.Status = StatusBankrupt
	if err := BuyUpgrade(&g, cfg, "order_book"); !errors.Is(err, ErrGameOver) {
		t.Errorf("game over: %v", err)
	}
}

func TestUpgradeRequirementsAreChecked(t *testing.T) {
	g, cfg := newTestGame()
	cfg.Upgrades = []UpgradeDef{
		{Key: "a", Cost: 1, Requires: content.Requires{Era: 1}},
		{Key: "b", Cost: 1, Requires: content.Requires{Upgrades: []string{"a"}}},
		{Key: "c", Cost: 1, Requires: content.Requires{WarehouseLevel: 2}},
		{Key: "d", Cost: 1, Requires: content.Requires{ProductionLevel: 2}},
	}
	code := func(key string) string {
		var lock *UpgradeLockedError
		if err := BuyUpgrade(&g, cfg, key); errors.As(err, &lock) {
			return lock.Code + ":" + lock.Need
		}
		return "ok"
	}
	if got := code("b"); got != "requires_upgrade:a" {
		t.Errorf("b: %s", got)
	}
	if got := code("c"); got != "warehouse_level:2" {
		t.Errorf("c: %s", got)
	}
	if got := code("d"); got != "production_level:2" {
		t.Errorf("d: %s", got)
	}
	if code("a") != "ok" || code("b") != "ok" {
		t.Error("a then b should be buyable")
	}
}

func TestUpgradesAreNotCountedInNetWorth(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 5000
	before := NetWorth(g, cfg)
	if err := BuyUpgrade(&g, cfg, "painted_stand"); err != nil {
		t.Fatal(err)
	}
	if got := NetWorth(g, cfg); got != before-1500 {
		t.Fatalf("net worth %d, want %d: an upgrade cannot be sold, so it is not counted", got, before-1500)
	}
}

func TestAGameWithoutAnUpgradesMapStillWorks(t *testing.T) {
	g, cfg := newTestGame()
	g.Upgrades, g.Carry = nil, nil // a save from before upgrades
	if g.Owns("freezer_1") || IceKeep(g, cfg) != 0 {
		t.Fatal("an old save owns nothing")
	}
	if err := BuyUpgrade(&g, cfg, "order_book"); err != nil || !g.Owns("order_book") {
		t.Fatalf("buy on an old save: %v", err)
	}
}

// Effects: each kind has one hook and a test.

func TestFreezerKeepsFreshIceForOneNight(t *testing.T) {
	g, cfg := newTestGame()
	rich(&g)
	own(&g, "freezer_1") // 20 cases
	g.WarehouseQty[StorageFrozen] = 5
	setInputs(&g, 10, 10, 35, 10) // production uses 10, leaving 25 fresh
	r, err := EndDay(&g, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if r.IceKept != 20 || r.IceMelted != 5 || g.Inventory[Ice] != 20 || g.IceOld != 20 {
		t.Fatalf("kept %d melted %d stock %d old %d, want 20/5/20/20", r.IceKept, r.IceMelted, g.Inventory[Ice], g.IceOld)
	}

	// Next night: production uses the old ice first, and old ice left over melts.
	setInputs(&g, 10, 10, 20, 10)
	g.Inventory[Lemonade] = 0
	g.IceOld = 20
	r, _ = EndDay(&g, cfg)
	if g.IceOld != 0 || r.IceKept != 0 || r.IceMelted != 10 {
		t.Fatalf("old %d kept %d melted %d: 10 old ice was used, 10 old ice left melts, and there was no fresh ice", g.IceOld, r.IceKept, r.IceMelted)
	}
}

func TestFreezerKeepsOnlyFreshIceAndSellsOldestFirst(t *testing.T) {
	g, cfg := newTestGame()
	own(&g, "freezer_1")
	g.WarehouseQty[StorageFrozen] = 5
	g.Inventory[Ice], g.IceOld = 30, 12
	g.CostBasis[Ice] = 300
	if err := Sell(&g, cfg, Ice, 5); err != nil {
		t.Fatal(err)
	}
	if g.IceOld != 7 {
		t.Fatalf("old ice %d after selling 5: the oldest sells first, want 7", g.IceOld)
	}
	if err := Sell(&g, cfg, Ice, 10); err != nil {
		t.Fatal(err)
	}
	if g.IceOld != 0 {
		t.Fatalf("old ice %d, want 0", g.IceOld)
	}
}

func TestFreezerBasisFollowsTheKeptIce(t *testing.T) {
	g, cfg := newTestGame()
	rich(&g)
	own(&g, "freezer_1")
	g.WarehouseQty[StorageFrozen] = 5
	g.ProductionQty = 0
	g.Inventory[Ice], g.CostBasis[Ice] = 30, 300
	if _, err := EndDay(&g, cfg); err != nil {
		t.Fatal(err)
	}
	if g.Inventory[Ice] != 20 || g.CostBasis[Ice] != 200 {
		t.Fatalf("stock %d basis %d, want 20 and $200", g.Inventory[Ice], g.CostBasis[Ice])
	}
}

func TestPreviewMatchesEndDayWithUpgrades(t *testing.T) {
	cfg := DefaultConfig()
	rng := rand.New(rand.NewSource(11))
	keys := []string{"freezer_1", "freezer_2", "citrus_press", "sugar_dissolver", "ice_machine", "bulk_racking", "accountant"}
	for i := 0; i < 500; i++ {
		g := NewGame(cfg, int64(i))
		g.ProductionLevel = 1 + rng.Intn(cfg.MaxLevel)
		g.ProductionQty = 1 + rng.Intn(cfg.MaxQuantity)
		g.WarehouseLevel = 1 + rng.Intn(cfg.MaxLevel)
		for _, k := range keys {
			if rng.Intn(2) == 0 {
				own(&g, k)
			}
		}
		for _, r := range Resources {
			g.WarehouseQty[ClassOf(cfg, r)] = 1 + rng.Intn(cfg.MaxQuantity)
			g.Inventory[r] = rng.Intn(Capacity(g, cfg, r) + 1)
		}
		g.IceOld = rng.Intn(g.Inventory[Ice] + 1)
		g.Carry["yield:lemonade"] = rng.Float64()
		g.Carry["save:sugar"] = rng.Float64()
		g.Capital = 1_000_000
		p := PreviewEndDay(g, cfg)
		report, err := EndDay(&g, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if p.LemonadeToProduce != report.Produced || p.IceToMelt != report.IceMelted || p.IceKept != report.IceKept {
			t.Fatalf("case %d: preview %+v, actual produced=%d melted=%d kept=%d", i, p, report.Produced, report.IceMelted, report.IceKept)
		}
	}
}

func TestForecastIsExactAndDoesNotDisturbTheWalk(t *testing.T) {
	cfg := DefaultConfig()
	sawEvent := false
	for seed := int64(1); seed <= 60; seed++ {
		g := NewGame(cfg, seed)
		rich(&g)
		own(&g, "farmers_almanac")
		twin := NewGame(cfg, seed) // never forecasts
		rich(&twin)
		for day := 0; day < 40; day++ {
			want := Forecast(g, cfg)
			var seen []ForecastEntry
			for d := 1; d <= 3; d++ {
				rep, err := EndDay(&g, cfg)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := EndDay(&twin, cfg); err != nil {
					t.Fatal(err)
				}
				for _, e := range rep.NewEvents {
					seen = append(seen, ForecastEntry{DaysAhead: d, Key: e.Key, Name: e.Name, Duration: e.DaysLeft})
				}
			}
			if !reflect.DeepEqual(want, seen) && (len(want) > 0 || len(seen) > 0) {
				t.Fatalf("seed %d day %d: forecast %+v, what happened %+v", seed, g.Day, want, seen)
			}
			sawEvent = sawEvent || len(seen) > 0
			for _, r := range cfg.Resources() {
				if g.Market[r].Price != twin.Market[r].Price {
					t.Fatalf("seed %d: owning a forecast changed the %s walk", seed, r)
				}
			}
		}
	}
	if !sawEvent {
		t.Fatal("test never saw an event")
	}
}

func TestWeatherRadioSeesOnlyTomorrowsWeather(t *testing.T) {
	cfg := DefaultConfig()
	for seed := int64(1); seed <= 200; seed++ {
		g := NewGame(cfg, seed)
		own(&g, "weather_radio")
		for _, f := range Forecast(g, cfg) {
			if f.DaysAhead != 1 {
				t.Fatalf("radio saw day %d", f.DaysAhead)
			}
			if f.Key != "heat_wave" && f.Key != "rainy_week" {
				t.Fatalf("radio saw a non-weather event %s", f.Key)
			}
		}
	}
	g := NewGame(cfg, 1)
	if len(Forecast(g, cfg)) != 0 {
		t.Fatal("no forecast without an upgrade")
	}
}

func TestDampingAndInsuranceChangeWhatYouSeeNotTheEvents(t *testing.T) {
	cfg := DefaultConfig()
	for seed := int64(1); seed <= 40; seed++ {
		plain, geared := NewGame(cfg, seed), NewGame(cfg, seed)
		rich(&plain)
		rich(&geared)
		own(&geared, "awnings", "insurance", "mascot", "orchard_lease")
		for d := 0; d < 80; d++ {
			a, _ := EndDay(&plain, cfg)
			b, _ := EndDay(&geared, cfg)
			if !reflect.DeepEqual(a.NewEvents, b.NewEvents) || len(plain.Events) != len(geared.Events) {
				t.Fatalf("seed %d day %d: upgrades changed the event history", seed, d)
			}
			for _, r := range cfg.Resources() {
				if plain.Market[r].Price != geared.Market[r].Price {
					t.Fatalf("seed %d: upgrades changed the walk of %s", seed, r)
				}
			}
		}
	}
}

func TestEventMultiplierScaling(t *testing.T) {
	g, cfg := newTestGame()
	rain := ActiveEvent{Key: "rainy_week", Multipliers: map[Resource]float64{Lemonade: 0.75}}
	holiday := ActiveEvent{Key: "holiday", Multipliers: map[Resource]float64{Lemonade: 1.35}}
	blight := ActiveEvent{Key: "lemon_blight", Multipliers: map[Resource]float64{Lemon: 1.7}}
	crash := ActiveEvent{Key: "x", Multipliers: map[Resource]float64{Lemonade: 0.5, Ice: 0.5}}
	get := func(e ActiveEvent, r Resource) float64 { m, _ := eventMultiplier(g, cfg, e, r); return m }
	if get(rain, Lemonade) != 0.75 {
		t.Fatal("no upgrades: unchanged")
	}
	own(&g, "awnings")
	if m := get(rain, Lemonade); m != 0.875 {
		t.Errorf("awnings: rainy week x%v, want 0.875", m)
	}
	own(&g, "mascot", "orchard_lease", "insurance")
	if m := get(holiday, Lemonade); m < 1.524 || m > 1.526 {
		t.Errorf("mascot: holiday x%v, want 1.525", m)
	}
	if m := get(blight, Lemon); m < 1.349 || m > 1.351 {
		t.Errorf("orchard lease: blight x%v, want 1.35", m)
	}
	if m := get(crash, Lemonade); m != 0.9 {
		t.Errorf("insurance floors a product at 0.9, got %v", m)
	}
	if m := get(crash, Ice); m != 0.5 {
		t.Errorf("insurance covers products only, ice got %v", m)
	}
}

func TestYieldBonusCarriesFractionsAsWholeCases(t *testing.T) {
	g, cfg := newTestGame()
	rich(&g)
	own(&g, "citrus_press") // +10%
	g.ProductionQty = 10
	g.WarehouseQty[StorageFinished] = 10
	total, batches := 0, 0
	for d := 0; d < 10; d++ {
		setInputs(&g, 3, 3, 3, 3) // 3 batches a day: 0.3 extra cases
		g.Inventory[Lemonade] = 0
		r, _ := EndDay(&g, cfg)
		total += r.Produced
		batches += 3
	}
	if total != batches+3 {
		t.Fatalf("made %d cases from %d batches, want %d: 10 days x 0.3 extra = 3", total, batches, batches+3)
	}
	if g.Carry["yield:lemonade"] > 1e-6 {
		t.Fatalf("remainder %v should be used up", g.Carry)
	}
}

func TestYieldBonusRespectsWarehouseSpace(t *testing.T) {
	g, cfg := newTestGame()
	rich(&g)
	own(&g, "citrus_press")
	g.Carry["yield:lemonade"] = 0.9
	setInputs(&g, 5, 5, 5, 5)
	g.Inventory[Lemonade] = 5 // 5 free of 10
	r, _ := EndDay(&g, cfg)
	if r.Produced > 5 {
		t.Fatalf("produced %d into 5 free spaces", r.Produced)
	}
}

func TestUseDiscountSavesWholeCases(t *testing.T) {
	g, cfg := newTestGame()
	rich(&g)
	own(&g, "sugar_dissolver") // -10%
	g.ProductionQty = 10
	for d := 0; d < 10; d++ {
		setInputs(&g, 10, 10, 10, 10)
		g.Inventory[Lemonade] = 0
		_, _ = EndDay(&g, cfg)
		if d == 9 {
			break
		}
	}
	// Ten batches of ten: sugar saved 1 case a day.
	if g.Inventory[Sugar] != 0+1 && g.Inventory[Sugar] != 1 {
		t.Fatalf("sugar left %d after the last day, want 1 saved", g.Inventory[Sugar])
	}
}

func TestDepthAndStorageAndUpkeepHooks(t *testing.T) {
	g, cfg := newTestGame()
	base := freeDepth(g, cfg, Lemonade)
	own(&g, "painted_stand")
	if freeDepth(g, cfg, Lemonade) != base+10 {
		t.Errorf("painted stand: depth %d, want %d", freeDepth(g, cfg, Lemonade), base+10)
	}
	lemonBase := freeDepth(g, cfg, Lemon)
	own(&g, "orchard_lease", "supplier_contract_2")
	if want := int(float64(lemonBase)*1.5 + 0.5); freeDepth(g, cfg, Lemon) != want {
		t.Errorf("lemon depth %d, want %d (+20%% and +30%%)", freeDepth(g, cfg, Lemon), want)
	}
	if freeDepth(g, cfg, Lemonade) != base+10 {
		t.Error("input depth upgrades must not change lemonade")
	}

	g2, _ := newTestGame()
	sugar, lemon := Capacity(g2, cfg, Sugar), Capacity(g2, cfg, Lemon)
	own(&g2, "bulk_racking")
	if Capacity(g2, cfg, Sugar) != sugar*115/100 || Capacity(g2, cfg, Lemon) != lemon {
		t.Errorf("bulk racking: sugar %d (was %d), lemon %d (was %d)", Capacity(g2, cfg, Sugar), sugar, Capacity(g2, cfg, Lemon), lemon)
	}

	g3, _ := newTestGame()
	g3.ProductionQty, g3.ProductionLevel, g3.WarehouseLevel = 10, 3, 3
	raw := ProductionUpkeep(g3, cfg)
	own(&g3, "automation_line")
	if got := ProductionUpkeep(g3, cfg); got != int(math.Round(float64(raw)*0.85)) {
		t.Errorf("automation line: production upkeep %d of %d", got, raw)
	}
	wh := WarehouseUpkeep(g3, cfg)
	own(&g3, "accountant")
	if got := WarehouseUpkeep(g3, cfg); got >= wh {
		t.Errorf("accountant should cut warehouse upkeep: %d vs %d", got, wh)
	}
}

func TestInputDiscountLowersInputAsksOnly(t *testing.T) {
	g, cfg := newTestGame()
	plain := Quotes(g, cfg)
	own(&g, "supplier_contract_1", "supplier_contract_2") // -6%
	q := Quotes(g, cfg)
	if q[Lemon].Ask >= plain[Lemon].Ask {
		t.Errorf("lemon ask %d, was %d", q[Lemon].Ask, plain[Lemon].Ask)
	}
	if q[Lemonade] != plain[Lemonade] || q[Lemon].Bid != plain[Lemon].Bid {
		t.Error("only input asks change")
	}
}

func TestIceMachineTopsUpIceAtItsPrice(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 10_000
	own(&g, "ice_machine")
	g.ProductionQty = 5
	for _, r := range Resources {
		g.WarehouseQty[ClassOf(cfg, r)] = 10
	}
	setInputs(&g, 50, 50, 0, 50)
	r, _ := EndDay(&g, cfg)
	if r.IceMade != 50 || r.IceMadeCost != 200 || r.Produced != 50 {
		t.Fatalf("made %d for $%d, produced %d; want 50 ice for $200 making 50 lemonade", r.IceMade, r.IceMadeCost, r.Produced)
	}
	// With nothing to make lemonade from, it makes no ice.
	g2, _ := newTestGame()
	own(&g2, "ice_machine")
	setInputs(&g2, 0, 10, 0, 10)
	if r, _ := EndDay(&g2, cfg); r.IceMade != 0 {
		t.Fatalf("made %d ice for a batch that cannot run", r.IceMade)
	}
}

func TestUnlocksFeaturesAndShelfLife(t *testing.T) {
	g, cfg := newTestGame()
	if HasUnlock(g, cfg, "bakery") || HasFeature(g, cfg, "pnl") || ExtraShelfLife(g, cfg, content.StorageCold) != 0 {
		t.Fatal("nothing without upgrades")
	}
	own(&g, "oven", "bookkeeper", "cold_room", "order_book")
	if !HasUnlock(g, cfg, "bakery") || !HasFeature(g, cfg, "pnl") || ExtraShelfLife(g, cfg, content.StorageCold) != 2 {
		t.Fatal("upgrades should unlock their targets")
	}
	if got := Features(g, cfg); !reflect.DeepEqual(got, []string{"pnl", "repeat_trades"}) {
		t.Fatalf("features %v", got)
	}
}

func TestBookkeeperReportsTheDaysProfitAndLoss(t *testing.T) {
	g, cfg := newTestGame()
	own(&g, "bookkeeper")
	_ = Buy(&g, cfg, Lemon, 4)
	_ = Buy(&g, cfg, Sugar, 4)
	_ = Buy(&g, cfg, Ice, 4)
	_ = Buy(&g, cfg, Cup, 4)
	g.Inventory[Lemonade] = 3
	_ = Sell(&g, cfg, Lemonade, 3)
	r, _ := EndDay(&g, cfg)
	if r.Pnl == nil {
		t.Fatal("no P&L")
	}
	want := g.Stats.Earned - g.Stats.Spent - r.UpkeepPaid
	if r.Pnl.Net != want || r.Pnl.Sales != g.Stats.Earned || r.Pnl.Purchases != g.Stats.Spent {
		t.Fatalf("pnl %+v, want net %d (earned %d spent %d upkeep %d)", *r.Pnl, want, g.Stats.Earned, g.Stats.Spent, r.UpkeepPaid)
	}
	g2, _ := newTestGame()
	if r, _ := EndDay(&g2, cfg); r.Pnl != nil {
		t.Fatal("no bookkeeper, no P&L")
	}
}

func TestUpgradeUpkeepIsPaidAndReported(t *testing.T) {
	g, cfg := newTestGame()
	rich(&g)
	own(&g, "freezer_1", "bookkeeper") // $5 + $5
	r, _ := EndDay(&g, cfg)
	if r.UpgradeUpkeep != 10 || r.UpkeepPaid != TotalUpkeep(g, cfg) {
		t.Fatalf("upgrade upkeep %d, paid %d, total %d", r.UpgradeUpkeep, r.UpkeepPaid, TotalUpkeep(g, cfg))
	}
}

func TestCloneCopiesUpgradeState(t *testing.T) {
	g, _ := newTestGame()
	own(&g, "freezer_1")
	g.Carry["x"] = 0.5
	c := g.Clone()
	c.Upgrades["bookkeeper"] = 1
	c.Carry["x"] = 0.9
	if g.Owns("bookkeeper") || g.Carry["x"] != 0.5 {
		t.Fatal("clone shares maps with the original")
	}
}
