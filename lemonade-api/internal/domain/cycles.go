package domain

import (
	"math"

	"lemonade-api/internal/domain/content"
)

// CycleState is the running economic regime. Key is empty when the economy is stable.
// Drift is the price multiplier inflation has built up; it eases back to 1 after the
// regime ends. Zero means 1, so games saved before cycles load unchanged.
type CycleState struct {
	Key      string
	DaysLeft int
	Drift    float64
}

// driftStepBack is how fast a built-up price drift unwinds once inflation is over.
const driftStepBack = 1.01

func cycleDef(key string) (content.CycleDef, bool) {
	for _, d := range content.Cycles {
		if d.Key == key {
			return d, true
		}
	}
	return content.CycleDef{}, false
}

// ActiveCycle is the running regime's definition, if any.
func ActiveCycle(g Game) (content.CycleDef, bool) {
	if g.Cycle.Key == "" {
		return content.CycleDef{}, false
	}
	return cycleDef(g.Cycle.Key)
}

// cycleDepthFactor scales every commodity's free depth (1 when stable).
func cycleDepthFactor(g Game) float64 {
	if d, ok := ActiveCycle(g); ok {
		return 1 + d.DepthPct/100
	}
	return 1
}

// cycleValuationFactor scales what a rival is worth to a buyer (1 when stable).
func cycleValuationFactor(g Game) float64 {
	if d, ok := ActiveCycle(g); ok {
		return 1 + d.ValuationPct/100
	}
	return 1
}

// cycleUpkeep is the extra daily upkeep a regime adds to a base upkeep.
func cycleUpkeep(g Game, base int) int {
	if d, ok := ActiveCycle(g); ok && d.UpkeepPct != 0 {
		return int(math.Round(float64(base) * d.UpkeepPct / 100))
	}
	return 0
}

// priceDrift is the multiplier inflation has put on every price (1 when none).
func priceDrift(g Game) float64 {
	if g.Cycle.Drift > 1 {
		return g.Cycle.Drift
	}
	return 1
}

// stepCycles is the economy's part of EndDay step 8. It draws exactly three numbers
// from its own stream every day, used or not, so the stream never shifts. The running
// regime counts down first; a new one may start only on a day none is running.
func stepCycles(g *Game, cfg Config, report *DayReport) {
	rng := dayRNG(g.Seed, g.Day, SaltCycles)
	chance, pick, span := rng.Float64(), rng.Float64(), rng.Float64()

	if d, ok := ActiveCycle(*g); ok {
		g.Cycle.DaysLeft--
		if g.Cycle.DaysLeft <= 0 {
			report.CycleEnded = d.Key
			g.Cycle.Key, g.Cycle.DaysLeft = "", 0
		}
	} else if chance < cfg.CycleChance && len(content.Cycles) > 0 {
		d := content.Cycles[min(int(pick*float64(len(content.Cycles))), len(content.Cycles)-1)]
		days := d.MinDays + int(span*float64(d.MaxDays-d.MinDays+1))
		g.Cycle.Key, g.Cycle.DaysLeft = d.Key, min(days, d.MaxDays)
		report.CycleStarted = d.Key
	}

	// Inflation builds the drift a day at a time, and it unwinds at the same pace.
	if d, ok := ActiveCycle(*g); ok && d.DriftPct != 0 {
		g.Cycle.Drift = priceDrift(*g) * (1 + d.DriftPct/100)
	} else if g.Cycle.Drift > 1 {
		g.Cycle.Drift = math.Max(1, g.Cycle.Drift/driftStepBack)
		if g.Cycle.Drift < 1.0005 {
			g.Cycle.Drift = 0
		}
	}
}
