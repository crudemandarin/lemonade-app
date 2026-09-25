package domain

// PnlLine is the bookkeeper's profit and loss for one day: what the player sold for and
// spent on stock and buildings that day, what upkeep and ice making cost, and the
// result. Trades are read from the timeline, which keeps every trade of the recent days.
type PnlLine struct {
	Sales      int
	Purchases  int
	Facilities int
	Upkeep     int
	IceMade    int
	Net        int
}

// pnlFor builds the line for the day that is ending (g.Day), after upkeep is settled.
func pnlFor(g Game, r DayReport) PnlLine {
	p := PnlLine{Upkeep: r.UpkeepPaid, IceMade: r.IceMadeCost, Sales: r.ForcedSaleProceeds}
	for _, pt := range g.Timeline {
		if pt.Day != g.Day {
			continue
		}
		switch pt.Kind {
		case PointSell:
			p.Sales += pt.Amount
		case PointBuy:
			p.Purchases += pt.Amount
		case PointExpand, PointUpgrade:
			p.Facilities += pt.Amount
		}
	}
	p.Net = p.Sales - p.Purchases - p.Facilities - p.Upkeep - p.IceMade
	return p
}
