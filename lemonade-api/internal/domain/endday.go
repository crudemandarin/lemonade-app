package domain

import (
	"math/rand"

	"lemonade-api/internal/domain/content"
)

// EndDay runs the single atomic end-of-day transition (SPEC rules 12-17). The steps
// run in a fixed order, one named function each, so the late game tracks can fill
// in their steps without renegotiating the order (late game roadmap section 3):
//
//  1. managers act [Upgrades B]
//  2. produce
//  3. freezer rotation [Upgrades A]
//  4. ice melt and fresh-goods spoilage
//  5. settle upkeep
//  6. record the day, then the bankruptcy check
//  7. advance the day, forget pressure
//  8. events, then economic cycles [Empire B]
//  9. market tick for every commodity
//  10. rivals tick [Empire A]
//  11. contract deadlines [Products C]
//  12. price log
//
// Events come before the market tick because they share one random stream
// (SaltMarket) and existing seeds must replay exactly. Randomness is derived from
// the game's seed and the new day number, so the same seed and actions always give
// the same game.
func EndDay(g *Game, cfg Config) (DayReport, error) {
	if g.Status != StatusActive {
		return DayReport{}, ErrGameOver
	}

	report := DayReport{Day: g.Day, CapitalBefore: g.Capital}

	stepManagers(g, cfg, &report)
	stepProduce(g, cfg, &report)
	stepFreezerRotation(g, cfg, &report)
	stepMeltAndSpoil(g, cfg, &report)
	insolvent := stepUpkeep(g, cfg, &report)
	if stepRecordAndCheckBankruptcy(g, &report, insolvent) {
		return report, nil
	}

	before := effectivePrices(*g, cfg)
	stepAdvance(g, cfg)
	rng := dayRNG(g.Seed, g.Day, SaltMarket)
	stepEvents(g, cfg, rng, &report)
	stepMarketTick(g, cfg, rng, before, &report)
	stepRivals(g, cfg, &report)
	stepContracts(g, cfg, &report)
	stepPriceLog(g, cfg)

	report.CapitalAfter = g.Capital
	return report, nil
}

// Step 1: managers act before anything else happens overnight. Owned by Upgrades B.
func stepManagers(g *Game, cfg Config, report *DayReport) {}

// Step 2: production converts inputs to products. An ice machine tops up the ice first.
func stepProduce(g *Game, cfg Config, report *DayReport) {
	report.IceMade, report.IceMadeCost = makeInputs(g, cfg)
	report.Produced = produce(g, cfg)
	g.Stats.Produced += report.Produced
}

// Step 3: a freezer keeps some of tonight's fresh ice for tomorrow, up to its capacity.
// Ice is tracked in two age buckets: IceOld (one night old) and the rest (fresh). Production
// used the old ice first, so whatever old ice is left melts, and only fresh ice moves
// to the old bucket. Without a freezer nothing is kept and all ice melts, as before.
func stepFreezerRotation(g *Game, cfg Config, report *DayReport) {
	fresh := max(0, g.Inventory[Ice]-g.IceOld)
	kept := min(IceKeep(*g, cfg), fresh)
	g.IceOld = kept
	report.IceKept = kept
}

// Step 4: goods that melt nightly are gone; perishables spoil (Products B).
func stepMeltAndSpoil(g *Game, cfg Config, report *DayReport) {
	for _, c := range cfg.Commodities {
		if c.ShelfLifeDays != content.MeltsNightly {
			continue
		}
		n := g.Inventory[c.Key]
		kept := 0
		if c.Key == Ice {
			kept = g.IceOld // in the freezer: it stays, and stays "old"
			n -= kept
			report.IceMelted += n
		}
		g.removeStock(c.Key, n)
		if c.Key == Ice {
			g.IceOld = kept
		}
	}
}

// Step 5: pay upkeep, selling stock to cover it if cash is short. Hubs (Empire A)
// and upgrades and managers (Upgrades) add to TotalUpkeep.
func stepUpkeep(g *Game, cfg Config, report *DayReport) (insolvent bool) {
	paid, soldCases, soldProceeds, insolvent := settleUpkeep(g, cfg)
	report.UpkeepPaid = paid
	report.UpgradeUpkeep = UpgradeUpkeep(*g, cfg)
	report.ForcedSaleCases, report.ForcedSaleProceeds = soldCases, soldProceeds
	g.Stats.UpkeepPaid += paid
	if HasFeature(*g, cfg, "pnl") {
		p := pnlFor(*g, *report)
		report.Pnl = &p
	}
	return insolvent
}

// Step 6: record the day on the timeline, then end the game if upkeep could not be
// paid. It reports whether the game ended.
func stepRecordAndCheckBankruptcy(g *Game, report *DayReport, insolvent bool) bool {
	g.record(TimelinePoint{Day: g.Day, Kind: PointEndDay, Amount: report.UpkeepPaid, Produced: report.Produced})
	if !insolvent {
		return false
	}
	// The game ends on the day it was lost: no day advance, market tick or new
	// events, so the final state shows the prices the player last traded at.
	g.Status = StatusBankrupt
	report.Bankrupt = true
	report.CapitalAfter = g.Capital
	return true
}

// Step 7: the new day starts and the market forgets part of yesterday's trading.
func stepAdvance(g *Game, cfg Config) {
	g.Day++
	g.forgetPressure(cfg)
}

// Step 8: events expire and may spawn; economic cycles follow (Empire B).
func stepEvents(g *Game, cfg Config, rng *rand.Rand, report *DayReport) {
	report.ExpiredEvents, report.NewEvents = tickEvents(g, rng, cfg)
}

// effectivePrices is every commodity's effective price right now.
func effectivePrices(g Game, cfg Config) map[Resource]int {
	out := make(map[Resource]int, len(cfg.Commodities))
	for _, r := range cfg.Resources() {
		out[r] = effectivePriceFor(g, cfg, g.Market[r].Price, r)
	}
	return out
}

// Step 9: every commodity's price takes one step of its walk, in catalog order.
func stepMarketTick(g *Game, cfg Config, rng *rand.Rand, before map[Resource]int, report *DayReport) {
	for _, r := range cfg.Resources() {
		m := g.Market[r]
		m.Price = walk(rng, m.Price, cfg.BasePrice[r], cfg)
		after := effectivePriceFor(*g, cfg, m.Price, r)

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
}

// Step 10: rivals fight for share. Owned by Empire A.
func stepRivals(g *Game, cfg Config, report *DayReport) {}

// Step 11: contracts past their deadline settle. Owned by Products C.
func stepContracts(g *Game, cfg Config, report *DayReport) {}

// Step 12: today's prices go on the price chart.
func stepPriceLog(g *Game, cfg Config) {
	g.logPrices(cfg)
}

// Limiting factors reported by produceQty.
const (
	LimitProduction = "production"
	LimitSpace      = "space"
)

// produceQty is how many batches of the main recipe a day's production makes, and
// what limited it: daily capacity, stock of each input (named by resource), or free
// space for the output. On a tie the more actionable factor wins, so production is
// named only when nothing else binds. LimitedBy is empty when the game has no
// production capacity at all. Shared by produce and PreviewEndDay so the preview
// cannot drift from what EndDay does.
func produceQty(g Game, cfg Config) (qty int, limitedBy string) {
	return produceQtyIgnoring(g, cfg, "")
}

// produceQtyIgnoring is produceQty as if the stock of one input (skip) were unlimited;
// the ice machine uses it to see how much ice a night's production could use.
func produceQtyIgnoring(g Game, cfg Config, skip Resource) (qty int, limitedBy string) {
	rec := cfg.MainRecipe()
	capacity := ProductionCapacity(g, cfg) / rec.OutputQty
	if capacity <= 0 {
		return 0, ""
	}
	first := true
	for _, in := range rec.Inputs {
		if in.Resource == skip {
			continue
		}
		if n := g.Inventory[in.Resource] / in.Qty; first || n < qty {
			qty, limitedBy, first = n, string(in.Resource), false
		}
	}
	if free := (Capacity(g, cfg, rec.Output) - g.Inventory[rec.Output]) / rec.OutputQty; first || free < qty {
		qty, limitedBy, first = free, LimitSpace, false
	}
	if capacity < qty {
		qty, limitedBy = capacity, LimitProduction
	}
	if qty < 0 {
		qty = 0
	}
	return qty, limitedBy
}

// makeInputs runs the "make" upgrades (the ice machine): each tops up a commodity to what
// tonight's production could use, up to its nightly limit, paying its unit cost from
// cash. It returns the cases made and what they cost.
func makeInputs(g *Game, cfg Config) (made, cost int) {
	for _, e := range effectsOf(*g, cfg, content.EffMake) {
		r := Resource(e.Target)
		batches, _ := produceQtyIgnoring(*g, cfg, r)
		n := batches*recipeUses(cfg.MainRecipe(), r) - g.Inventory[r]
		n = min(n, int(e.Value), Capacity(*g, cfg, r)-g.Inventory[r])
		if unit := int(e.Aux); unit > 0 {
			n = min(n, g.Capital/unit)
		}
		if n <= 0 {
			continue
		}
		spent := n * int(e.Aux)
		g.Capital -= spent
		g.Inventory[r] += n
		g.addBasis(r, spent)
		made += n
		cost += spent
	}
	return made, cost
}

// produce runs the main recipe (lemonade: 1 lemon + 1 sugar + 1 ice + 1 cup) and
// returns the cases made. The output costs what its inputs cost. Use discounts save whole
// cases of an input, and yield bonuses add whole cases of output; both carry fractions.
func produce(g *Game, cfg Config) int {
	n, _ := produceQty(*g, cfg)
	rec := cfg.MainRecipe()
	cost := 0
	for _, in := range rec.Inputs {
		use := n * in.Qty
		cost += g.removeStock(in.Resource, use-g.takeSavings(cfg, in.Resource, use))
	}
	out := n * rec.OutputQty
	out += g.yieldExtra(cfg, rec.Key, out, Capacity(*g, cfg, rec.Output)-g.Inventory[rec.Output]-out)
	g.Inventory[rec.Output] += out
	g.addBasis(rec.Output, cost)
	return out
}

// recipeUses is how many cases of r one batch of rec consumes.
func recipeUses(rec Recipe, r Resource) int {
	for _, in := range rec.Inputs {
		if in.Resource == r {
			return in.Qty
		}
	}
	return 0
}

// Projection is what End day would do to the current state, without doing it.
type Projection struct {
	LemonadeToProduce int
	// IceToMelt is the ice left over after production has used its share and the
	// freezer has kept what it can; IceKept is that kept ice.
	IceToMelt int
	IceKept   int
	LimitedBy string
}

// PreviewEndDay reports the production and ice melt EndDay would give now. It runs the
// same steps on a copy of the parts they touch, so it cannot drift from EndDay. It does
// not modify the game.
func PreviewEndDay(g Game, cfg Config) Projection {
	c := g
	c.Inventory = make(map[Resource]int, len(g.Inventory))
	for k, v := range g.Inventory {
		c.Inventory[k] = v
	}
	c.CostBasis = make(map[Resource]int, len(g.CostBasis))
	for k, v := range g.CostBasis {
		c.CostBasis[k] = v
	}
	c.Carry = make(map[string]float64, len(g.Carry))
	for k, v := range g.Carry {
		c.Carry[k] = v
	}
	var report DayReport
	makeInputs(&c, cfg)
	_, limitedBy := produceQty(c, cfg)
	report.Produced = produce(&c, cfg)
	stepFreezerRotation(&c, cfg, &report)
	stepMeltAndSpoil(&c, cfg, &report)
	return Projection{
		LemonadeToProduce: report.Produced,
		IceToMelt:         report.IceMelted,
		IceKept:           report.IceKept,
		LimitedBy:         limitedBy,
	}
}
