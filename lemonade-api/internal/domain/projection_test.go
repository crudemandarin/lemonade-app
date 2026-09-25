package domain

import (
	"math/rand"
	"testing"
)

// The preview must equal what EndDay really does, for any inventory and facilities.
func TestPreviewMatchesEndDay(t *testing.T) {
	cfg := DefaultConfig()
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 500; i++ {
		g := NewGame(cfg, int64(i))
		g.ProductionLevel = 1 + rng.Intn(cfg.MaxLevel)
		g.ProductionQty = 1 + rng.Intn(cfg.MaxQuantity)
		g.WarehouseLevel = 1 + rng.Intn(cfg.MaxLevel)
		for _, r := range Resources {
			g.WarehouseQty[r] = 1 + rng.Intn(cfg.MaxQuantity)
			g.Inventory[r] = rng.Intn(Capacity(g, cfg, r) + 1)
		}
		g.Capital = 1_000_000 // solvent: EndDay's report is unaffected either way
		before := g.Clone()

		p := PreviewEndDay(g, cfg)
		if !reflectEqualGame(before, g) {
			t.Fatal("PreviewEndDay mutated the game")
		}
		report, err := EndDay(&g, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if p.LemonadeToProduce != report.Produced || p.IceToMelt != report.IceMelted {
			t.Fatalf("case %d: preview %+v, actual produced=%d melted=%d (inv %v)", i, p, report.Produced, report.IceMelted, before.Inventory)
		}
	}
}

func reflectEqualGame(a, b Game) bool {
	for _, r := range Resources {
		if a.Inventory[r] != b.Inventory[r] {
			return false
		}
	}
	return a.Capital == b.Capital && a.Day == b.Day && len(a.Timeline) == len(b.Timeline)
}

func TestPreviewLimitedBy(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(g *Game)
		qty     int
		ice     int
		limited string
	}{
		{"zero stock", func(g *Game) {}, 0, 0, "lemon"},
		{"production bound", func(g *Game) { roomy(g); setInputs(g, 15, 15, 15, 15) }, 10, 5, "production"},
		{"tie goes to the input", func(g *Game) { setInputs(g, 10, 10, 10, 10) }, 10, 0, "lemon"},
		{"lemon bound", func(g *Game) { setInputs(g, 3, 10, 10, 10) }, 3, 7, "lemon"},
		{"sugar bound", func(g *Game) { setInputs(g, 10, 4, 10, 10) }, 4, 6, "sugar"},
		{"ice bound", func(g *Game) { setInputs(g, 10, 10, 5, 10) }, 5, 0, "ice"},
		{"cup bound", func(g *Game) { setInputs(g, 10, 10, 10, 2) }, 2, 8, "cup"},
		{"full lemonade warehouse", func(g *Game) { setInputs(g, 10, 10, 10, 10); g.Inventory[Lemonade] = 6 }, 4, 6, "space"},
		{"ice larger than production", func(g *Game) { roomy(g); setInputs(g, 15, 15, 25, 15) }, 10, 15, "production"},
		{"no capacity", func(g *Game) { setInputs(g, 5, 5, 5, 5); g.ProductionQty = 0 }, 0, 5, ""},
	}
	for _, tt := range tests {
		g, cfg := newTestGame()
		tt.setup(&g)
		p := PreviewEndDay(g, cfg)
		if p.LemonadeToProduce != tt.qty || p.IceToMelt != tt.ice || p.LimitedBy != tt.limited {
			t.Errorf("%s: got %+v, want qty=%d ice=%d limitedBy=%q", tt.name, p, tt.qty, tt.ice, tt.limited)
		}
	}
}

// roomy triples every warehouse so stock can exceed one day's production.
func roomy(g *Game) {
	for _, r := range Resources {
		g.WarehouseQty[r] = 3
	}
}
