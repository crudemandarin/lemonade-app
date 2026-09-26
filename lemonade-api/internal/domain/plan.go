package domain

// The production plan (late game Products B): an ordered list of recipes with an optional
// daily target in cases of output (0 means as much as possible). Each night production
// works down the list, and every row draws on what is left of the shared daily capacity,
// the input stock and the free space for its output. A game with no plan makes lemonade,
// as much as possible: the base game.

// PlanRow is one line of the plan.
type PlanRow struct {
	Recipe string
	// Target is the most cases of the output to make in a night; 0 means no limit.
	Target int
}

// Limiting factors a plan row can report besides an input's key: see LimitProduction and
// LimitSpace. LimitTarget means the row reached its own target.
const LimitTarget = "target"

// PlanResult is what one plan row does tonight.
type PlanResult struct {
	Recipe Recipe
	// Batches run and Output cases made (before yield bonuses), and what limited the row.
	Batches   int
	Output    int
	LimitedBy string
}

// EffectivePlan is the plan production follows: the stored one, or lemonade alone.
// Rows for recipes the player does not know are skipped, so the plan cannot outlive one.
func EffectivePlan(g Game, cfg Config) []PlanRow {
	var out []PlanRow
	for _, row := range g.ProductionPlan {
		if rec, ok := cfg.LookupRecipe(row.Recipe); ok && RecipeKnown(g, rec) {
			out = append(out, row)
		}
	}
	if len(g.ProductionPlan) == 0 {
		return []PlanRow{{Recipe: cfg.MainRecipe().Key}}
	}
	return out
}

// planBatches works out each plan row's batches for tonight on the game as it is now,
// without changing it. skip names an input to treat as unlimited (the ice machine uses
// it to see how much ice tonight's plan could use). Rows share capacity in order, and
// stock and space are shared too: a row that uses lemons leaves fewer for the next.
func planBatches(g Game, cfg Config, skip Resource) []PlanResult {
	remaining := ProductionCapacity(g, cfg)
	stock := make(map[Resource]int, len(g.Inventory))
	for r, n := range g.Inventory {
		stock[r] = n
	}
	space := make(map[string]int)
	freeIn := func(r Resource) int {
		class := ClassOf(cfg, r)
		if _, ok := space[class]; !ok {
			space[class] = FreeSpace(g, cfg, r)
		}
		return space[class]
	}

	var out []PlanResult
	for _, row := range EffectivePlan(g, cfg) {
		rec, _ := cfg.LookupRecipe(row.Recipe)
		res := PlanResult{Recipe: rec}
		if ProductionCapacity(g, cfg) <= 0 {
			// No production at all: nothing is made and nothing is to blame.
			out = append(out, res)
			continue
		}
		capacity := remaining / rec.OutputQty
		batches, limited, first := 0, "", true
		for _, in := range rec.Inputs {
			if in.Resource == skip {
				continue
			}
			if n := stock[in.Resource] / in.Qty; first || n < batches {
				batches, limited, first = n, string(in.Resource), false
			}
		}
		if free := freeIn(rec.Output) / rec.OutputQty; first || free < batches {
			batches, limited = free, LimitSpace
		}
		if row.Target > 0 && row.Target/rec.OutputQty < batches {
			batches, limited = row.Target/rec.OutputQty, LimitTarget
		}
		// On a tie the more actionable factor wins, so production is named only when
		// nothing else binds.
		if capacity < batches {
			batches, limited = capacity, LimitProduction
		}
		batches = max(batches, 0)
		res.Batches, res.Output, res.LimitedBy = batches, batches*rec.OutputQty, limited
		if batches > 0 {
			remaining -= batches * rec.OutputQty
			for _, in := range rec.Inputs {
				stock[in.Resource] -= batches * in.Qty
			}
			space[ClassOf(cfg, rec.Output)] = freeIn(rec.Output) - res.Output
		}
		out = append(out, res)
	}
	return out
}

// PlanProjection is one row of the production plan as End day would run it.
type PlanProjection struct {
	Recipe    string
	Name      string
	Output    Resource
	Cases     int
	LimitedBy string
}

// MadeLine is what one recipe made in a night, for the day report.
type MadeLine struct {
	Recipe string
	Output Resource
	Cases  int
}

// Errors from setting the plan.
var (
	ErrPlanRecipe    = errorsNew("the plan names a recipe you cannot make")
	ErrPlanDuplicate = errorsNew("a recipe can appear once in the plan")
	ErrPlanTarget    = errorsNew("a target cannot be negative")
)

// MaxPlanRows caps the plan at the number of recipes there are.
func MaxPlanRows(cfg Config) int { return len(cfg.Recipes) }

// SetProductionPlan replaces the plan. An empty list restores the default (lemonade,
// as much as possible). Every recipe must be one the player has learned, and each may
// appear once.
func SetProductionPlan(g *Game, cfg Config, rows []PlanRow) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	seen := map[string]bool{}
	for _, row := range rows {
		rec, ok := cfg.LookupRecipe(row.Recipe)
		if !ok || !RecipeKnown(*g, rec) {
			return ErrPlanRecipe
		}
		if seen[row.Recipe] {
			return ErrPlanDuplicate
		}
		if row.Target < 0 {
			return ErrPlanTarget
		}
		seen[row.Recipe] = true
	}
	g.ProductionPlan = append([]PlanRow(nil), rows...)
	return nil
}
