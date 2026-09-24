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

		if after != before[r] {
			report.PriceChanges = append(report.PriceChanges, PriceChange{
				Resource: r,
				Before:   before[r],
				After:    after,
			})
		}

		prev := before[r]
		m.PreviousEffective = &prev
		m.History = append(m.History, after)
		if len(m.History) > cfg.HistoryLength {
			m.History = m.History[len(m.History)-cfg.HistoryLength:]
		}
	}

	if insolvent {
		g.Status = StatusBankrupt
	}
	report.Bankrupt = g.Status == StatusBankrupt
	report.CapitalAfter = g.Capital

	return report, nil
}

// produce converts inputs to lemonade: 1 lemon + 1 sugar + 1 ice + 1 cup -> 1 lemonade,
// limited by daily capacity, stock of each input, and free lemonade warehouse space.
func produce(g *Game, cfg Config) int {
	n := ProductionCapacity(*g, cfg)
	for _, in := range Inputs {
		if g.Inventory[in] < n {
			n = g.Inventory[in]
		}
	}
	if free := Capacity(*g, cfg, Lemonade) - g.Inventory[Lemonade]; free < n {
		n = free
	}
	if n < 0 {
		n = 0
	}

	for _, in := range Inputs {
		g.Inventory[in] -= n
	}
	g.Inventory[Lemonade] += n
	return n
}
