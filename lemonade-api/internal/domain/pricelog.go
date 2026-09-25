package domain

// PricePoint is one day's effective prices as the player saw them (after the
// market tick and events) per commodity, plus the events active that day.
// Like the timeline it is a record: nothing in the rules reads it back.
type PricePoint struct {
	Day    int
	Prices map[Resource]int
	Events []string // active event keys
}

// logPrices appends today's prices to the log.
func (g *Game) logPrices(cfg Config) {
	p := PricePoint{Prices: make(map[Resource]int, len(cfg.Commodities))}
	quotes := Quotes(*g, cfg)
	for _, r := range cfg.Resources() {
		p.Prices[r] = quotes[r].Price
	}
	for _, e := range g.Events {
		p.Events = append(p.Events, e.Key)
	}
	p.Day = g.Day
	g.PriceLog = append(g.PriceLog, p)
}

// SeedPriceLog gives a game saved before the log existed a first point from the
// prices it last showed (the newest history entry), so its chart starts today.
// It leaves a game that already has a log alone.
func SeedPriceLog(g *Game) {
	if len(g.PriceLog) > 0 {
		return
	}
	p := PricePoint{Day: g.Day, Prices: make(map[Resource]int, len(g.Market))}
	for r, m := range g.Market {
		if m != nil && len(m.History) > 0 {
			p.Prices[r] = m.History[len(m.History)-1]
		}
	}
	for _, e := range g.Events {
		p.Events = append(p.Events, e.Key)
	}
	g.PriceLog = []PricePoint{p}
}
