package domain

// liquidationOrder is the order stock is sold in when cash cannot cover upkeep:
// the finished product first (it would be sold anyway), then raw inputs by value.
// Ice is last; it has already melted by the time upkeep is settled.
var liquidationOrder = []Resource{Lemonade, Lemon, Sugar, Cup, Ice}

// settleUpkeep charges one day's upkeep. Upkeep is always owed: if cash falls short,
// stock is sold at the current bid, just enough to cover it. If even everything
// sold is not enough, the player pays what they can and is insolvent, which ends the
// game (DECISIONS 16; this replaces "unpaid upkeep is forgiven" and "any leftover
// stock is a grace"). It returns what was paid and what, if anything, was sold.
func settleUpkeep(g *Game, cfg Config) (paid, soldCases, soldProceeds int, insolvent bool) {
	due := TotalUpkeep(*g, cfg)
	if g.Capital < due {
		soldCases, soldProceeds = sellStockToCover(g, cfg, due-g.Capital)
	}
	paid = due
	if g.Capital < due {
		paid, insolvent = g.Capital, true
	}
	g.Capital -= paid
	return paid, soldCases, soldProceeds, insolvent
}

// sellStockToCover sells stock at bid, in liquidationOrder, until need dollars have
// been raised or nothing is left. Each case is priced with the player's own selling
// impact, so a big forced sale takes more cases to raise the same cash.
func sellStockToCover(g *Game, cfg Config, need int) (cases, proceeds int) {
	quotes := Quotes(*g, cfg)
	for _, r := range liquidationOrder {
		if need <= 0 {
			break
		}
		stock := g.Inventory[r]
		if stock == 0 {
			continue
		}
		n, raised := 0, 0
		for n < stock && raised < need {
			n++
			raised += unitBid(cfg, r, quotes[r].Bid, g.SellPressure[r], n)
		}
		g.removeStock(r, n)
		g.addPressure(false, r, n)
		g.Capital += raised
		g.Stats.CasesSold += n
		g.Stats.Earned += raised
		cases += n
		proceeds += raised
		need -= raised
	}
	return cases, proceeds
}
