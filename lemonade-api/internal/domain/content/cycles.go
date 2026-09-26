package content

// Economic cycles are long regimes: at most one is active at a time, and a new one
// may start on any day none is (late game Empire B). Effects are percentages.

// Cycle keys.
const (
	CycleBoom      = "boom"
	CycleRecession = "recession"
	CycleInflation = "inflation"
)

// CycleDef is one regime.
type CycleDef struct {
	Key  string
	Name string
	// MinDays and MaxDays bound how long it lasts (equal for a fixed length).
	MinDays, MaxDays int
	// DepthPct changes every commodity's free depth; ValuationPct every rival's buyout
	// value; DriftPct moves every price by that much a day while it lasts; UpkeepPct
	// raises the daily upkeep.
	DepthPct, ValuationPct, DriftPct, UpkeepPct float64
	Text                                        string
}

// Cycles is the table (late game content, section I). The stable state has no row.
var Cycles = []CycleDef{
	{Key: CycleBoom, Name: "Boom", MinDays: 15, MaxDays: 25, DepthPct: 15, ValuationPct: 20,
		Text: "Customers are spending: markets absorb 15% more, and rivals ask 20% more."},
	{Key: CycleRecession, Name: "Recession", MinDays: 15, MaxDays: 25, DepthPct: -15, ValuationPct: -25,
		Text: "Belts are tight: markets absorb 15% less, and rivals sell 25% cheaper."},
	{Key: CycleInflation, Name: "Inflation", MinDays: 20, MaxDays: 20, DriftPct: 1, UpkeepPct: 10,
		Text: "Every price drifts up 1% a day and upkeep is 10% higher. Prices ease back after."},
}

// CycleStartChance is the daily chance a regime starts when none is active.
const CycleStartChance = 0.03
