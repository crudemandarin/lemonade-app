package domain

// Cost basis is the running weighted-average cost of the stock on hand, kept as
// total dollars per resource (not per case) so rounding never accumulates.
// It is a record for the player's benefit: nothing in the rules reads it.

func (g *Game) addBasis(r Resource, dollars int) {
	if g.CostBasis == nil {
		g.CostBasis = make(map[Resource]int)
	}
	g.CostBasis[r] += dollars
}

// removeStock takes n cases out of inventory and the matching share of basis,
// rounded to the nearest dollar; taking the last case zeroes the basis exactly.
// It returns the basis removed.
func (g *Game) removeStock(r Resource, n int) int {
	held, basis := g.Inventory[r], g.CostBasis[r]
	removed := 0
	switch {
	case held <= 0 || n <= 0:
	case n >= held:
		removed = basis
	default:
		removed = (2*basis*n + held) / (2 * held)
	}
	if removed > basis {
		removed = basis
	}
	if removed < 0 {
		removed = 0
	}
	g.Inventory[r] -= n
	g.takeOldest(r, n)
	if r == Ice {
		// Ice held over by a freezer is the oldest, so it goes first.
		g.IceOld = max(0, g.IceOld-n)
	}
	if g.CostBasis != nil {
		g.CostBasis[r] = basis - removed
	}
	return removed
}

// AvgCost is the average dollars paid per case held, rounded; 0 when none is held.
func AvgCost(g Game, r Resource) int {
	held := g.Inventory[r]
	if held <= 0 {
		return 0
	}
	return (2*g.CostBasis[r] + held) / (2 * held)
}

// UnrealizedGain is what the stock would raise at the current bid, minus what it cost.
func UnrealizedGain(g Game, cfg Config, r Resource) int {
	if g.Inventory[r] <= 0 {
		return 0
	}
	return g.Inventory[r]*Quotes(g, cfg)[r].Bid - g.CostBasis[r]
}

// SeedCostBasis gives a game saved before cost basis existed an approximate one:
// stock at the current walked price. It leaves a game that already has one alone.
func SeedCostBasis(g *Game) {
	if g.CostBasis != nil {
		return
	}
	g.CostBasis = make(map[Resource]int, len(g.Inventory))
	for r, held := range g.Inventory {
		if held > 0 && g.Market[r] != nil {
			g.CostBasis[r] = held * int(g.Market[r].Price+0.5)
		}
	}
}
