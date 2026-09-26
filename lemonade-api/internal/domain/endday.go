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
	results := produce(g, cfg)
	report.Produced = producedTotal(results)
	for _, r := range results {
		if r.Output > 0 {
			report.Made = append(report.Made, MadeLine{Recipe: r.Recipe.Key, Output: r.Recipe.Output, Cases: r.Output})
		}
	}
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

// Step 4: goods that melt nightly are gone; perishables spoil (see spoilage.go).
func stepMeltAndSpoil(g *Game, cfg Config, report *DayReport) {
	spoilPerishables(g, cfg, report)
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
	// Rival moves announced a day ago happen now; they draw no random numbers.
	fireRivalEvents(g, cfg)
	stepCycles(g, cfg, report)
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

		// Every unlocked resource is reported, changed or not, so the report can always
		// show it. Goods the player has no use for yet still walk, but stay out of it.
		if CommodityUnlocked(*g, cfg, r) {
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
}

// Step 11: contracts past their deadline settle. Owned by Products C.
func stepContracts(g *Game, cfg Config, report *DayReport) {}

// Step 12: today's prices go on the price chart.
func stepPriceLog(g *Game, cfg Config) {
	g.logPrices(cfg)
}

// Limiting factors a plan row reports besides an input's key.
const (
	LimitProduction = "production"
	LimitSpace      = "space"
)

// makeInputs runs the "make" upgrades (the ice machine): each tops up a commodity to what
// tonight's production plan could use, up to its nightly limit, paying its unit cost from
// cash. It returns the cases made and what they cost.
func makeInputs(g *Game, cfg Config) (made, cost int) {
	for _, e := range effectsOf(*g, cfg, content.EffMake) {
		r := Resource(e.Target)
		need := 0
		for _, row := range planBatches(*g, cfg, r) {
			need += row.Batches * recipeUses(row.Recipe, r)
		}
		n := need - g.Inventory[r]
		n = min(n, int(e.Value), FreeSpace(*g, cfg, r))
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

// produce runs the production plan (see plan.go) and returns what each row made. A row's
// output costs what its inputs cost. Use discounts save whole cases of an input, and yield
// bonuses add whole cases of output; both carry fractions.
func produce(g *Game, cfg Config) []PlanResult {
	results := planBatches(*g, cfg, "")
	for i := range results {
		res := &results[i]
		if res.Batches <= 0 {
			continue
		}
		rec := res.Recipe
		cost := 0
		for _, in := range rec.Inputs {
			use := res.Batches * in.Qty
			cost += g.removeStock(in.Resource, use-g.takeSavings(cfg, in.Resource, use))
		}
		out := res.Batches * rec.OutputQty
		out += g.yieldExtra(cfg, rec.Key, out, FreeSpace(*g, cfg, rec.Output)-out)
		g.Inventory[rec.Output] += out
		g.addBasis(rec.Output, cost)
		res.Output = out
	}
	return results
}

// producedTotal is the cases of output across a night's plan results.
func producedTotal(results []PlanResult) int {
	n := 0
	for _, r := range results {
		n += r.Output
	}
	return n
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
	// Plan is one line per row of the production plan: what it would make and what limits it.
	Plan []PlanProjection
	// WillSpoil is the cases of each perishable that go off tonight.
	WillSpoil map[Resource]int
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
	c.Aged = make(map[Resource][]int, len(g.Aged))
	for k, v := range g.Aged {
		c.Aged[k] = append([]int(nil), v...)
	}
	var report DayReport
	makeInputs(&c, cfg)
	var plan []PlanProjection
	limitedBy := ""
	for i, row := range planBatches(c, cfg, "") {
		if i == 0 {
			limitedBy = row.LimitedBy
		}
		plan = append(plan, PlanProjection{Recipe: row.Recipe.Key, Name: row.Recipe.Name, Output: row.Recipe.Output, Cases: row.Output, LimitedBy: row.LimitedBy})
	}
	results := produce(&c, cfg)
	report.Produced = producedTotal(results)
	for i := range results {
		if i < len(plan) {
			plan[i].Cases = results[i].Output
		}
	}
	lemonade := 0
	for _, r := range results {
		if r.Recipe.Output == cfg.MainRecipe().Output {
			lemonade += r.Output
		}
	}
	stepFreezerRotation(&c, cfg, &report)
	stepMeltAndSpoil(&c, cfg, &report)
	return Projection{
		LemonadeToProduce: lemonade,
		Plan:              plan,
		WillSpoil:         report.Spoiled,
		IceToMelt:         report.IceMelted,
		IceKept:           report.IceKept,
		LimitedBy:         limitedBy,
	}
}
