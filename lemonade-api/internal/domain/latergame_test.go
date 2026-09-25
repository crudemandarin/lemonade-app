package domain

import (
	"math"
	"testing"
)

// The impact shape relative to depth must reproduce the old flat 0.3%-a-case slope at the
// level-1 depth of 80, so early play is exactly as it was.
func TestLevelOneImpactMatchesTheOldFlatSlope(t *testing.T) {
	cfg := DefaultConfig()
	const depth = 80
	for _, plain := range []int{9, 11, 22, 81, 99} {
		for pressure := 0.0; pressure <= 400; pressure += 7 {
			for k := 1; k <= 300; k += 3 {
				excess := pressure + float64(k) - depth
				var oldAsk, oldBid int
				switch {
				case excess <= 0:
					oldAsk, oldBid = plain, plain
				default:
					m := math.Min(0.003*excess, cfg.ImpactCap)
					oldAsk = snapUp(float64(plain) * (1 + m))
					oldBid = max(1, snapDown(float64(plain)*(1-m)))
				}
				if got := unitAsk(cfg, depth, plain, pressure, k); got != oldAsk {
					t.Fatalf("ask plain=%d pressure=%v k=%d: got %d, old formula %d", plain, pressure, k, got, oldAsk)
				}
				if got := unitBid(cfg, depth, plain, pressure, k); got != oldBid {
					t.Fatalf("bid plain=%d pressure=%v k=%d: got %d, old formula %d", plain, pressure, k, got, oldBid)
				}
			}
		}
	}
}

func TestFreeDepthGrowsWithWarehouseLevel(t *testing.T) {
	g, cfg := newTestGame()
	for level, want := range []int{80, 280, 480, 960} {
		g.WarehouseLevel = level + 1
		for _, r := range Resources {
			if got := freeDepth(g, cfg, r); got != want {
				t.Fatalf("level %d %s: free depth %d, want %d", level+1, r, got, want)
			}
		}
		if got := FreeDepthLeft(g, cfg, Lemonade, 30); got != want-30 {
			t.Fatalf("level %d: depth left %d, want %d", level+1, got, want-30)
		}
	}
}

// The same share of the depth moves the price the same amount at every size.
func TestImpactIsTheSameShareOfDepthAtEveryLevel(t *testing.T) {
	cfg := DefaultConfig()
	base := unitAsk(cfg, 80, 1000, 0, 100) // 20 cases past a depth of 80: 25% of depth
	for _, depth := range []int{160, 320, 640} {
		k := depth + depth/4
		if got := unitAsk(cfg, depth, 1000, 0, k); got != base {
			t.Fatalf("depth %d: %d, want %d", depth, got, base)
		}
	}
}

// Tier sizes only grow and upkeep only falls, so a game saved under the old table always
// fits its stock in the new capacity and never pays more than before.
func TestOldSavesFitTheNewCapacity(t *testing.T) {
	oldSize := []int{10, 20, 40, 80}
	oldWarehouseUpkeep := []int{2, 6, 16, 40}
	oldProductionUpkeep := []int{20, 50, 120, 280}
	cfg := DefaultConfig()
	for i, size := range oldSize {
		level := i + 1
		g := NewGame(cfg, 1)
		g.WarehouseLevel, g.ProductionLevel = level, level
		for _, r := range Resources {
			g.WarehouseQty[r] = 3
			g.Inventory[r] = 3 * size // full under the old table
		}
		for _, r := range Resources {
			if Capacity(g, cfg, r) < g.Inventory[r] {
				t.Fatalf("level %d %s: capacity %d below stored stock %d", level, r, Capacity(g, cfg, r), g.Inventory[r])
			}
		}
		if warehouseTier(cfg, level).Upkeep > oldWarehouseUpkeep[i] || productionTier(cfg, level).Upkeep > oldProductionUpkeep[i] {
			t.Fatalf("level %d: upkeep rose", level)
		}
	}
}

// steadyNetPerDay models a player who sells q lemonade and buys q of each input every
// day, forever, with ten buildings of level lvl: the market starts each day with the
// pressure left from yesterday (q times Recovery's complement over Recovery), the k-th
// case of the day is priced at that pressure plus k, and the buildings needed to move q
// cases cost their upkeep. It returns the best net profit over q and that q.
func steadyNetPerDay(cfg Config, lvl int) (best, bestQ int) {
	g := NewGame(cfg, 1)
	g.WarehouseLevel, g.ProductionLevel = lvl, lvl
	quotes := Quotes(g, cfg)
	prod, ware := productionTier(cfg, lvl), warehouseTier(cfg, lvl)
	capacity := 10 * prod.Size
	best = math.MinInt
	for q := 1; q <= capacity; q++ {
		start := float64(q) * (1 - cfg.Recovery) / cfg.Recovery
		gross := 0
		for k := 1; k <= q; k++ {
			gross += unitBid(cfg, freeDepth(g, cfg, Lemonade), quotes[Lemonade].Bid, start, k)
			for _, in := range Inputs {
				gross -= unitAsk(cfg, freeDepth(g, cfg, in), quotes[in].Ask, start, k)
			}
		}
		buildings := (q + prod.Size - 1) / prod.Size
		whBuildings := (q + ware.Size - 1) / ware.Size
		upkeep := buildings*prod.Upkeep + len(Resources)*whBuildings*ware.Upkeep
		if net := gross - upkeep; net > best {
			best, bestQ = net, q
		}
	}
	return best, bestQ
}

// The permanent guard against the plateau: with the same number of buildings, each
// production level must earn strictly more per day than the one below it.
func TestEachLevelPaysMoreThanTheLast(t *testing.T) {
	cfg := DefaultConfig()
	prev := math.MinInt
	for lvl := 1; lvl <= cfg.MaxLevel; lvl++ {
		net, q := steadyNetPerDay(cfg, lvl)
		t.Logf("L%d %s: best volume %d a day, net $%d a day", lvl, productionTier(cfg, lvl).Name, q, net)
		if net <= prev {
			t.Fatalf("L%d nets $%d a day, no more than L%d's $%d", lvl, net, lvl-1, prev)
		}
		prev = net
	}
}
