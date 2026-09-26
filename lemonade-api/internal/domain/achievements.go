package domain

import (
	"fmt"
	"slices"

	"lemonade-api/internal/domain/content"
)

// Achievements are cosmetic: they are evaluated after a mutation has run and never
// feed back into the game. The table is content.Achievements; each row's condition is
// a content.Predicate from a small closed set, interpreted here.

// AchievementContext is what the API layer loads beyond the game: the player's
// history across runs.
type AchievementContext struct {
	// Unlocked are the keys the player already has; Evaluate never returns them.
	Unlocked map[string]bool
	// RunsFinished counts the player's finished runs, including one this mutation ended.
	RunsFinished int
	// PreviousBest is the best score before this mutation (nil with no finished run).
	PreviousBest *int
	// FinishedScore is the score of the run this mutation ended (nil if none ended).
	FinishedScore *int
	// BoardRank is the player's rank on the global board once the run that just ended
	// counts (0 when no run ended).
	BoardRank int
}

// Evaluate returns the keys of the achievements that hold after a mutation and are
// not unlocked yet, each once, in table order. before is the game before the
// mutation and after the game once it succeeded.
func Evaluate(defs []content.AchievementDef, before, after Game, cfg Config, ctx AchievementContext) []string {
	f := &facts{before: before, after: after, cfg: cfg, ctx: ctx}
	var out []string
	for _, d := range defs {
		if ctx.Unlocked[d.Key] || slices.Contains(out, d.Key) {
			continue
		}
		if f.holds(d.Check) {
			out = append(out, d.Key)
		}
	}
	return out
}

// facts answers predicates about one mutation, computing net worth at most once.
type facts struct {
	before, after Game
	cfg           Config
	ctx           AchievementContext
	netWorth      *int
}

func (f *facts) worth() int {
	if f.netWorth == nil {
		nw := NetWorth(f.after, f.cfg)
		f.netWorth = &nw
	}
	return *f.netWorth
}

// dayEnded reports whether the mutation ended a day and the run survived it.
func (f *facts) dayEnded() bool { return f.after.Day > f.before.Day }

// wentBankrupt reports whether the mutation ended the run in bankruptcy.
func (f *facts) wentBankrupt() bool {
	return f.before.Status == StatusActive && f.after.Status == StatusBankrupt
}

func (f *facts) holds(p content.Predicate) bool {
	g, s := f.after, f.after.Goals
	switch p.Kind {
	case content.KindNetWorthAtLeast:
		return f.worth() >= p.N
	case content.KindDayAtLeast:
		return g.Day >= p.N
	case content.KindDayAtMost:
		return g.Day <= p.N
	case content.KindStatAtLeast:
		v, ok := statValue(g, p.Stat)
		return ok && v >= p.N
	case content.KindCashWithNoStock:
		return g.Capital >= p.N && totalStock(g) == 0
	case content.KindStockTotalAtLeast:
		return totalStock(g) >= p.N
	case content.KindStockAtLeast:
		return g.Inventory[Resource(p.Commodity)] >= p.N
	case content.KindAllWarehousesFull:
		return allWarehousesFull(g, f.cfg)
	case content.KindFacilityMaxed:
		return facilityMaxed(g, f.cfg, FacilityType(p.Facility))
	case content.KindUpgradesOwned:
		return len(ownedCatalogUpgrades(g)) >= p.N
	case content.KindRivalsBoughtAtLeast:
		return acquiredRivals(g) >= p.N
	case content.KindRivalBought:
		return g.Rivals[p.Key].Status == RivalAcquired
	case content.KindHostileBuyout:
		for _, r := range g.Rivals {
			if r.Status == RivalAcquired && r.Hostile {
				return true
			}
		}
		return false
	case content.KindTerritoryEntered:
		return g.Territories[p.Key].Entered
	case content.KindTerritoryShare:
		t := g.Territories[p.Key]
		return t.Entered && t.Share >= float64(p.N)
	case content.KindUpgradeSetOwned:
		return upgradeSetOwned(g, p.Key)
	case content.KindAllOf:
		for _, q := range p.All {
			if !f.holds(q) {
				return false
			}
		}
		return len(p.All) > 0
	case content.KindRunsFinishedAtLeast:
		return f.ctx.RunsFinished >= p.N
	case content.KindNewPersonalBest:
		return f.ctx.FinishedScore != nil && f.ctx.PreviousBest != nil && *f.ctx.FinishedScore > *f.ctx.PreviousBest
	case content.KindBoardRankAtMost:
		return f.ctx.BoardRank >= 1 && f.ctx.BoardRank <= p.N
	case content.KindBoughtInputAtPercent:
		return s.LowestInputBuyPercent > 0 && s.LowestInputBuyPercent <= p.N
	case content.KindSoldAtPercentOfBase:
		return s.BestSellPercent[p.Commodity] >= p.N
	case content.KindSoldAtPercentOfCost:
		return s.BestCostPercent[p.Commodity] >= p.N
	case content.KindSoldDuringEvent:
		return s.EventSales[eventSalesKey(p.Event, Resource(p.Commodity))] >= p.N
	case content.KindProfitDuringEvent:
		return f.dayEnded() && s.DaysClosed > 0 && s.LastDayProfit > 0 && eventActive(f.before.Events, p.Event)
	case content.KindEveryEventSeen:
		return len(f.cfg.Events) > 0 && seenCount(s, f.cfg) == len(f.cfg.Events)
	case content.KindSoldWithoutImpact:
		return s.SalesWithImpact == 0 && s.Sold[p.Commodity] >= p.N
	case content.KindClosingCashBetween:
		return f.dayEnded() && g.Status == StatusActive && g.Capital >= p.N && g.Capital <= p.M
	case content.KindComeback:
		return s.DaysClosed > 0 && s.LowestClosingLiquid < p.N && f.worth() >= p.M
	case content.KindBankruptHoldingOnly:
		return f.wentBankrupt() && holdsOnly(f.before, Resource(p.Commodity))
	case content.KindBankruptByDay:
		return g.Status == StatusBankrupt && g.Day <= p.N
	case content.KindGaveUpWithNetWorth:
		return g.Status == StatusGaveUp && f.worth() >= p.N
	}
	return false
}

// statValue reads one stat a StatAtLeast check names.
func statValue(g Game, stat string) (int, bool) {
	switch stat {
	case content.StatProduced:
		return g.Stats.Produced, true
	case content.StatFacilitiesBought:
		return g.Stats.FacilitiesBought, true
	case content.StatUpgrades:
		return g.Stats.Upgrades, true
	case content.StatFacilitiesSold:
		return g.Stats.FacilitiesSold, true
	case content.StatMostProducedInADay:
		return g.Goals.MostProducedInADay, true
	case content.StatLongestFullProduction:
		return g.Goals.LongestFullProduction, true
	case content.StatLongestIdle:
		return g.Goals.LongestIdle, true
	case content.StatTradesWithImpact:
		return g.Goals.TradesWithImpact, true
	case content.StatBestDayProfit:
		return g.Goals.BestDayProfit, true
	case content.StatPriceWarsWon:
		return g.Goals.PriceWarsWon, true
	}
	return 0, false
}

// ownedCatalogUpgrades lists the owned upgrades that are in the catalog's categories
// (territory upgrades are not, until Empire adds their category to the achievement).
func ownedCatalogUpgrades(g Game) []string {
	var out []string
	for _, u := range content.Upgrades {
		if g.Upgrades[u.Key] > 0 && inUpgradeCategories(u.Category) {
			out = append(out, u.Key)
		}
	}
	return out
}

func inUpgradeCategories(category string) bool {
	for _, c := range content.UpgradeCategories {
		if c == category {
			return true
		}
	}
	return false
}

// upgradeSetOwned reports whether the game owns every upgrade of the category ("" is
// every catalog upgrade).
func upgradeSetOwned(g Game, category string) bool {
	found := false
	for _, u := range content.Upgrades {
		if !inUpgradeCategories(u.Category) || (category != "" && u.Category != category) {
			continue
		}
		found = true
		if g.Upgrades[u.Key] == 0 {
			return false
		}
	}
	return found
}

func acquiredRivals(g Game) int {
	n := 0
	for _, r := range g.Rivals {
		if r.Status == RivalAcquired {
			n++
		}
	}
	return n
}

func totalStock(g Game) int {
	n := 0
	for _, q := range g.Inventory {
		n += q
	}
	return n
}

func allWarehousesFull(g Game, cfg Config) bool {
	for _, class := range StorageClasses(cfg) {
		c := ClassCapacity(g, cfg, class)
		if c <= 0 || ClassStock(g, cfg, class) < c {
			return false
		}
	}
	return true
}

func facilityMaxed(g Game, cfg Config, kind FacilityType) bool {
	switch kind {
	case Production:
		return g.ProductionLevel >= cfg.MaxLevel && g.ProductionQty >= cfg.MaxQuantity
	case Warehouse:
		if g.WarehouseLevel < cfg.MaxLevel {
			return false
		}
		for _, class := range StorageClasses(cfg) {
			if g.WarehouseQty[class] < cfg.MaxQuantity {
				return false
			}
		}
		return true
	}
	return false
}

func eventActive(events []ActiveEvent, key string) bool {
	for _, e := range events {
		if e.Key == key {
			return true
		}
	}
	return false
}

func seenCount(s GoalStats, cfg Config) int {
	n := 0
	for _, e := range cfg.Events {
		if slices.Contains(s.EventsSeen, e.Key) {
			n++
		}
	}
	return n
}

// holdsOnly reports whether the game holds some of r and nothing else.
func holdsOnly(g Game, r Resource) bool {
	if g.Inventory[r] <= 0 {
		return false
	}
	for k, q := range g.Inventory {
		if k != r && q > 0 {
			return false
		}
	}
	return true
}

// Progress is how far the game is toward a measurable predicate: current (capped at
// target) of target. ok is false for checks with no meaningful progress bar.
func Progress(p content.Predicate, g Game, cfg Config, ctx AchievementContext) (current, target int, ok bool) {
	s := g.Goals
	switch p.Kind {
	case content.KindNetWorthAtLeast:
		current, target, ok = NetWorth(g, cfg), p.N, true
	case content.KindDayAtLeast:
		current, target, ok = g.Day, p.N, true
	case content.KindStatAtLeast:
		current, ok = statValue(g, p.Stat)
		target = p.N
	case content.KindStockTotalAtLeast:
		current, target, ok = totalStock(g), p.N, true
	case content.KindStockAtLeast:
		current, target, ok = g.Inventory[Resource(p.Commodity)], p.N, true
	case content.KindRunsFinishedAtLeast:
		current, target, ok = ctx.RunsFinished, p.N, true
	case content.KindSoldDuringEvent:
		current, target, ok = s.EventSales[eventSalesKey(p.Event, Resource(p.Commodity))], p.N, true
	case content.KindEveryEventSeen:
		current, target, ok = seenCount(s, cfg), len(cfg.Events), true
	case content.KindUpgradesOwned:
		current, target, ok = len(ownedCatalogUpgrades(g)), p.N, true
	case content.KindSoldWithoutImpact:
		// Once a sale has paid impact this run, the count no longer leads anywhere.
		current, target, ok = s.Sold[p.Commodity], p.N, s.SalesWithImpact == 0
	}
	if !ok || target <= 0 {
		return 0, 0, false
	}
	return max(0, min(current, target)), target, true
}

// ValidatePredicate reports why a predicate cannot be evaluated against cfg, or nil.
func ValidatePredicate(p content.Predicate, cfg Config) error {
	positive := func() error {
		if p.N < 1 {
			return fmt.Errorf("%s needs N >= 1, got %d", p.Kind, p.N)
		}
		return nil
	}
	commodity := func() error {
		if !cfg.Valid(Resource(p.Commodity)) {
			return fmt.Errorf("%s: unknown commodity %q", p.Kind, p.Commodity)
		}
		return nil
	}
	event := func() error {
		for _, e := range cfg.Events {
			if e.Key == p.Event {
				return nil
			}
		}
		return fmt.Errorf("%s: unknown event %q", p.Kind, p.Event)
	}
	percent := func() error {
		if p.N < 1 || p.N > 1000 {
			return fmt.Errorf("%s: percent %d out of range", p.Kind, p.N)
		}
		return nil
	}

	switch p.Kind {
	case content.KindNetWorthAtLeast, content.KindDayAtLeast, content.KindDayAtMost,
		content.KindCashWithNoStock, content.KindStockTotalAtLeast, content.KindRunsFinishedAtLeast,
		content.KindBoardRankAtMost, content.KindBankruptByDay, content.KindGaveUpWithNetWorth:
		return positive()
	case content.KindStatAtLeast:
		if _, ok := statValue(Game{}, p.Stat); !ok {
			return fmt.Errorf("unknown stat %q", p.Stat)
		}
		return positive()
	case content.KindStockAtLeast, content.KindSoldWithoutImpact:
		if err := commodity(); err != nil {
			return err
		}
		return positive()
	case content.KindAllWarehousesFull, content.KindEveryEventSeen, content.KindNewPersonalBest:
		return nil
	case content.KindUpgradesOwned:
		return positive()
	case content.KindRivalsBoughtAtLeast:
		return positive()
	case content.KindHostileBuyout:
		return nil
	case content.KindRivalBought:
		for _, d := range cfg.Rivals {
			if d.Key == p.Key {
				return nil
			}
		}
		return fmt.Errorf("%s: unknown rival %q", p.Kind, p.Key)
	case content.KindTerritoryEntered, content.KindTerritoryShare:
		for _, d := range cfg.Territories {
			if d.Key == p.Key {
				if p.Kind == content.KindTerritoryShare {
					return percent()
				}
				return nil
			}
		}
		return fmt.Errorf("%s: unknown territory %q", p.Kind, p.Key)
	case content.KindUpgradeSetOwned:
		if p.Key != "" && !inUpgradeCategories(p.Key) {
			return fmt.Errorf("%s: unknown category %q", p.Kind, p.Key)
		}
		return nil
	case content.KindFacilityMaxed:
		if !FacilityType(p.Facility).Valid() {
			return fmt.Errorf("unknown facility %q", p.Facility)
		}
		return nil
	case content.KindAllOf:
		if len(p.All) == 0 {
			return fmt.Errorf("all_of needs at least one check")
		}
		for _, q := range p.All {
			if err := ValidatePredicate(q, cfg); err != nil {
				return err
			}
		}
		return nil
	case content.KindBoughtInputAtPercent:
		return percent()
	case content.KindSoldAtPercentOfBase, content.KindSoldAtPercentOfCost:
		if err := commodity(); err != nil {
			return err
		}
		return percent()
	case content.KindSoldDuringEvent:
		if err := commodity(); err != nil {
			return err
		}
		if err := event(); err != nil {
			return err
		}
		return positive()
	case content.KindProfitDuringEvent:
		return event()
	case content.KindClosingCashBetween:
		if p.N < 0 || p.M < p.N {
			return fmt.Errorf("closing_cash_between needs 0 <= min <= max, got %d..%d", p.N, p.M)
		}
		return nil
	case content.KindComeback:
		if p.N < 1 || p.M <= p.N {
			return fmt.Errorf("comeback needs 0 < low < high, got %d, %d", p.N, p.M)
		}
		return nil
	case content.KindBankruptHoldingOnly:
		return commodity()
	}
	return fmt.Errorf("unknown predicate kind %q", p.Kind)
}

// Proof is one achievement a finished run proves: Key, first proven by runs[Run].
type Proof struct {
	Key string
	Run int
}

// ProvenByRuns is the retroactive grant: the achievements a player's stored finished
// runs (oldest first) prove, each with the first run that proves it. boardRank is the
// player's current rank on the global board (0 if unranked); it is credited to their
// best run. A run keeps only its final state and totals, so only checks that those
// settle count: net worth (the final value, or the cash peak, which net worth is never
// below), days, the totals in Stats, the run count, a new best, the board rank, and how
// the run ended. Everything that needs the moment-to-moment game is skipped.
func ProvenByRuns(defs []content.AchievementDef, runs []RunRecord, boardRank int) []Proof {
	best := -1
	for i, r := range runs {
		if best < 0 || r.Score > runs[best].Score {
			best = i
		}
	}
	var out []Proof
	for _, d := range defs {
		for i := range runs {
			if provenBy(d.Check, runs, i, best, boardRank, false) {
				out = append(out, Proof{Key: d.Key, Run: i})
				break
			}
		}
	}
	return out
}

// provenBy reports whether runs[i] proves p. atEnd restricts it to the run's final
// state, which AllOf needs so that every part is true at the same moment.
func provenBy(p content.Predicate, runs []RunRecord, i, best, boardRank int, atEnd bool) bool {
	r := runs[i]
	switch p.Kind {
	case content.KindNetWorthAtLeast:
		if atEnd {
			return r.NetWorth >= p.N
		}
		return max(r.NetWorth, r.Stats.PeakCapital) >= p.N
	case content.KindDayAtLeast:
		return r.Days >= p.N
	case content.KindDayAtMost:
		return atEnd && r.Days <= p.N
	case content.KindStatAtLeast:
		switch p.Stat {
		case content.StatProduced:
			return r.Stats.Produced >= p.N
		case content.StatFacilitiesBought:
			return r.Stats.FacilitiesBought >= p.N
		case content.StatUpgrades:
			return r.Stats.Upgrades >= p.N
		case content.StatFacilitiesSold:
			return r.Stats.FacilitiesSold >= p.N
		}
		return false
	case content.KindRunsFinishedAtLeast:
		return i+1 >= p.N
	case content.KindNewPersonalBest:
		for j := 0; j < i; j++ {
			if runs[j].Score >= r.Score {
				return false
			}
		}
		return i > 0
	case content.KindBoardRankAtMost:
		return i == best && boardRank >= 1 && boardRank <= p.N
	case content.KindBankruptByDay:
		return r.EndedBy == EndedByBankrupt && r.Days <= p.N
	case content.KindGaveUpWithNetWorth:
		return r.EndedBy == EndedByGaveUp && r.NetWorth >= p.N
	case content.KindAllOf:
		for _, q := range p.All {
			if !provenBy(q, runs, i, best, boardRank, true) {
				return false
			}
		}
		return len(p.All) > 0
	}
	return false
}
