package domain

// PricePoint is one day's effective prices as the player saw them (after the
// market tick and events), in Resources order, plus the events active that day.
// Like the timeline it is a record: nothing in the rules reads it back.
type PricePoint struct {
	Day    int
	Prices [5]int
	Events []string // active event keys
}

// logPrices appends today's prices to the log.
func (g *Game) logPrices(cfg Config) {
	var p PricePoint
	quotes := Quotes(*g, cfg)
	for i, r := range Resources {
		p.Prices[i] = quotes[r].Price
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
	p := PricePoint{Day: g.Day}
	for i, r := range Resources {
		if m := g.Market[r]; m != nil && len(m.History) > 0 {
			p.Prices[i] = m.History[len(m.History)-1]
		}
	}
	for _, e := range g.Events {
		p.Events = append(p.Events, e.Key)
	}
	g.PriceLog = []PricePoint{p}
}
