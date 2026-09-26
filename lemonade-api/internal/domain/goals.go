package domain

import (
	"slices"
	"sort"
)

// GoalStats are run-scoped facts that only achievements read; no rule of the game
// reads them back, so they can never change how a run plays. Buy and Sell update the
// trading facts as they happen. The end-of-day facts are recorded by RecordDayFacts,
// which the API layer calls after EndDay (not inside it), so the bots and the balance
// report never see them.
type GoalStats struct {
	// DaysClosed counts the days ended since these facts were first tracked (0 on a
	// game saved before they existed, until its next end of day).
	DaysClosed int
	// LastClosingNetWorth is the net worth right after the last end of day.
	LastClosingNetWorth int
	// LastDayProfit is how much the net worth grew over the last ended day (closing to
	// closing); BestDayProfit is the most it ever grew in one day.
	LastDayProfit int
	BestDayProfit int
	// LowestClosingLiquid is the least cash plus stock at the bid after any end of day.
	LowestClosingLiquid int
	// MostProducedInADay is the most lemonade made in one night.
	MostProducedInADay int
	// FullProductionStreak counts consecutive days that produced at full capacity.
	FullProductionStreak  int
	LongestFullProduction int
	// LastTradeDay is the last day the player bought or sold anything (0: never).
	LastTradeDay int
	// IdleStreak counts consecutive days ended without a trade.
	IdleStreak  int
	LongestIdle int
	// EventsSeen are the keys of every market event active at some point, sorted.
	EventsSeen []string
	// Sold counts cases the player sold, per commodity key (forced sales excluded).
	Sold map[string]int
	// EventSales counts cases sold while an event was active, keyed "event/commodity".
	EventSales map[string]int
	// TradesWithImpact counts buys and sells that paid price impact; SalesWithImpact
	// only the sells.
	TradesWithImpact int
	SalesWithImpact  int
	// LowestInputBuyPercent is the cheapest input purchase as a whole percent of its base
	// price, rounded up (0: none yet).
	LowestInputBuyPercent int
	// BestSellPercent is the best sale per commodity as a percent of its base price, and
	// BestCostPercent as a percent of what the cases sold had cost; both rounded down.
	BestSellPercent map[string]int
	BestCostPercent map[string]int
	// PriceWarsWon counts price wars that ended with the player holding at least the total
	// territory share they had when it began; WarStartShare is that share while one runs.
	PriceWarsWon  int
	WarTracking   bool
	WarStartShare float64
}

func (s GoalStats) clone() GoalStats {
	c := s
	c.EventsSeen = slices.Clone(s.EventsSeen)
	c.Sold = cloneCounts(s.Sold)
	c.EventSales = cloneCounts(s.EventSales)
	c.BestSellPercent = cloneCounts(s.BestSellPercent)
	c.BestCostPercent = cloneCounts(s.BestCostPercent)
	return c
}

func cloneCounts(m map[string]int) map[string]int {
	if m == nil {
		return nil
	}
	c := make(map[string]int, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func bump(m *map[string]int, key string, n int) {
	if *m == nil {
		*m = map[string]int{}
	}
	(*m)[key] += n
}

func raise(m *map[string]int, key string, v int) {
	if *m == nil {
		*m = map[string]int{}
	}
	if v > (*m)[key] {
		(*m)[key] = v
	}
}

// eventSalesKey names one event and commodity pair in GoalStats.EventSales.
func eventSalesKey(event string, r Resource) string { return event + "/" + string(r) }

// noteBuy records a purchase of qty cases for cost, which would have been plain
// without price impact.
func (g *Game) noteBuy(cfg Config, r Resource, qty, cost, plain int) {
	s := &g.Goals
	s.LastTradeDay = g.Day
	if cost > plain {
		s.TradesWithImpact++
	}
	if base := cfg.BasePrice[r]; base > 0 && slices.Contains(cfg.Inputs(), r) {
		pct := (cost*100 + qty*base - 1) / (qty * base) // rounded up: never flatters
		if s.LowestInputBuyPercent == 0 || pct < s.LowestInputBuyPercent {
			s.LowestInputBuyPercent = pct
		}
	}
}

// noteSell records a sale of qty cases for proceeds (plain without impact) of stock
// that had cost basis dollars.
func (g *Game) noteSell(cfg Config, r Resource, qty, proceeds, plain, basis int) {
	s := &g.Goals
	s.LastTradeDay = g.Day
	bump(&s.Sold, string(r), qty)
	if proceeds < plain {
		s.TradesWithImpact++
		s.SalesWithImpact++
	}
	if base := cfg.BasePrice[r]; base > 0 {
		raise(&s.BestSellPercent, string(r), proceeds*100/(qty*base))
	}
	if basis > 0 {
		raise(&s.BestCostPercent, string(r), proceeds*100/basis)
	}
	for _, e := range g.Events {
		bump(&s.EventSales, eventSalesKey(e.Key, r), qty)
	}
}

// RecordDayFacts updates the end-of-day goal facts. before is the game just before
// EndDay, g the game after it, and report what EndDay returned. It never changes
// anything the rules read.
func RecordDayFacts(before Game, g *Game, cfg Config, report DayReport) {
	s := &g.Goals
	parts := NetWorthBreakdown(*g, cfg)
	liquid := parts.Cash + parts.Stock

	opening := s.LastClosingNetWorth
	if s.DaysClosed == 0 {
		// First day tracked: the net worth before the night is the best baseline known.
		opening = NetWorth(before, cfg)
	}
	s.LastDayProfit = parts.Total - opening
	s.BestDayProfit = max(s.BestDayProfit, s.LastDayProfit)
	s.LastClosingNetWorth = parts.Total
	if s.DaysClosed == 0 || liquid < s.LowestClosingLiquid {
		s.LowestClosingLiquid = liquid
	}
	s.DaysClosed++

	s.MostProducedInADay = max(s.MostProducedInADay, report.Produced)
	if capacity := ProductionCapacity(before, cfg); capacity > 0 && report.Produced >= capacity {
		s.FullProductionStreak++
	} else {
		s.FullProductionStreak = 0
	}
	s.LongestFullProduction = max(s.LongestFullProduction, s.FullProductionStreak)

	if s.LastTradeDay != report.Day {
		s.IdleStreak++
	} else {
		s.IdleStreak = 0
	}
	s.LongestIdle = max(s.LongestIdle, s.IdleStreak)

	s.noteEvents(before.Events)
	s.noteEvents(g.Events)
	s.notePriceWar(*g)
}

func (s *GoalStats) noteEvents(events []ActiveEvent) {
	for _, e := range events {
		if !slices.Contains(s.EventsSeen, e.Key) {
			s.EventsSeen = append(s.EventsSeen, e.Key)
		}
	}
	sort.Strings(s.EventsSeen)
}

// notePriceWar records how a price war ends. It is judged at closing: the war is over on
// the first day it is no longer active, and won if the total share held is not lower than
// at the closing where it was first seen.
func (s *GoalStats) notePriceWar(g Game) {
	share := 0.0
	for _, t := range g.Territories {
		if t.Entered {
			share += t.Share
		}
	}
	switch {
	case priceWarActive(g) && !s.WarTracking:
		s.WarTracking, s.WarStartShare = true, share
	case !priceWarActive(g) && s.WarTracking:
		s.WarTracking = false
		if share >= s.WarStartShare {
			s.PriceWarsWon++
		}
	}
}
