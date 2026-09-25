package domain

// NetWorthParts is what a player is worth right now: cash, plus stock at the current
// bid (what it would raise), plus what every building would resell for, plus rivals
// bought out at ResaleRate times the price paid (entry costs and campaigns do not count).
type NetWorthParts struct {
	Cash       int
	Stock      int
	Facilities int
	// Acquisitions is what the rivals the player bought out count for.
	Acquisitions int
	Total        int
}

// NetWorthBreakdown returns the parts and the total.
func NetWorthBreakdown(g Game, cfg Config) NetWorthParts {
	quotes := Quotes(g, cfg)
	stock := 0
	for _, r := range cfg.Resources() {
		stock += g.Inventory[r] * quotes[r].Bid
	}
	nw := NetWorthParts{Cash: g.Capital, Stock: stock, Facilities: FacilityResaleValue(g, cfg), Acquisitions: AcquisitionValue(g, cfg)}
	nw.Total = nw.Cash + nw.Stock + nw.Facilities + nw.Acquisitions
	return nw
}

// NetWorth is the total of NetWorthBreakdown. The score of a run is its final net worth.
func NetWorth(g Game, cfg Config) int { return NetWorthBreakdown(g, cfg).Total }
