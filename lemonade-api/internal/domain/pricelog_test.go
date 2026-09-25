package domain

import "testing"

func TestNewGameStartsThePriceLog(t *testing.T) {
	g, cfg := newTestGame()
	if len(g.PriceLog) != 1 {
		t.Fatalf("got %d points", len(g.PriceLog))
	}
	p := g.PriceLog[0]
	if p.Day != 1 || len(p.Events) != 0 {
		t.Fatalf("%+v", p)
	}
	for _, r := range Resources {
		if p.Prices[r] != cfg.BasePrice[r] {
			t.Errorf("%s: %d, want base %d", r, p.Prices[r], cfg.BasePrice[r])
		}
	}
}

func TestEndDayAppendsThePricesThePlayerSees(t *testing.T) {
	g, cfg := newTestGame()
	cfg.EventChance = 1
	for day := 1; day <= 30; day++ {
		if _, err := EndDay(&g, cfg); err != nil {
			t.Fatal(err)
		}
		if len(g.PriceLog) != day+1 {
			t.Fatalf("day %d: %d points, want %d", day, len(g.PriceLog), day+1)
		}
		last := g.PriceLog[len(g.PriceLog)-1]
		if last.Day != g.Day {
			t.Fatalf("point is for day %d, game is on day %d", last.Day, g.Day)
		}
		quotes := Quotes(g, cfg)
		for _, r := range Resources {
			if last.Prices[r] != quotes[r].Price {
				t.Fatalf("day %d %s: logged %d, shown %d", g.Day, r, last.Prices[r], quotes[r].Price)
			}
		}
		if len(last.Events) != len(g.Events) {
			t.Fatalf("day %d: logged events %v, active %v", g.Day, last.Events, g.Events)
		}
		for i, e := range g.Events {
			if last.Events[i] != e.Key {
				t.Fatalf("event %d: %s vs %s", i, last.Events[i], e.Key)
			}
		}
	}
}

func TestBankruptEndDayDoesNotLogAPrice(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 0
	if _, err := EndDay(&g, cfg); err != nil || g.Status != StatusBankrupt {
		t.Fatalf("err=%v status=%s", err, g.Status)
	}
	if len(g.PriceLog) != 1 {
		t.Fatalf("%d points, want only the start", len(g.PriceLog))
	}
}

func TestSeedPriceLogForOldSaves(t *testing.T) {
	g, cfg := newTestGame()
	for i := 0; i < 3; i++ {
		EndDay(&g, cfg)
	}
	g.PriceLog = nil
	SeedPriceLog(&g)
	if len(g.PriceLog) != 1 || g.PriceLog[0].Day != g.Day {
		t.Fatalf("%+v", g.PriceLog)
	}
	quotes := Quotes(g, cfg)
	for _, r := range Resources {
		if g.PriceLog[0].Prices[r] != quotes[r].Price {
			t.Errorf("%s: %d vs %d", r, g.PriceLog[0].Prices[r], quotes[r].Price)
		}
	}
	SeedPriceLog(&g)
	if len(g.PriceLog) != 1 {
		t.Fatal("seeding twice added a point")
	}
}
