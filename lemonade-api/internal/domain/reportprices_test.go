package domain

import "testing"

// 9a: the day report always lists every resource, in display order.
func TestEndDayReportsAllFivePrices(t *testing.T) {
	g, cfg := newTestGame()
	cfg.Sigma = 0 // no random walk, and at base price: nothing moves
	cfg.EventChance = 0
	report, err := EndDay(&g, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.PriceChanges) != len(Resources) {
		t.Fatalf("got %d price entries, want %d: %+v", len(report.PriceChanges), len(Resources), report.PriceChanges)
	}
	for i, c := range report.PriceChanges {
		if c.Resource != Resources[i] {
			t.Errorf("entry %d is %s, want %s", i, c.Resource, Resources[i])
		}
		if c.Before != c.After {
			t.Errorf("%s changed %d -> %d with no volatility", c.Resource, c.Before, c.After)
		}
	}
}

func TestEndDayReportsMixedPriceChanges(t *testing.T) {
	g, cfg := newTestGame()
	cfg.Sigma = 0
	cfg.EventChance = 0
	g.Market[Lemon].Price = 40 // reverts toward base 20 -> 37; others stay put
	before := Quotes(g, cfg)
	report, err := EndDay(&g, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.PriceChanges) != len(Resources) {
		t.Fatalf("got %d entries, want %d", len(report.PriceChanges), len(Resources))
	}
	after := Quotes(g, cfg)
	for _, c := range report.PriceChanges {
		if c.Before != before[c.Resource].Price || c.After != after[c.Resource].Price {
			t.Errorf("%s: %d -> %d, want %d -> %d", c.Resource, c.Before, c.After, before[c.Resource].Price, after[c.Resource].Price)
		}
	}
	if report.PriceChanges[0].Resource != Lemon || report.PriceChanges[0].Before == report.PriceChanges[0].After {
		t.Errorf("lemon should have changed: %+v", report.PriceChanges[0])
	}
}
