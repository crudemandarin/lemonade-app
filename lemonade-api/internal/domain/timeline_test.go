package domain

import (
	"reflect"
	"testing"
)

func lastPoint(t *testing.T, g Game) TimelinePoint {
	t.Helper()
	if len(g.Timeline) == 0 {
		t.Fatal("timeline is empty")
	}
	return g.Timeline[len(g.Timeline)-1]
}

func stockOf(g Game) [5]int {
	var s [5]int
	for i, r := range Resources {
		s[i] = g.Inventory[r]
	}
	return s
}

func TestNewGameStartsTheTimeline(t *testing.T) {
	g, cfg := newTestGame()
	if len(g.Timeline) != 1 {
		t.Fatalf("timeline = %+v, want one start point", g.Timeline)
	}
	p := g.Timeline[0]
	if p.Kind != PointStart || p.Day != 1 || p.Capital != cfg.StartingCapital || p.Stock != [5]int{} {
		t.Fatalf("start point = %+v", p)
	}
	if g.Stats.PeakCapital != cfg.StartingCapital || g.Stats.PeakDay != 1 {
		t.Fatalf("stats = %+v", g.Stats)
	}
}

func TestBuyAndSellRecordPoints(t *testing.T) {
	g, cfg := newTestGame()
	ask := Quotes(g, cfg)[Lemon].Ask
	if err := Buy(&g, cfg, Lemon, 3); err != nil {
		t.Fatal(err)
	}
	p := lastPoint(t, g)
	if p.Kind != PointBuy || p.Resource != Lemon || p.Qty != 3 || p.Amount != 3*ask || p.Day != 1 {
		t.Fatalf("buy point = %+v", p)
	}
	if p.Capital != g.Capital || p.Stock != stockOf(g) || p.Stock[0] != 3 {
		t.Fatalf("buy snapshot = %+v, game capital %d stock %v", p, g.Capital, stockOf(g))
	}

	bid := Quotes(g, cfg)[Lemon].Bid
	if err := Sell(&g, cfg, Lemon, 2); err != nil {
		t.Fatal(err)
	}
	p = lastPoint(t, g)
	if p.Kind != PointSell || p.Resource != Lemon || p.Qty != 2 || p.Amount != 2*bid || p.Capital != g.Capital {
		t.Fatalf("sell point = %+v", p)
	}
	if g.Stats.CasesBought != 3 || g.Stats.Spent != 3*ask || g.Stats.CasesSold != 2 || g.Stats.Earned != 2*bid {
		t.Fatalf("stats = %+v", g.Stats)
	}
}

func TestFailedActionsRecordNothing(t *testing.T) {
	g, cfg := newTestGame()
	before := len(g.Timeline)
	stats := g.Stats
	_ = Buy(&g, cfg, Lemon, 999) // over capacity
	_ = Buy(&g, cfg, Lemon, 0)   // invalid
	_ = Sell(&g, cfg, Lemon, 1)  // nothing to sell
	g.Capital = 0
	_ = Expand(&g, cfg, Production, "") // cannot afford
	_ = Upgrade(&g, cfg, Warehouse)     // cannot afford
	if len(g.Timeline) != before || g.Stats != stats {
		t.Fatalf("failed actions changed the record: %d points, stats %+v", len(g.Timeline), g.Stats)
	}
}

func TestConsecutiveTradesCoalesce(t *testing.T) {
	g, cfg := newTestGame()
	ask := Quotes(g, cfg)[Lemon].Ask
	for i := 0; i < 4; i++ {
		if err := Buy(&g, cfg, Lemon, 1); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.Timeline) != 2 { // start + one merged buy
		t.Fatalf("timeline = %+v, want start + one merged buy", g.Timeline)
	}
	p := lastPoint(t, g)
	if p.Qty != 4 || p.Amount != 4*ask || p.Capital != g.Capital || p.Stock[0] != 4 {
		t.Fatalf("merged point = %+v", p)
	}

	// A different resource, a different kind, or a different day starts a new point.
	_ = Buy(&g, cfg, Sugar, 1)
	_ = Sell(&g, cfg, Sugar, 1)
	if len(g.Timeline) != 4 {
		t.Fatalf("timeline has %d points, want 4: %+v", len(g.Timeline), g.Timeline)
	}
	g.Inventory[Lemonade] = 1 // keep the game alive overnight
	if _, err := EndDay(&g, cfg); err != nil {
		t.Fatal(err)
	}
	before := len(g.Timeline)
	_ = Buy(&g, cfg, Lemon, 1)
	if len(g.Timeline) != before+1 {
		t.Fatal("a buy on a new day must not merge into yesterday's")
	}
}

func TestFacilityActionsRecordPoints(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 5000
	if err := Expand(&g, cfg, Warehouse, Ice); err != nil {
		t.Fatal(err)
	}
	p := lastPoint(t, g)
	if p.Kind != PointExpand || p.Facility != Warehouse || p.Resource != Ice || p.Amount != 100 || p.Capital != 4900 {
		t.Fatalf("warehouse expand = %+v", p)
	}
	if err := Expand(&g, cfg, Production, ""); err != nil {
		t.Fatal(err)
	}
	p = lastPoint(t, g)
	if p.Kind != PointExpand || p.Facility != Production || p.Resource != "" || p.Amount != 500 {
		t.Fatalf("production expand = %+v", p)
	}
	buildings := warehouseBuildings(g)
	if err := Upgrade(&g, cfg, Warehouse); err != nil {
		t.Fatal(err)
	}
	p = lastPoint(t, g)
	if p.Kind != PointUpgrade || p.Facility != Warehouse || p.Amount != 100*buildings {
		t.Fatalf("upgrade = %+v", p)
	}
	if g.Stats.FacilitiesBought != 2 || g.Stats.Upgrades != 1 || g.Stats.FacilitySpend != 100+500+100*buildings {
		t.Fatalf("stats = %+v", g.Stats)
	}
	// Two expansions in a row are two events, never merged.
	before := len(g.Timeline)
	_ = Expand(&g, cfg, Warehouse, Ice)
	_ = Expand(&g, cfg, Warehouse, Ice)
	if len(g.Timeline) != before+2 {
		t.Fatal("facility events must not coalesce")
	}
}

func TestEndDayRecordsAPoint(t *testing.T) {
	g, cfg := newTestGame()
	setInputs(&g, 10, 10, 10, 10)
	due := TotalUpkeep(g, cfg)
	report, err := EndDay(&g, cfg)
	if err != nil {
		t.Fatal(err)
	}
	p := lastPoint(t, g)
	if p.Kind != PointEndDay || p.Day != 1 || p.Produced != report.Produced || p.Amount != due {
		t.Fatalf("end-day point = %+v (report %+v)", p, report)
	}
	if p.Capital != g.Capital || p.Stock != stockOf(g) || p.Stock[2] != 0 {
		t.Fatalf("end-day snapshot = %+v (ice must have melted)", p)
	}
	if g.Stats.Produced != report.Produced || g.Stats.UpkeepPaid != due {
		t.Fatalf("stats = %+v", g.Stats)
	}
}

func TestForcedSalesCountAsSales(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 0
	g.Inventory[Lemonade] = 3
	report, _ := EndDay(&g, cfg)
	if report.ForcedSaleCases == 0 {
		t.Fatal("expected a forced sale")
	}
	if g.Stats.CasesSold != report.ForcedSaleCases || g.Stats.Earned != report.ForcedSaleProceeds {
		t.Fatalf("stats = %+v, report = %+v", g.Stats, report)
	}
}

func TestPeakCapitalTracksTheHighestPoint(t *testing.T) {
	g, cfg := newTestGame()
	g.Inventory[Lemonade] = 10
	if err := Sell(&g, cfg, Lemonade, 10); err != nil { // capital rises on day 1
		t.Fatal(err)
	}
	peak := g.Capital
	if err := Buy(&g, cfg, Lemon, 5); err != nil { // and falls again
		t.Fatal(err)
	}
	if g.Stats.PeakCapital != peak || g.Stats.PeakDay != 1 {
		t.Fatalf("peak = $%d on day %d, want $%d on day 1", g.Stats.PeakCapital, g.Stats.PeakDay, peak)
	}
}

func TestOldTradesAreCompactedButMilestonesStay(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 1_000_000
	for day := 1; day <= 12; day++ {
		if err := Expand(&g, cfg, Warehouse, Lemon); err != nil && day <= 8 {
			t.Fatal(err)
		}
		_ = Buy(&g, cfg, Lemon, 1)
		_ = Sell(&g, cfg, Lemon, 1)
		g.Inventory[Lemonade] = 1 // keep the game alive
		if _, err := EndDay(&g, cfg); err != nil {
			t.Fatal(err)
		}
	}

	recentFrom := g.Day - detailDays // days before this were compacted at the last action
	var oldTrades, oldMilestones, endDays int
	for _, p := range g.Timeline {
		switch {
		case p.Kind == PointEndDay:
			endDays++
		case (p.Kind == PointBuy || p.Kind == PointSell) && p.Day < recentFrom:
			oldTrades++
		case p.Kind == PointExpand && p.Day < recentFrom:
			oldMilestones++
		}
	}
	if oldTrades != 0 {
		t.Errorf("%d trade points older than %d days survived compaction", oldTrades, detailDays)
	}
	if oldMilestones == 0 {
		t.Error("old facility purchases were dropped; milestones must stay")
	}
	if endDays != 12 {
		t.Errorf("end-of-day points = %d, want one per day (12)", endDays)
	}
	if g.Timeline[0].Kind != PointStart {
		t.Error("the start point must be kept")
	}
}

func TestTimelineIsBounded(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 1_000_000
	for i := 0; i < 2*maxTimelinePoints; i++ {
		r := Resources[i%2] // alternate so nothing coalesces
		if i%4 < 2 {
			_ = Buy(&g, cfg, r, 1)
		} else {
			_ = Sell(&g, cfg, r, 1)
		}
	}
	if len(g.Timeline) > maxTimelinePoints {
		t.Fatalf("timeline has %d points, want at most %d", len(g.Timeline), maxTimelinePoints)
	}
	if g.Timeline[0].Kind != PointStart {
		t.Error("the start point must survive trimming")
	}
}

func TestCloneCopiesTheTimeline(t *testing.T) {
	g, cfg := newTestGame()
	_ = Buy(&g, cfg, Lemon, 1)
	c := g.Clone()
	if !reflect.DeepEqual(g.Timeline, c.Timeline) {
		t.Fatal("clone differs")
	}
	c.Timeline[0].Capital = -1
	c.Timeline = append(c.Timeline, TimelinePoint{})
	if g.Timeline[0].Capital == -1 || len(g.Timeline) != 2 {
		t.Fatal("clone shares the timeline with the original")
	}
}
