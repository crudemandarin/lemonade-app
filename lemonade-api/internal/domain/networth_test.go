package domain

import "testing"

func TestNetWorthCountsCashStockAtBidAndBuildingResale(t *testing.T) {
	g, cfg := newTestGame()
	// Start: $1,000 cash; five Pantries ($50 each) and one Kitchen ($250) resell for $500.
	if nw := NetWorthBreakdown(g, cfg); nw.Cash != 1000 || nw.Stock != 0 || nw.Facilities != 500 || nw.Total != 1500 {
		t.Fatalf("%+v", nw)
	}
	Buy(&g, cfg, Lemon, 5) // $110 spent; the bid is $18, so the stock is worth $90
	nw := NetWorthBreakdown(g, cfg)
	if nw.Cash != 890 || nw.Stock != 90 || nw.Total != 890+90+500 {
		t.Fatalf("%+v", nw)
	}
	if NetWorth(g, cfg) != nw.Total {
		t.Fatal("NetWorth and its breakdown disagree")
	}
}

func TestNetWorthTracksBuildingSalesFairly(t *testing.T) {
	g, cfg := newTestGame()
	g.ProductionQty = 2
	before := NetWorth(g, cfg)
	if err := SellFacility(&g, cfg, Production, ""); err != nil {
		t.Fatal(err)
	}
	// Selling turns resale value into cash one for one, so net worth does not move.
	if NetWorth(g, cfg) != before {
		t.Fatalf("net worth %d -> %d", before, NetWorth(g, cfg))
	}
}

func TestFinishRunRecordsTheOutcome(t *testing.T) {
	g, cfg := newTestGame()
	for i := 0; i < 3; i++ {
		EndDay(&g, cfg)
	}
	g.RunID = "run-1"
	rec := FinishRun(g, cfg, EndedByGaveUp)
	if rec.RunID != "run-1" || rec.Days != g.Day || rec.EndedBy != EndedByGaveUp {
		t.Fatalf("%+v", rec)
	}
	if rec.NetWorth != NetWorth(g, cfg) || rec.Score != rec.NetWorth || rec.Capital != g.Capital {
		t.Fatalf("score %d net worth %d capital %d", rec.Score, rec.NetWorth, rec.Capital)
	}
	if len(rec.Timeline) != len(g.Timeline) || len(rec.PriceLog) != len(g.PriceLog) || rec.Stats != g.Stats {
		t.Fatal("record lacks the run's history")
	}
	// The record is a copy: later changes to the game do not leak into it.
	g.Timeline[0].Capital = -1
	if rec.Timeline[0].Capital == -1 {
		t.Fatal("record aliases the game's timeline")
	}
}
