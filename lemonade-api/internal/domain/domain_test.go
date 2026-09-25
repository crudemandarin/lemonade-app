package domain

import (
	"errors"
	"math"
	"math/rand"
	"reflect"
	"testing"
)

func newTestGame() (Game, Config) {
	cfg := DefaultConfig()
	return NewGame(cfg, 42), cfg
}

func TestNewGame(t *testing.T) {
	g, cfg := newTestGame()

	if g.Day != 1 || g.Capital != 1000 || g.Status != StatusActive {
		t.Fatalf("got day=%d capital=%d status=%s", g.Day, g.Capital, g.Status)
	}
	if g.WarehouseLevel != 1 || g.ProductionLevel != 1 || g.ProductionQty != 1 {
		t.Fatalf("facilities not at level 1 / quantity 1: %+v", g)
	}
	for _, r := range Resources {
		if g.Inventory[r] != 0 {
			t.Errorf("inventory[%s] = %d, want 0", r, g.Inventory[r])
		}
		if g.WarehouseQty[r] != 1 {
			t.Errorf("warehouseQty[%s] = %d, want 1", r, g.WarehouseQty[r])
		}
		if got := Capacity(g, cfg, r); got != 10 {
			t.Errorf("capacity[%s] = %d, want 10", r, got)
		}
		if q := Quotes(g, cfg)[r]; q.Price != cfg.BasePrice[r] {
			t.Errorf("price[%s] = %d, want %d", r, q.Price, cfg.BasePrice[r])
		}
	}
	if got := ProductionCapacity(g, cfg); got != 10 {
		t.Errorf("production capacity = %d, want 10", got)
	}
	if got := TotalUpkeep(g, cfg); got != 30 {
		t.Errorf("upkeep = %d, want 15", got)
	}
}

func TestCapacityAcrossLevelsAndQuantities(t *testing.T) {
	g, cfg := newTestGame()
	tests := []struct {
		level, qty, want int
	}{
		{1, 1, 10}, {1, 3, 30}, {2, 2, 50}, {3, 1, 60}, {4, 10, 1500},
	}
	for _, tt := range tests {
		g.WarehouseLevel = tt.level
		g.WarehouseQty[Lemon] = tt.qty
		if got := Capacity(g, cfg, Lemon); got != tt.want {
			t.Errorf("level %d qty %d: capacity = %d, want %d", tt.level, tt.qty, got, tt.want)
		}
	}
}

func TestQuoteRounding(t *testing.T) {
	tests := []struct {
		price, bid, ask int
	}{
		{1, 1, 2},
		{2, 1, 3},
		{10, 9, 11},
		{20, 18, 22},
		{100, 90, 110},
		{93, 83, 103},
	}
	for _, tt := range tests {
		q := quote(tt.price, 0.10)
		if q.Bid != tt.bid || q.Ask != tt.ask {
			t.Errorf("price %d: bid/ask = %d/%d, want %d/%d", tt.price, q.Bid, q.Ask, tt.bid, tt.ask)
		}
	}
	for p := 1; p <= 500; p++ {
		q := quote(p, 0.10)
		if q.Bid >= q.Ask || q.Bid < 1 {
			t.Fatalf("price %d: bid %d ask %d violates bid < ask, bid >= 1", p, q.Bid, q.Ask)
		}
	}
}

func TestBuy(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(g *Game)
		qty     int
		wantErr error
	}{
		{"success", func(g *Game) {}, 5, nil},
		{"zero qty", func(g *Game) {}, 0, ErrInvalidQuantity},
		{"negative qty", func(g *Game) {}, -1, ErrInvalidQuantity},
		{"insufficient funds", func(g *Game) { g.Capital = 10 }, 5, ErrInsufficientFunds},
		{"capacity exceeded", func(g *Game) {}, 11, ErrCapacityExceeded},
		{"capacity exceeded by existing stock", func(g *Game) { g.Inventory[Lemon] = 8 }, 3, ErrCapacityExceeded},
		{"game over", func(g *Game) { g.Status = StatusBankrupt }, 1, ErrGameOver},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, cfg := newTestGame()
			tt.setup(&g)
			before := g
			before.Inventory = copyInv(g.Inventory)

			err := Buy(&g, cfg, Lemon, tt.qty)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				if g.Capital != before.Capital || !reflect.DeepEqual(g.Inventory, before.Inventory) {
					t.Fatal("failed buy changed state")
				}
				return
			}
			ask := Quotes(g, cfg)[Lemon].Ask
			if g.Capital != 1000-ask*tt.qty || g.Inventory[Lemon] != tt.qty {
				t.Fatalf("capital=%d inventory=%d", g.Capital, g.Inventory[Lemon])
			}
		})
	}
}

func TestSell(t *testing.T) {
	tests := []struct {
		name    string
		stock   int
		qty     int
		status  Status
		wantErr error
	}{
		{"success", 5, 5, StatusActive, nil},
		{"partial", 5, 2, StatusActive, nil},
		{"insufficient stock", 1, 2, StatusActive, ErrInsufficientStock},
		{"zero qty", 5, 0, StatusActive, ErrInvalidQuantity},
		{"game over", 5, 1, StatusBankrupt, ErrGameOver},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, cfg := newTestGame()
			g.Inventory[Lemonade] = tt.stock
			g.Status = tt.status

			err := Sell(&g, cfg, Lemonade, tt.qty)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				if g.Capital != 1000 || g.Inventory[Lemonade] != tt.stock {
					t.Fatal("failed sell changed state")
				}
				return
			}
			bid := Quotes(g, cfg)[Lemonade].Bid
			if g.Capital != 1000+bid*tt.qty || g.Inventory[Lemonade] != tt.stock-tt.qty {
				t.Fatalf("capital=%d stock=%d", g.Capital, g.Inventory[Lemonade])
			}
		})
	}
}

func TestNoSameDayArbitrage(t *testing.T) {
	g, cfg := newTestGame()
	if err := Buy(&g, cfg, Lemon, 10); err != nil {
		t.Fatal(err)
	}
	if err := Sell(&g, cfg, Lemon, 10); err != nil {
		t.Fatal(err)
	}
	if g.Capital >= 1000 {
		t.Fatalf("buy then sell made money: capital = %d", g.Capital)
	}
}

func TestExpand(t *testing.T) {
	t.Run("warehouse success adds one building's capacity", func(t *testing.T) {
		g, cfg := newTestGame()
		if err := Expand(&g, cfg, Warehouse, Sugar); err != nil {
			t.Fatal(err)
		}
		if g.Capital != 900 || g.WarehouseQty[Sugar] != 2 || Capacity(g, cfg, Sugar) != 20 {
			t.Fatalf("capital=%d qty=%d cap=%d", g.Capital, g.WarehouseQty[Sugar], Capacity(g, cfg, Sugar))
		}
		if Capacity(g, cfg, Lemon) != 10 {
			t.Fatal("expanding sugar changed lemon capacity")
		}
	})
	t.Run("production success", func(t *testing.T) {
		g, cfg := newTestGame()
		if err := Expand(&g, cfg, Production, ""); err != nil {
			t.Fatal(err)
		}
		if g.Capital != 500 || g.ProductionQty != 2 || ProductionCapacity(g, cfg) != 20 {
			t.Fatalf("capital=%d qty=%d cap=%d", g.Capital, g.ProductionQty, ProductionCapacity(g, cfg))
		}
	})
	t.Run("insufficient funds", func(t *testing.T) {
		g, cfg := newTestGame()
		g.Capital = 99
		if err := Expand(&g, cfg, Warehouse, Lemon); !errors.Is(err, ErrInsufficientFunds) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("max quantity", func(t *testing.T) {
		g, cfg := newTestGame()
		g.WarehouseQty[Lemon] = cfg.MaxQuantity
		g.ProductionQty = cfg.MaxQuantity
		if err := Expand(&g, cfg, Warehouse, Lemon); !errors.Is(err, ErrMaxQuantity) {
			t.Fatalf("warehouse err = %v", err)
		}
		if err := Expand(&g, cfg, Production, ""); !errors.Is(err, ErrMaxQuantity) {
			t.Fatalf("production err = %v", err)
		}
	})
	t.Run("game over", func(t *testing.T) {
		g, cfg := newTestGame()
		g.Status = StatusBankrupt
		if err := Expand(&g, cfg, Production, ""); !errors.Is(err, ErrGameOver) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestUpgradeCostTable(t *testing.T) {
	// UX-MOCKS-AND-CHANGES 1.3: per-building upgrade cost to the next level.
	cfg := DefaultConfig()
	for name, tc := range map[string]struct {
		tiers []Tier
		want  []int
	}{
		"warehouse":  {cfg.WarehouseTiers, []int{155, 200, 400, 0}},
		"production": {cfg.ProductionTiers, []int{780, 1000, 2000, 0}},
	} {
		for i, want := range tc.want {
			if got := tc.tiers[i].UpgradeCost; got != want {
				t.Errorf("%s level %d upgrade cost = %d, want %d", name, i+1, got, want)
			}
		}
	}
}

func TestUpgradeFreshGameExample(t *testing.T) {
	// Spec example: a fresh game's 5 Pantries become 5 Garages for 5 x $155 = $775.
	g, cfg := newTestGame()
	if err := Upgrade(&g, cfg, Warehouse); err != nil {
		t.Fatal(err)
	}
	if g.Capital != 225 || g.WarehouseLevel != 2 || Capacity(g, cfg, Lemon) != 25 {
		t.Fatalf("capital=%d level=%d capacity=%d", g.Capital, g.WarehouseLevel, Capacity(g, cfg, Lemon))
	}
}

func TestUpgrade(t *testing.T) {
	t.Run("warehouse cost scales with total buildings", func(t *testing.T) {
		g, cfg := newTestGame()
		g.Capital = 5000
		g.WarehouseQty[Lemon] = 2 // 6 buildings total
		if err := Upgrade(&g, cfg, Warehouse); err != nil {
			t.Fatal(err)
		}
		if g.Capital != 5000-6*155 || g.WarehouseLevel != 2 {
			t.Fatalf("capital=%d level=%d", g.Capital, g.WarehouseLevel)
		}
		if Capacity(g, cfg, Lemon) != 50 || Capacity(g, cfg, Sugar) != 25 {
			t.Fatal("capacity did not follow the new level")
		}
	})
	t.Run("production cost scales with quantity", func(t *testing.T) {
		g, cfg := newTestGame()
		g.Capital = 5000
		g.ProductionQty = 2
		if err := Upgrade(&g, cfg, Production); err != nil {
			t.Fatal(err)
		}
		if g.Capital != 5000-2*780 || g.ProductionLevel != 2 || ProductionCapacity(g, cfg) != 50 {
			t.Fatalf("capital=%d level=%d cap=%d", g.Capital, g.ProductionLevel, ProductionCapacity(g, cfg))
		}
	})
	t.Run("insufficient funds", func(t *testing.T) {
		g, cfg := newTestGame()
		g.Capital = 779
		if err := Upgrade(&g, cfg, Production); !errors.Is(err, ErrInsufficientFunds) {
			t.Fatalf("err = %v", err)
		}
		if g.ProductionLevel != 1 {
			t.Fatal("failed upgrade changed level")
		}
	})
	t.Run("max level", func(t *testing.T) {
		g, cfg := newTestGame()
		g.WarehouseLevel = cfg.MaxLevel
		g.ProductionLevel = cfg.MaxLevel
		if err := Upgrade(&g, cfg, Warehouse); !errors.Is(err, ErrMaxLevel) {
			t.Fatalf("warehouse err = %v", err)
		}
		if err := Upgrade(&g, cfg, Production); !errors.Is(err, ErrMaxLevel) {
			t.Fatalf("production err = %v", err)
		}
	})
}

func TestUpkeepSum(t *testing.T) {
	g, cfg := newTestGame()
	g.WarehouseLevel = 2
	g.WarehouseQty[Lemon] = 3 // 7 buildings
	g.ProductionLevel = 3
	g.ProductionQty = 2
	want := 7*cfg.WarehouseTiers[1].Upkeep + 2*cfg.ProductionTiers[2].Upkeep
	if got := TotalUpkeep(g, cfg); got != want {
		t.Fatalf("upkeep = %d, want %d", got, want)
	}
}

func TestEndDayProduction(t *testing.T) {
	tests := []struct {
		name  string
		setup func(g *Game)
		want  int
	}{
		{"limited by inputs", func(g *Game) { setInputs(g, 4, 9, 9, 9) }, 4},
		{"limited by rate", func(g *Game) { setInputs(g, 10, 10, 10, 10); g.ProductionQty = 1 }, 10},
		{"limited by lemonade space", func(g *Game) { setInputs(g, 10, 10, 10, 10); g.Inventory[Lemonade] = 7 }, 3},
		{"no inputs", func(g *Game) {}, 0},
		{"limited by scarcest input", func(g *Game) { setInputs(g, 10, 10, 10, 0) }, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, cfg := newTestGame()
			tt.setup(&g)
			lemonadeBefore := g.Inventory[Lemonade]
			report, err := EndDay(&g, cfg)
			if err != nil {
				t.Fatal(err)
			}
			if report.Produced != tt.want || g.Inventory[Lemonade] != lemonadeBefore+tt.want {
				t.Fatalf("produced=%d lemonade=%d, want %d", report.Produced, g.Inventory[Lemonade], tt.want)
			}
		})
	}
}

func TestEndDayUsesUpgradedProduction(t *testing.T) {
	g, cfg := newTestGame()
	g.ProductionLevel = 2
	g.WarehouseLevel = 2
	setInputs(&g, 20, 20, 20, 20)
	report, _ := EndDay(&g, cfg)
	if report.Produced != 20 {
		t.Fatalf("produced = %d, want 20", report.Produced)
	}
}

func TestEndDayMeltsIceAndKeepsOthers(t *testing.T) {
	g, cfg := newTestGame()
	setInputs(&g, 5, 5, 5, 5)
	g.ProductionQty = 0 // isolate melt from production
	report, _ := EndDay(&g, cfg)

	if report.IceMelted != 5 || g.Inventory[Ice] != 0 {
		t.Fatalf("iceMelted=%d ice=%d", report.IceMelted, g.Inventory[Ice])
	}
	if g.Inventory[Lemon] != 5 || g.Inventory[Sugar] != 5 || g.Inventory[Cup] != 5 {
		t.Fatalf("other resources did not carry forward: %v", g.Inventory)
	}
}

func TestEndDayUpkeepSettlement(t *testing.T) {
	t.Run("paid from cash, nothing sold", func(t *testing.T) {
		g, cfg := newTestGame()
		due := TotalUpkeep(g, cfg)
		report, _ := EndDay(&g, cfg)
		if report.UpkeepPaid != due || g.Capital != 1000-due || report.CapitalBefore != 1000 || report.CapitalAfter != 1000-due {
			t.Fatalf("report = %+v capital = %d", report, g.Capital)
		}
		if report.ForcedSaleCases != 0 || report.ForcedSaleProceeds != 0 {
			t.Fatalf("sold stock without needing to: %+v", report)
		}
	})

	t.Run("short on cash: stock is sold at bid to cover it", func(t *testing.T) {
		g, cfg := newTestGame()
		due := TotalUpkeep(g, cfg)
		bid := Quotes(g, cfg)[Lemonade].Bid
		g.Capital = 4
		g.Inventory[Lemonade] = 3
		report, _ := EndDay(&g, cfg)

		wantCases := (due - 4 + bid - 1) / bid // just enough lemonade to cover the shortfall
		if report.ForcedSaleCases != wantCases || report.ForcedSaleProceeds != wantCases*bid {
			t.Fatalf("forced sale = %d cases for $%d, want %d cases for $%d", report.ForcedSaleCases, report.ForcedSaleProceeds, wantCases, wantCases*bid)
		}
		if report.UpkeepPaid != due || g.Capital != 4+wantCases*bid-due || g.Inventory[Lemonade] != 3-wantCases {
			t.Fatalf("paid=%d capital=%d lemonade=%d", report.UpkeepPaid, g.Capital, g.Inventory[Lemonade])
		}
		if g.Status != StatusActive {
			t.Fatal("solvent player was declared bankrupt")
		}
	})

	t.Run("sells the finished product before raw inputs", func(t *testing.T) {
		g, cfg := newTestGame()
		g.Capital = 0
		g.Inventory[Lemonade], g.Inventory[Lemon] = 5, 5 // a lone input never produces
		EndDay(&g, cfg)
		if g.Inventory[Lemon] != 5 || g.Inventory[Lemonade] == 5 {
			t.Fatalf("expected lemonade sold first: lemon=%d lemonade=%d", g.Inventory[Lemon], g.Inventory[Lemonade])
		}
	})

	t.Run("cannot cover it: everything is sold and the game ends", func(t *testing.T) {
		g, cfg := newTestGame()
		g.Capital = 0
		g.Inventory[Sugar] = 1 // worth far less than a day's upkeep
		report, _ := EndDay(&g, cfg)
		if g.Status != StatusBankrupt || !report.Bankrupt {
			t.Fatal("expected bankrupt")
		}
		if g.Capital != 0 || g.Inventory[Sugar] != 0 || report.UpkeepPaid >= TotalUpkeep(g, cfg) {
			t.Fatalf("capital=%d sugar=%d paid=%d", g.Capital, g.Inventory[Sugar], report.UpkeepPaid)
		}
	})
}

func TestEndDayAdvancesDay(t *testing.T) {
	g, cfg := newTestGame()
	report, _ := EndDay(&g, cfg)
	if report.Day != 1 || g.Day != 2 {
		t.Fatalf("report.Day=%d g.Day=%d", report.Day, g.Day)
	}
}

// Whether a player survives the end of day depends on cash plus what their stock
// sells for at bid, measured against one day's upkeep.
func TestBankruptcyOutcomes(t *testing.T) {
	g0, cfg := newTestGame()
	due := TotalUpkeep(g0, cfg)
	q := Quotes(g0, cfg)
	lemonBid, lemonadeBid := q[Lemon].Bid, q[Lemonade].Bid

	tests := []struct {
		name      string
		capital   int
		inventory map[Resource]int
		bankrupt  bool
	}{
		{"no cash, no stock", 0, nil, true},
		{"one dollar short, no stock", due - 1, nil, true},
		{"exactly enough cash", due, nil, false},
		{"no cash but one lemonade covers it", 0, map[Resource]int{Lemonade: 1}, lemonadeBid < due},
		{"no cash, lemons worth less than upkeep", 0, map[Resource]int{Lemon: (due - 1) / lemonBid}, true},
		{"no cash, lemons worth more than upkeep", 0, map[Resource]int{Lemon: due/lemonBid + 1}, false},
		{"cash plus stock together cover it", due - lemonBid, map[Resource]int{Lemon: 1}, false},
		{"ice does not count: it melts first", 0, map[Resource]int{Ice: 10}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, cfg := newTestGame()
			g.Capital = tt.capital
			for r, n := range tt.inventory {
				g.Inventory[r] = n
			}
			report, _ := EndDay(&g, cfg)
			if got := g.Status == StatusBankrupt; got != tt.bankrupt || report.Bankrupt != tt.bankrupt {
				t.Fatalf("bankrupt = %v (report %v), want %v", got, report.Bankrupt, tt.bankrupt)
			}
		})
	}
}

func TestEndDayBankruptcy(t *testing.T) {
	t.Run("bankruptcy blocks every later action", func(t *testing.T) {
		g, cfg := newTestGame()
		g.Capital = 0
		EndDay(&g, cfg)
		if g.Status != StatusBankrupt {
			t.Fatal("expected bankrupt")
		}
		if err := Buy(&g, cfg, Lemon, 1); !errors.Is(err, ErrGameOver) {
			t.Fatalf("buy after game over: %v", err)
		}
		if _, err := EndDay(&g, cfg); !errors.Is(err, ErrGameOver) {
			t.Fatalf("end day after game over: %v", err)
		}
	})
	t.Run("spending to zero mid-day is allowed", func(t *testing.T) {
		g, cfg := newTestGame()
		g.Capital = Quotes(g, cfg)[Lemon].Ask * 10
		if err := Buy(&g, cfg, Lemon, 10); err != nil {
			t.Fatal(err)
		}
		if g.Capital != 0 || g.Status != StatusActive {
			t.Fatalf("capital=%d status=%s", g.Capital, g.Status)
		}
	})
	t.Run("a full batch bought with the last dollar is not lost overnight", func(t *testing.T) {
		g, cfg := newTestGame()
		g.Capital = 0
		setInputs(&g, 10, 10, 10, 10) // produces 10 lemonade tonight
		EndDay(&g, cfg)
		if g.Status != StatusActive {
			t.Fatal("expected active: tonight's lemonade covers upkeep")
		}
		if g.Inventory[Lemonade] == 0 {
			t.Fatal("expected leftover lemonade after covering upkeep")
		}
	})
}

func TestDeterminism(t *testing.T) {
	run := func() Game {
		g, cfg := newTestGame()
		for i := 0; i < 50; i++ {
			g.Capital = 1000
			g.Inventory[Lemonade] = 1 // keep the game alive
			if _, err := EndDay(&g, cfg); err != nil {
				t.Fatal(err)
			}
		}
		return g
	}
	a, b := run(), run()
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same seed and actions produced different games")
	}
}

func TestDifferentSeedsDiverge(t *testing.T) {
	cfg := DefaultConfig()
	a, b := NewGame(cfg, 1), NewGame(cfg, 2)
	for i := 0; i < 20; i++ {
		a.Capital, b.Capital = 1000, 1000
		EndDay(&a, cfg)
		EndDay(&b, cfg)
	}
	if reflect.DeepEqual(a.Market, b.Market) {
		t.Fatal("different seeds gave identical markets")
	}
}

func TestPriceClampAndMeanReversion(t *testing.T) {
	g, cfg := newTestGame()
	cfg.EventChance = 0
	sums := map[Resource]float64{}
	const days = 1000
	for i := 0; i < days; i++ {
		g.Capital = 1000
		g.Inventory[Lemonade] = 1
		if _, err := EndDay(&g, cfg); err != nil {
			t.Fatal(err)
		}
		for _, r := range Resources {
			base := float64(cfg.BasePrice[r])
			p := g.Market[r].Price
			if p < 0.25*base-1e-9 || p > 4*base+1e-9 {
				t.Fatalf("day %d: %s price %.2f outside [%.2f, %.2f]", g.Day, r, p, 0.25*base, 4*base)
			}
			sums[r] += p
		}
	}
	for _, r := range Resources {
		avg := sums[r] / days
		base := float64(cfg.BasePrice[r])
		if avg < 0.8*base || avg > 1.25*base {
			t.Errorf("%s average %.2f is not near base %.0f", r, avg, base)
		}
	}
}

func TestHistoryCappedAndPreviousPrice(t *testing.T) {
	g, cfg := newTestGame()
	if g.Market[Lemon].PreviousEffective != nil {
		t.Fatal("previous price should be nil on day 1")
	}
	for i := 0; i < 30; i++ {
		g.Capital = 1000
		g.Inventory[Lemonade] = 1
		EndDay(&g, cfg)
	}
	m := g.Market[Lemon]
	if len(m.History) != cfg.HistoryLength {
		t.Fatalf("history length = %d, want %d", len(m.History), cfg.HistoryLength)
	}
	if m.PreviousEffective == nil || *m.PreviousEffective != m.History[len(m.History)-2] {
		t.Fatal("previous price should equal the second-newest history entry")
	}
}

func TestQuotesAlwaysWholeDollarsMinOne(t *testing.T) {
	g, cfg := newTestGame()
	for _, r := range Resources {
		g.Market[r].Price = 0.01
	}
	for r, q := range Quotes(g, cfg) {
		if q.Price < 1 || q.Bid < 1 || q.Ask <= q.Bid {
			t.Errorf("%s quote %+v violates min $1 / bid < ask", r, q)
		}
	}
}

func TestEventsStackAndOnlyAffectTargets(t *testing.T) {
	g, cfg := newTestGame()
	g.Events = []ActiveEvent{
		{Key: "a", Multipliers: map[Resource]float64{Lemonade: 1.4, Ice: 1.3}, DaysLeft: 2},
		{Key: "b", Multipliers: map[Resource]float64{Lemonade: 0.5}, DaysLeft: 1},
	}
	q := Quotes(g, cfg)
	base := float64(cfg.BasePrice[Lemonade])
	if want := int(math.Round(base * 1.4 * 0.5)); q[Lemonade].Price != want {
		t.Errorf("lemonade price = %d, want %d (base x1.4 x0.5)", q[Lemonade].Price, want)
	}
	if q[Ice].Price != 13 {
		t.Errorf("ice price = %d, want 13", q[Ice].Price)
	}
	if q[Lemon].Price != 20 || q[Sugar].Price != 10 {
		t.Error("untargeted resources changed")
	}
	if g.Market[Lemonade].Price != base {
		t.Error("events must not change the walked price")
	}
}

func TestEventSpawnAndExpiry(t *testing.T) {
	g, cfg := newTestGame()
	cfg.EventChance = 1
	cfg.Events = []EventDef{
		{Key: "only", Name: "Only", Duration: 2, Multipliers: map[Resource]float64{Lemon: 2}},
	}
	g.Inventory[Lemonade] = 1

	report, _ := EndDay(&g, cfg)
	if len(report.NewEvents) != 1 || len(g.Events) != 1 || g.Events[0].DaysLeft != 2 {
		t.Fatalf("day 1 end: new=%v active=%v", report.NewEvents, g.Events)
	}

	// Still active, so it is not re-spawned; countdown drops to 1.
	report, _ = EndDay(&g, cfg)
	if len(report.NewEvents) != 0 || len(g.Events) != 1 || g.Events[0].DaysLeft != 1 {
		t.Fatalf("day 2 end: new=%v active=%v", report.NewEvents, g.Events)
	}

	// Expires now; the same tick may then spawn it again since it is no longer active.
	report, _ = EndDay(&g, cfg)
	if len(report.ExpiredEvents) != 1 || report.ExpiredEvents[0].Key != "only" {
		t.Fatalf("expired = %v", report.ExpiredEvents)
	}
}

func TestEventExpiryWithoutSpawn(t *testing.T) {
	g, cfg := newTestGame()
	cfg.EventChance = 0
	g.Events = []ActiveEvent{{Key: "x", Multipliers: map[Resource]float64{Lemon: 2}, DaysLeft: 1}}
	g.Inventory[Lemonade] = 1

	report, _ := EndDay(&g, cfg)
	if len(g.Events) != 0 || len(report.ExpiredEvents) != 1 {
		t.Fatalf("active=%v expired=%v", g.Events, report.ExpiredEvents)
	}
}

func eventKeys(events []ActiveEvent) map[string]bool {
	keys := make(map[string]bool, len(events))
	for _, e := range events {
		keys[e.Key] = true
	}
	return keys
}

func TestConflictingEventsAreNotEligible(t *testing.T) {
	defs := []EventDef{
		{Key: "heat", Excludes: []string{"rain"}},
		{Key: "rain"},
		{Key: "other"},
	}
	keys := func(defs []EventDef) []string {
		out := []string{}
		for _, d := range defs {
			out = append(out, d.Key)
		}
		return out
	}

	// The exclusion is declared on "heat" only, but must work in both directions.
	if got := keys(eligibleEvents(defs, []ActiveEvent{{Key: "heat"}})); !reflect.DeepEqual(got, []string{"other"}) {
		t.Errorf("heat active: eligible = %v, want [other]", got)
	}
	if got := keys(eligibleEvents(defs, []ActiveEvent{{Key: "rain"}})); !reflect.DeepEqual(got, []string{"other"}) {
		t.Errorf("rain active: eligible = %v, want [other]", got)
	}
	if got := keys(eligibleEvents(defs, []ActiveEvent{{Key: "other"}})); !reflect.DeepEqual(got, []string{"heat", "rain"}) {
		t.Errorf("other active: eligible = %v, want [heat rain]", got)
	}
	if got := keys(eligibleEvents(defs, nil)); len(got) != 3 {
		t.Errorf("nothing active: eligible = %v, want all three", got)
	}
}

func TestDefaultConfigHeatWaveAndRainyWeekConflict(t *testing.T) {
	cfg := DefaultConfig()
	for _, active := range []string{"heat_wave", "rainy_week"} {
		other := map[string]string{"heat_wave": "rainy_week", "rainy_week": "heat_wave"}[active]
		for _, d := range eligibleEvents(cfg.Events, []ActiveEvent{{Key: active}}) {
			if d.Key == other {
				t.Errorf("%s is eligible while %s is active", other, active)
			}
		}
	}
}

func TestHeatWaveAndRainyWeekNeverOverlap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EventChance = 1 // an event every day, to stress the rule
	for seed := int64(1); seed <= 20; seed++ {
		g := NewGame(cfg, seed)
		rng := rand.New(rand.NewSource(seed))
		sawHeat, sawRain := false, false
		for day := 0; day < 300; day++ {
			tickEvents(&g, rng, cfg)
			active := eventKeys(g.Events)
			if active["heat_wave"] && active["rainy_week"] {
				t.Fatalf("seed %d day %d: heat wave and rainy week active together: %v", seed, day, g.Events)
			}
			sawHeat = sawHeat || active["heat_wave"]
			sawRain = sawRain || active["rainy_week"]
		}
		if !sawHeat || !sawRain {
			t.Fatalf("seed %d: simulation never saw both events (heat=%v rain=%v), so the check proves nothing", seed, sawHeat, sawRain)
		}
	}
}

func setInputs(g *Game, lemon, sugar, ice, cup int) {
	g.Inventory[Lemon], g.Inventory[Sugar], g.Inventory[Ice], g.Inventory[Cup] = lemon, sugar, ice, cup
}

func copyInv(m map[Resource]int) map[Resource]int {
	out := make(map[Resource]int, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// The game ends on the day it was lost: nothing after upkeep runs.
func TestBankruptEndDayDoesNotTick(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 0
	g.Events = []ActiveEvent{{Key: "holiday", Name: "Holiday", Multipliers: map[Resource]float64{Lemonade: 1.35}, DaysLeft: 1}}
	before := g.Clone()

	report, err := EndDay(&g, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Bankrupt || g.Status != StatusBankrupt {
		t.Fatal("expected bankrupt")
	}
	if g.Day != before.Day || report.Day != before.Day {
		t.Fatalf("day advanced: g.Day=%d report.Day=%d, want %d", g.Day, report.Day, before.Day)
	}
	if len(g.Events) != 1 || g.Events[0].DaysLeft != 1 || len(report.NewEvents)+len(report.ExpiredEvents) != 0 {
		t.Fatalf("events ticked: %+v", g.Events)
	}
	if len(report.PriceChanges) != 0 {
		t.Fatalf("prices changed: %+v", report.PriceChanges)
	}
	for _, r := range Resources {
		if g.Market[r].Price != before.Market[r].Price || len(g.Market[r].History) != len(before.Market[r].History) {
			t.Fatalf("%s market ticked", r)
		}
	}
	if report.CapitalAfter != g.Capital {
		t.Fatalf("CapitalAfter = %d, want %d", report.CapitalAfter, g.Capital)
	}
}

func TestUnknownFacilityTypeIsRejected(t *testing.T) {
	g, cfg := newTestGame()
	before := g.Clone()
	if err := Expand(&g, cfg, FacilityType("garage"), ""); !errors.Is(err, ErrInvalidFacility) {
		t.Fatalf("Expand err = %v, want ErrInvalidFacility", err)
	}
	if err := Upgrade(&g, cfg, FacilityType("garage")); !errors.Is(err, ErrInvalidFacility) {
		t.Fatalf("Upgrade err = %v, want ErrInvalidFacility", err)
	}
	if g.Capital != before.Capital {
		t.Fatal("a rejected action changed capital")
	}
}
