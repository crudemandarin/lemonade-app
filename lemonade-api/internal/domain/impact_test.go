package domain

import (
	"errors"
	"math/rand"
	"testing"
)

// impactCfg is DefaultConfig with a shallow market, so impact is easy to reach in tests.
func impactCfg(free int) Config {
	cfg := DefaultConfig()
	for _, r := range Resources {
		cfg.FreeDepth[r] = free
	}
	cfg.DepthByLevel = []float64{1, 1, 1, 1} // depth does not grow with the warehouse here
	return cfg
}

func TestTradesWithinFreeDepthCostExactlyTheQuote(t *testing.T) {
	g, cfg := newTestGame()
	q := Quotes(g, cfg)[Lemon]
	free := cfg.FreeDepth[Lemon]
	g.Capital = 1_000_000
	g.WarehouseQty[StorageCold] = 10
	g.WarehouseLevel = 4 // room for 800

	if err := Buy(&g, cfg, Lemon, free); err != nil {
		t.Fatal(err)
	}
	if spent := 1_000_000 - g.Capital; spent != free*q.Ask {
		t.Fatalf("bought %d for $%d, want exactly %d x $%d", free, spent, free, q.Ask)
	}
	before := g.Capital
	if err := Sell(&g, cfg, Lemon, free); err != nil {
		t.Fatal(err)
	}
	if got := g.Capital - before; got != free*q.Bid {
		t.Fatalf("sold %d for $%d, want exactly %d x $%d", free, got, free, q.Bid)
	}
}

func TestBuyingBeyondTheFreeDepthRaisesTheAsk(t *testing.T) {
	cfg := impactCfg(10)
	g := NewGame(cfg, 1)
	g.Capital = 1_000_000
	g.WarehouseLevel = 4
	g.WarehouseQty[StorageCold] = 10
	ask := Quotes(g, cfg)[Lemon].Ask

	if got := MarginalAsk(g, cfg, Lemon); got != ask {
		t.Fatalf("marginal ask before any trade = %d, want %d", got, ask)
	}
	if err := Buy(&g, cfg, Lemon, 30); err != nil {
		t.Fatal(err)
	}
	spent := 1_000_000 - g.Capital
	if spent <= 30*ask {
		t.Fatalf("30 cases cost $%d, want more than the plain $%d", spent, 30*ask)
	}
	// 10 are free; the 11th costs 1.5% more, the 30th 30% more (slope 0.015 per case beyond free).
	if got := MarginalAsk(g, cfg, Lemon); got <= ask {
		t.Fatalf("marginal ask after buying = %d, want above %d", got, ask)
	}
	if g.BuyPressure[Lemon] != 30 {
		t.Fatalf("buy pressure = %v", g.BuyPressure[Lemon])
	}
}

func TestBuyingRaisesTheAskButNotTheBid(t *testing.T) {
	cfg := impactCfg(5)
	g := NewGame(cfg, 1)
	g.Capital = 1_000_000
	g.WarehouseLevel = 4
	g.WarehouseQty[StorageDry] = 10
	q := Quotes(g, cfg)[Sugar]
	Buy(&g, cfg, Sugar, 40)
	if MarginalBid(g, cfg, Sugar) != q.Bid {
		t.Fatalf("buying moved the bid: %d, want %d", MarginalBid(g, cfg, Sugar), q.Bid)
	}
	Sell(&g, cfg, Sugar, 40)
	if MarginalAsk(g, cfg, Sugar) < q.Ask {
		t.Fatal("selling should not lower the ask below the plain quote")
	}
}

func TestSellingBeyondTheFreeDepthLowersTheBid(t *testing.T) {
	cfg := impactCfg(10)
	g := NewGame(cfg, 1)
	g.WarehouseLevel = 4
	g.WarehouseQty[StorageFinished] = 10
	g.Inventory[Lemonade] = 50
	bid := Quotes(g, cfg)[Lemonade].Bid
	before := g.Capital
	if err := Sell(&g, cfg, Lemonade, 50); err != nil {
		t.Fatal(err)
	}
	if got := g.Capital - before; got >= 50*bid {
		t.Fatalf("50 cases raised $%d, want less than the plain $%d", got, 50*bid)
	}
	if MarginalBid(g, cfg, Lemonade) >= bid {
		t.Fatal("the bid should have dropped")
	}
}

func TestImpactIsCapped(t *testing.T) {
	cfg := impactCfg(1)
	cfg.ImpactCap = 0.6
	g := NewGame(cfg, 1)
	g.BuyPressure[Lemon], g.SellPressure[Lemon] = 100_000, 100_000
	q := Quotes(g, cfg)[Lemon]
	if got, max := MarginalAsk(g, cfg, Lemon), int(float64(q.Ask)*1.6)+1; got > max {
		t.Fatalf("ask %d exceeds the +60%% cap (%d)", got, max)
	}
	if got, min := MarginalBid(g, cfg, Lemon), int(float64(q.Bid)*0.4); got < min {
		t.Fatalf("bid %d is below the -60%% cap (%d)", got, min)
	}
	if MarginalBid(g, cfg, Lemon) < 1 {
		t.Fatal("bid must stay at least $1")
	}
}

func TestPressureRecoversOvernight(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Recovery = 0.5
	g := NewGame(cfg, 1)
	g.BuyPressure[Lemon], g.SellPressure[Lemonade] = 80, 20
	if _, err := EndDay(&g, cfg); err != nil {
		t.Fatal(err)
	}
	if g.BuyPressure[Lemon] != 40 || g.SellPressure[Lemonade] != 10 {
		t.Fatalf("pressure after a night: %v / %v, want 40 / 10", g.BuyPressure[Lemon], g.SellPressure[Lemonade])
	}
	for i := 0; i < 40; i++ {
		EndDay(&g, cfg)
	}
	if g.BuyPressure[Lemon] != 0 {
		t.Fatalf("pressure never settles to zero: %v", g.BuyPressure[Lemon])
	}
}

func TestBulkClampsToImpactAwareAffordability(t *testing.T) {
	cfg := impactCfg(5)
	g := NewGame(cfg, 1)
	g.WarehouseLevel = 4
	g.WarehouseQty[StorageCold] = 10
	g.Capital = 400
	plain := Quotes(g, cfg)[Lemon].Ask

	if err := BuyClamped(&g, cfg, Lemon, 1_000_000); err != nil {
		t.Fatal(err)
	}
	if g.Capital < 0 {
		t.Fatalf("overspent: capital %d", g.Capital)
	}
	if bought := g.Inventory[Lemon]; bought >= 400/plain {
		t.Fatalf("bought %d, but with price impact fewer than the plain %d fit in $400", bought, 400/plain)
	}
	// One more case would not have fit.
	if MarginalAsk(g, cfg, Lemon) <= g.Capital {
		t.Fatalf("stopped early: $%d left and the next case costs $%d", g.Capital, MarginalAsk(g, cfg, Lemon))
	}
}

func TestQuotesReportAverageAndSlippage(t *testing.T) {
	cfg := impactCfg(10)
	g := NewGame(cfg, 1)
	g.Capital = 1_000_000
	g.WarehouseLevel = 4
	g.WarehouseQty[StorageCold] = 10

	within := QuoteBuy(g, cfg, Lemon, 10, false)
	if within.Total != within.PlainTotal || within.Slippage() != 0 {
		t.Fatalf("inside free depth: %+v", within)
	}
	deep := QuoteBuy(g, cfg, Lemon, 50, false)
	if deep.Total <= deep.PlainTotal || deep.Slippage() <= 0 || deep.Average() <= float64(Quotes(g, cfg)[Lemon].Ask) {
		t.Fatalf("beyond free depth: %+v", deep)
	}
	// A quote does not change the game.
	if g.Capital != 1_000_000 || g.BuyPressure[Lemon] != 0 {
		t.Fatal("QuoteBuy mutated the game")
	}
	// The quote matches what a purchase actually costs.
	if err := Buy(&g, cfg, Lemon, 50); err != nil {
		t.Fatal(err)
	}
	if spent := 1_000_000 - g.Capital; spent != deep.Total {
		t.Fatalf("quoted $%d, paid $%d", deep.Total, spent)
	}
}

// Property tests over random trades.
func TestImpactInvariants(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	for i := 0; i < 400; i++ {
		cfg := impactCfg(5 + rng.Intn(40))
		g := NewGame(cfg, int64(i))
		g.Capital = 10_000_000
		g.WarehouseLevel = 1 + rng.Intn(4)
		r := Resources[rng.Intn(len(Resources))]
		g.WarehouseQty[ClassOf(cfg, r)] = 10
		g.BuyPressure[r] = float64(rng.Intn(60))
		g.SellPressure[r] = float64(rng.Intn(60))
		n := 1 + rng.Intn(Capacity(g, cfg, r))

		plainAsk, plainBid := Quotes(g, cfg)[r].Ask, Quotes(g, cfg)[r].Bid
		before := g.Capital
		if err := Buy(&g, cfg, r, n); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		cost := before - g.Capital
		if cost < n*plainAsk {
			t.Fatalf("case %d: cost with impact $%d is below the plain $%d", i, cost, n*plainAsk)
		}
		mid := g.Capital
		if err := Sell(&g, cfg, r, n); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		proceeds := g.Capital - mid
		if proceeds > n*plainBid {
			t.Fatalf("case %d: proceeds $%d exceed the plain bid value $%d", i, proceeds, n*plainBid)
		}
		if g.Capital > before {
			t.Fatalf("case %d: buying %d and selling %d back profited ($%d -> $%d)", i, n, n, before, g.Capital)
		}
		if g.BuyPressure[r] < 0 || g.SellPressure[r] < 0 {
			t.Fatalf("case %d: negative pressure", i)
		}
	}
}

func TestImpactIsDeterministic(t *testing.T) {
	run := func() int {
		cfg := impactCfg(8)
		g := NewGame(cfg, 5)
		g.Capital = 1_000_000
		g.WarehouseLevel = 3
		for _, r := range Resources {
			g.WarehouseQty[ClassOf(cfg, r)] = 10
		}
		for d := 0; d < 20; d++ {
			for _, in := range Inputs {
				BuyClamped(&g, cfg, in, 40)
			}
			EndDay(&g, cfg)
			SellClamped(&g, cfg, Lemonade, 1000)
		}
		return g.Capital
	}
	if a, b := run(), run(); a != b {
		t.Fatalf("two identical games diverged: $%d vs $%d", a, b)
	}
}

func TestForcedSalesUseImpactAwareBids(t *testing.T) {
	cfg := impactCfg(1)
	cfg.ImpactShape = 0.1 // 10% a case at a depth of 1
	g := NewGame(cfg, 1)
	g.WarehouseLevel = 4
	g.WarehouseQty[StorageFinished] = 10
	g.ProductionQty = 10 // enough upkeep that the sale runs past the free depth of 1
	g.Inventory[Lemonade] = 200
	g.CostBasis[Lemonade] = 200 * 50
	g.Capital = 0
	due := TotalUpkeep(g, cfg)
	plainBid := Quotes(g, cfg)[Lemonade].Bid

	_, sold, proceeds, insolvent := settleUpkeep(&g, cfg)
	if insolvent {
		t.Fatal("200 lemonade should cover the upkeep")
	}
	if proceeds < due {
		t.Fatalf("raised $%d, need $%d", proceeds, due)
	}
	// With impact the same cash takes more cases than at the plain bid.
	if plain := (due + plainBid - 1) / plainBid; sold <= plain {
		t.Fatalf("sold %d cases, want more than the %d it would take at the plain bid", sold, plain)
	}
}

func TestBuyErrorsStayTheSame(t *testing.T) {
	cfg := impactCfg(2)
	g := NewGame(cfg, 1)
	g.Capital = 10
	if err := Buy(&g, cfg, Lemon, 5); !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("got %v", err)
	}
	g.Capital = 100_000
	if err := Buy(&g, cfg, Lemon, 11); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("got %v", err)
	}
}

// A game saved before market depth existed has no pressure maps at all.
func TestOldSavesWithoutPressureMapsStillTrade(t *testing.T) {
	g, cfg := newTestGame()
	g.BuyPressure, g.SellPressure = nil, nil
	if err := Buy(&g, cfg, Lemon, 3); err != nil {
		t.Fatal(err)
	}
	if err := Sell(&g, cfg, Lemon, 1); err != nil {
		t.Fatal(err)
	}
	if g.BuyPressure[Lemon] != 3 || g.SellPressure[Lemon] != 1 {
		t.Fatalf("pressure %v / %v", g.BuyPressure, g.SellPressure)
	}
	EndDay(&g, cfg) // forgetting must cope too
}
