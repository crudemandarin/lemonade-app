package domain

import (
	"reflect"
	"testing"

	"lemonade-api/internal/domain/content"
)

// playDays ends n days with no other action and returns the cycle after each.
func playDays(t *testing.T, seed int64, cfg Config, n int) []CycleState {
	t.Helper()
	g := NewGame(cfg, seed)
	g.Capital = 1_000_000 // survive upkeep; only the economy is under test
	var out []CycleState
	for i := 0; i < n; i++ {
		if _, err := EndDay(&g, cfg); err != nil {
			t.Fatal(err)
		}
		out = append(out, g.Cycle)
	}
	return out
}

func TestCyclesAreDeterministicAndOneAtATime(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CycleChance = 0.2 // start often, so a short run sees several
	a, b := playDays(t, 7, cfg, 200), playDays(t, 7, cfg, 200)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("the same seed must give the same cycles")
	}
	started, seen := 0, map[string]bool{}
	prev := CycleState{}
	for _, c := range a {
		if c.Key != "" {
			seen[c.Key] = true
			if prev.Key != "" && prev.Key != c.Key {
				t.Fatalf("regime went from %s to %s without a stable day between", prev.Key, c.Key)
			}
			if prev.Key == "" {
				started++
			}
			d, _ := cycleDef(c.Key)
			if c.DaysLeft < 1 || c.DaysLeft > d.MaxDays {
				t.Fatalf("%s has %d days left", c.Key, c.DaysLeft)
			}
		}
		prev = c
	}
	if started < 3 || len(seen) < 2 {
		t.Fatalf("expected several cycles of more than one kind, got %d starts and %v", started, seen)
	}
}

func TestNoCyclesMeansNoStateAndNothingChanges(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CycleChance = 0
	for _, c := range playDays(t, 3, cfg, 150) {
		if c != (CycleState{}) {
			t.Fatalf("no cycle should ever start, got %+v", c)
		}
	}
}

func TestCycleStreamDoesNotMoveTheMarketWalk(t *testing.T) {
	// Same seed, cycles on and off: until a regime changes the price rules, the walked
	// prices (before any effect on them) are the same, because the cycle has its own stream.
	off, on := DefaultConfig(), DefaultConfig()
	off.CycleChance = 0
	on.CycleChance = 1 // a cycle starts on day 2, but the walk itself must not shift
	a, b := NewGame(off, 11), NewGame(on, 11)
	a.Capital, b.Capital = 1_000_000, 1_000_000
	for i := 0; i < 30; i++ {
		if _, err := EndDay(&a, off); err != nil {
			t.Fatal(err)
		}
		if _, err := EndDay(&b, on); err != nil {
			t.Fatal(err)
		}
		for _, r := range off.Resources() {
			if a.Market[r].Price != b.Market[r].Price {
				t.Fatalf("day %d: %s walked %v with cycles off but %v with them on", a.Day, r, a.Market[r].Price, b.Market[r].Price)
			}
		}
	}
}

func TestCycleEffects(t *testing.T) {
	cfg := DefaultConfig()
	base := NewGame(cfg, 5)
	with := func(key string) Game {
		g := base.Clone()
		g.Cycle = CycleState{Key: key, DaysLeft: 10}
		return g
	}
	d0 := freeDepth(base, cfg, Lemon)
	if got, want := freeDepth(with(content.CycleBoom), cfg, Lemon), d0*115/100; got != want {
		t.Errorf("boom depth %d, want %d", got, want)
	}
	if got, want := freeDepth(with(content.CycleRecession), cfg, Lemon), d0*85/100; got != want {
		t.Errorf("recession depth %d, want %d", got, want)
	}

	// Rival valuations: a boom raises the buyout price, a recession lowers it.
	p0, _ := BuyoutPrice(base, cfg, "lil_lucy", false)
	boom, _ := BuyoutPrice(with(content.CycleBoom), cfg, "lil_lucy", false)
	rec, _ := BuyoutPrice(with(content.CycleRecession), cfg, "lil_lucy", false)
	if !(rec < p0 && p0 < boom) {
		t.Errorf("buyout prices: recession %d, stable %d, boom %d", rec, p0, boom)
	}

	// Inflation raises upkeep by 10% while it runs.
	g := with(content.CycleInflation)
	g.WarehouseLevel = 1
	plain := TotalUpkeep(base, cfg)
	if plain > 0 {
		if got, want := TotalUpkeep(g, cfg), plain+int(float64(plain)*0.1+0.5); got != want {
			t.Errorf("inflation upkeep %d, want %d", got, want)
		}
	}
}

func TestInflationDriftBuildsThenUnwinds(t *testing.T) {
	cfg := DefaultConfig()
	g := NewGame(cfg, 9)
	g.Cycle = CycleState{Key: content.CycleInflation, DaysLeft: 5}
	var rep DayReport
	cfg.CycleChance = 0
	peak := 1.0
	for i := 0; i < 5; i++ {
		g.Day++
		stepCycles(&g, cfg, &rep)
		peak = max(peak, priceDrift(g))
	}
	if g.Cycle.Key != "" || rep.CycleEnded != content.CycleInflation {
		t.Fatalf("inflation should have ended, cycle %+v ended %q", g.Cycle, rep.CycleEnded)
	}
	if peak < 1.0303 {
		t.Fatalf("four days of 1%% should have built about 4%%, got %v", peak)
	}
	for i := 0; i < 400 && priceDrift(g) > 1; i++ {
		g.Day++
		stepCycles(&g, cfg, &rep)
	}
	if priceDrift(g) != 1 {
		t.Fatalf("drift should unwind to 1, got %v", priceDrift(g))
	}
}
