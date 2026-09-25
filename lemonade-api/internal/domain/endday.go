package domain

import "math/rand"

// EndDay runs the single atomic end-of-day transition (SPEC rules 12-17):
// produce, melt ice, pay upkeep, advance the day, tick the market and events,
// then check bankruptcy. Randomness is derived from the game's seed and the new
// day number, so the same seed and actions always give the same game.
func EndDay(g *Game, cfg Config) (DayReport, error) {
	if g.Status != StatusActive {
		return DayReport{}, ErrGameOver
	}

	report := DayReport{Day: g.Day, CapitalBefore: g.Capital}

	report.Produced = produce(g, cfg)

	report.IceMelted = g.Inventory[Ice]
	g.Inventory[Ice] = 0

	paid, soldCases, soldProceeds, insolvent := settleUpkeep(g, cfg)
	report.UpkeepPaid = paid
	report.ForcedSaleCases, report.ForcedSaleProceeds = soldCases, soldProceeds
	g.Stats.Produced += report.Produced
	g.Stats.UpkeepPaid += paid
	g.record(TimelinePoint{Day: g.Day, Kind: PointEndDay, Amount: paid, Produced: report.Produced})

	if insolvent {
		// The game ends on the day it was lost: no day advance, market tick or new
		// events, so the final state shows the prices the player last traded at.
		g.Status = StatusBankrupt
		report.Bankrupt = true
		report.CapitalAfter = g.Capital
		return report, nil
	}

	before := make(map[Resource]int, len(Resources))
	for _, r := range Resources {
		before[r] = effectivePrice(g.Market[r].Price, g.Events, r)
	}

	g.Day++
	rng := rand.New(rand.NewSource(g.Seed ^ int64(g.Day)))

	report.ExpiredEvents, report.NewEvents = tickEvents(g, rng, cfg)

	for _, r := range Resources {
		m := g.Market[r]
		m.Price = walk(rng, m.Price, cfg.BasePrice[r], cfg)
		after := effectivePrice(m.Price, g.Events, r)

		// Every resource is reported, changed or not, so the report can always show it.
		report.PriceChanges = append(report.PriceChanges, PriceChange{
			Resource: r,
			Before:   before[r],
			After:    after,
		})

		prev := before[r]
		m.PreviousEffective = &prev
		m.History = append(m.History, after)
		if len(m.History) > cfg.HistoryLength {
			m.History = m.History[len(m.History)-cfg.HistoryLength:]
		}
	}

	report.CapitalAfter = g.Capital

	return report, nil
}

// Limiting factors reported by produceQty.
const (
	LimitProduction = "production"
	LimitSpace      = "space"
)

// produceQty is how many lemonade a day's production makes, and what limited it:
// daily capacity, stock of each input (named by resource), or free lemonade
// warehouse space. On a tie the more actionable factor wins, so production is
// named only when nothing else binds. LimitedBy is empty when the game has no
// production capacity at all. Shared by produce and PreviewEndDay so the
// preview cannot drift from what EndDay does.
func produceQty(g Game, cfg Config) (qty int, limitedBy string) {
	capacity := ProductionCapacity(g, cfg)
	if capacity <= 0 {
		return 0, ""
	}
	first := true
	for _, in := range Inputs {
		if first || g.Inventory[in] < qty {
			qty, limitedBy, first = g.Inventory[in], string(in), false
		}
	}
	if free := Capacity(g, cfg, Lemonade) - g.Inventory[Lemonade]; free < qty {
		qty, limitedBy = free, LimitSpace
	}
	if capacity < qty {
		qty, limitedBy = capacity, LimitProduction
	}
	if qty < 0 {
		qty = 0
	}
	return qty, limitedBy
}

// produce converts inputs to lemonade: 1 lemon + 1 sugar + 1 ice + 1 cup -> 1 lemonade.
func produce(g *Game, cfg Config) int {
	n, _ := produceQty(*g, cfg)
	for _, in := range Inputs {
		g.Inventory[in] -= n
	}
	g.Inventory[Lemonade] += n
	return n
}

// Projection is what End day would do to the current state, without doing it.
type Projection struct {
	LemonadeToProduce int
	// IceToMelt is the ice left over after production has used its share.
	IceToMelt int
	LimitedBy string
}

// PreviewEndDay reports the production and ice melt EndDay would give now.
// It does not modify the game.
func PreviewEndDay(g Game, cfg Config) Projection {
	qty, limitedBy := produceQty(g, cfg)
	return Projection{
		LemonadeToProduce: qty,
		IceToMelt:         g.Inventory[Ice] - qty,
		LimitedBy:         limitedBy,
	}
}
