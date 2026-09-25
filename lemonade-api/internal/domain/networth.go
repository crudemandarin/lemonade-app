package domain

// NetWorthParts is what a player is worth right now: cash, plus stock at the current
// bid (what it would raise), plus what every building would resell for.
type NetWorthParts struct {
	Cash       int
	Stock      int
	Facilities int
	Total      int
}

// NetWorthBreakdown returns the parts and the total.
func NetWorthBreakdown(g Game, cfg Config) NetWorthParts {
	quotes := Quotes(g, cfg)
	stock := 0
	for _, r := range cfg.Resources() {
		stock += g.Inventory[r] * quotes[r].Bid
	}
	nw := NetWorthParts{Cash: g.Capital, Stock: stock, Facilities: FacilityResaleValue(g, cfg)}
	nw.Total = nw.Cash + nw.Stock + nw.Facilities
	return nw
}

// NetWorth is the total of NetWorthBreakdown. The score of a run is its final net worth.
func NetWorth(g Game, cfg Config) int { return NetWorthBreakdown(g, cfg).Total }
