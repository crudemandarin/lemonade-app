package domain

import (
	"math/rand"
	"testing"
)

func TestAverageCostIsAWeightedAverage(t *testing.T) {
	g, cfg := newTestGame()
	Buy(&g, cfg, Lemon, 2) // 2 × $22
	g.Market[Lemon].Price = 30
	Buy(&g, cfg, Lemon, 2)                                      // 2 × $33
	if g.CostBasis[Lemon] != 44+66 || AvgCost(g, Lemon) != 28 { // 110/4 = 27.5 rounds to 28
		t.Fatalf("basis=%d avg=%d", g.CostBasis[Lemon], AvgCost(g, Lemon))
	}

	// Selling takes a proportional share of the basis, so the average stays put.
	Sell(&g, cfg, Lemon, 2)
	if g.CostBasis[Lemon] != 55 || AvgCost(g, Lemon) != 28 {
		t.Fatalf("after selling half: basis=%d avg=%d", g.CostBasis[Lemon], AvgCost(g, Lemon))
	}
	Sell(&g, cfg, Lemon, 2)
	if g.CostBasis[Lemon] != 0 || AvgCost(g, Lemon) != 0 {
		t.Fatalf("selling the last case must zero the basis exactly, got %d", g.CostBasis[Lemon])
	}
}

func TestProductionConservesCost(t *testing.T) {
	g, cfg := newTestGame()
	for _, r := range Inputs {
		Buy(&g, cfg, r, 7)
	}
	inputsBefore := 0
	for _, r := range Inputs {
		inputsBefore += g.CostBasis[r]
	}

	if n := produce(&g, cfg); n != 7 {
		t.Fatalf("produced %d", n)
	}
	inputsAfter := 0
	for _, r := range Inputs {
		inputsAfter += g.CostBasis[r]
		if g.Inventory[r] == 0 && g.CostBasis[r] != 0 {
			t.Errorf("%s: no stock but basis %d", r, g.CostBasis[r])
		}
	}
	if inputsAfter != 0 || g.CostBasis[Lemonade] != inputsBefore {
		t.Fatalf("inputs %d -> %d, lemonade basis %d, want %d", inputsBefore, inputsAfter, g.CostBasis[Lemonade], inputsBefore)
	}
	// 7 lemonade cost 7 × (22+11+11+11) = 385 to make.
	if AvgCost(g, Lemonade) != 55 {
		t.Fatalf("cost to make = %d, want 55", AvgCost(g, Lemonade))
	}
}

func TestProductionPartlyConsumesInputs(t *testing.T) {
	g, cfg := newTestGame()
	Buy(&g, cfg, Lemon, 10)
	Buy(&g, cfg, Sugar, 3)
	Buy(&g, cfg, Ice, 10)
	Buy(&g, cfg, Cup, 10)
	before := g.CostBasis[Lemon] + g.CostBasis[Sugar] + g.CostBasis[Ice] + g.CostBasis[Cup]
	produce(&g, cfg) // limited by sugar: 3
	after := g.CostBasis[Lemon] + g.CostBasis[Sugar] + g.CostBasis[Ice] + g.CostBasis[Cup]
	if g.CostBasis[Lemonade] != before-after {
		t.Fatalf("lemonade basis %d, want %d", g.CostBasis[Lemonade], before-after)
	}
}

func TestIceMeltZeroesItsBasis(t *testing.T) {
	g, cfg := newTestGame()
	Buy(&g, cfg, Ice, 5)
	if _, err := EndDay(&g, cfg); err != nil {
		t.Fatal(err)
	}
	if g.Inventory[Ice] != 0 || g.CostBasis[Ice] != 0 {
		t.Fatalf("ice stock %d basis %d", g.Inventory[Ice], g.CostBasis[Ice])
	}
}

func TestForcedSalesReduceBasis(t *testing.T) {
	g, cfg := newTestGame()
	Buy(&g, cfg, Lemon, 10)
	g.Capital = 0 // upkeep must come out of stock
	settleUpkeep(&g, cfg)
	if g.Inventory[Lemon] == 10 {
		t.Fatal("expected some stock to be sold")
	}
	want := 220 * g.Inventory[Lemon] / 10
	if d := g.CostBasis[Lemon] - want; d < -1 || d > 1 {
		t.Fatalf("basis %d for %d cases left, want about %d", g.CostBasis[Lemon], g.Inventory[Lemon], want)
	}
}

// Random play never breaks the invariants, and each sale's share of basis is
// within a dollar of the exact proportion.
func TestCostBasisInvariantsHoldUnderRandomPlay(t *testing.T) {
	cfg := DefaultConfig()
	rng := rand.New(rand.NewSource(3))
	for game := 0; game < 30; game++ {
		g := NewGame(cfg, int64(game))
		for step := 0; step < 400 && g.Status == StatusActive; step++ {
			r := Resources[rng.Intn(len(Resources))]
			switch rng.Intn(6) {
			case 0, 1:
				BuyClamped(&g, cfg, r, 1+rng.Intn(15))
			case 2:
				held, basis, qty := g.Inventory[r], g.CostBasis[r], 1+rng.Intn(10)
				if SellClamped(&g, cfg, r, qty) == nil && held > 0 {
					sold := held - g.Inventory[r]
					exact := float64(basis) * float64(sold) / float64(held)
					removed := basis - g.CostBasis[r]
					if d := float64(removed) - exact; d > 1 || d < -1 {
						t.Fatalf("removed %d of %d for %d/%d, exact %.2f", removed, basis, sold, held, exact)
					}
				}
			case 3:
				Expand(&g, cfg, Warehouse, r)
			default:
				EndDay(&g, cfg)
			}
			for _, res := range Resources {
				if g.CostBasis[res] < 0 {
					t.Fatalf("game %d step %d: %s basis %d", game, step, res, g.CostBasis[res])
				}
				if g.Inventory[res] == 0 && g.CostBasis[res] != 0 {
					t.Fatalf("game %d step %d: %s has no stock but basis %d", game, step, res, g.CostBasis[res])
				}
			}
		}
	}
}

func TestSeedCostBasisForOldSaves(t *testing.T) {
	g, cfg := newTestGame()
	g.CostBasis = nil
	g.Inventory[Lemon], g.Inventory[Sugar] = 4, 0
	g.Market[Lemon].Price = 22.4
	SeedCostBasis(&g)
	if g.CostBasis[Lemon] != 4*22 || g.CostBasis[Sugar] != 0 {
		t.Fatalf("seeded %v", g.CostBasis)
	}
	_ = cfg
	// A game that already has a basis is left alone.
	g.CostBasis[Lemon] = 7
	SeedCostBasis(&g)
	if g.CostBasis[Lemon] != 7 {
		t.Fatal("seed overwrote an existing basis")
	}
}

func TestUnrealizedGainIsAgainstTheBid(t *testing.T) {
	g, cfg := newTestGame()
	Buy(&g, cfg, Lemon, 5) // $110 spent; bid is $18 → 90 on hand
	if got := UnrealizedGain(g, cfg, Lemon); got != 90-110 {
		t.Fatalf("got %d, want -20", got)
	}
	if UnrealizedGain(g, cfg, Sugar) != 0 {
		t.Fatal("no stock means no gain")
	}
}
