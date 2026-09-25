package domain

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"lemonade-api/internal/domain/content"
)

// tune edits one territory row of a config copy.
func tune(cfg *Config, key string, cost int, share float64, depth int) {
	for i, d := range cfg.Territories {
		if d.Key == key {
			if cost >= 0 {
				cfg.Territories[i].EntryCost = cost
			}
			if share >= 0 {
				cfg.Territories[i].EntryShare = share
			}
			if depth >= 0 {
				cfg.Territories[i].Depth = depth
			}
		}
	}
}

// scaleRivals multiplies every rival's starting buyout (the valuation multiple knob).
func scaleValuation(cfg *Config, m float64) { cfg.ValuationMultiple = m }

type variant struct {
	name  string
	apply func(*Config)
}

func sweepSummary(cfg Config, seeds int) string {
	rs := runSeeds(cfg, diligent, 200, seeds)
	var eraDays [6][]int
	var eraNW [6][]int
	bankrupt := 0
	var nw90, nw150, nw200 []int
	for _, r := range rs {
		for e := 2; e <= 5; e++ {
			if r.eraDay[e] > 0 {
				eraDays[e] = append(eraDays[e], r.eraDay[e])
				eraNW[e] = append(eraNW[e], r.eraNW[e])
			}
		}
		if r.bankrupt > 0 {
			bankrupt++
		}
		if len(r.nw) >= 90 {
			nw90 = append(nw90, r.nw[89])
		}
		if len(r.nw) >= 150 {
			nw150 = append(nw150, r.nw[149])
		}
		if len(r.nw) >= 200 {
			nw200 = append(nw200, r.nw[199])
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "bankrupt %d%% |", 100*bankrupt/seeds)
	for e := 2; e <= 5; e++ {
		if len(eraDays[e]) == 0 {
			fmt.Fprintf(&b, " E%d -", e)
			continue
		}
		fmt.Fprintf(&b, " E%d d%d (%d%%, %s)", e, median(eraDays[e]), 100*len(eraDays[e])/seeds, compactMoney(median(eraNW[e])))
	}
	if len(nw90) > 0 {
		fmt.Fprintf(&b, " | nw d90 %s", compactMoney(median(nw90)))
	}
	if len(nw150) > 0 {
		fmt.Fprintf(&b, " d150 %s", compactMoney(median(nw150)))
	}
	if len(nw200) > 0 {
		fmt.Fprintf(&b, " d200 %s", compactMoney(median(nw200)))
	}
	return b.String()
}

// TestEmpireSweep prints the diligent bot's era table for named config variants, one
// group of knobs at a time:
//
//	SWEEP=1 SWEEP_ONLY=city SWEEP_SEEDS=16 go test ./internal/domain -run TestEmpireSweep -v -count=1
func TestEmpireSweep(t *testing.T) {
	if os.Getenv("SWEEP") == "" {
		t.Skip("set SWEEP=1 to run the empire sweeps")
	}
	seeds := 16
	if v := os.Getenv("SWEEP_SEEDS"); v != "" {
		fmt.Sscan(v, &seeds)
	}
	only := os.Getenv("SWEEP_ONLY")
	for _, v := range sweepVariants() {
		if only != "" && !strings.Contains(v.name, only) {
			continue
		}
		cfg := DefaultConfig()
		v.apply(&cfg)
		fmt.Printf("%-34s %s\n", v.name, sweepSummary(cfg, seeds))
	}
}

var _ = content.Passive

func sweepVariants() []variant {
	return []variant{
		{"base", func(c *Config) {}},
		{"city cost 10k", func(c *Config) { tune(c, "city", 10000, -1, -1) }},
		{"city cost 8k share 12", func(c *Config) { tune(c, "city", 8000, 12, -1) }},
		{"region share 10", func(c *Config) { tune(c, "region", -1, 10, -1) }},
		{"region cost 80k share 10", func(c *Config) { tune(c, "region", 80000, 10, -1) }},
		{"all: 10k/80k/600k/4M", func(c *Config) {
			tune(c, "city", 10000, -1, -1)
			tune(c, "region", 80000, 10, -1)
			tune(c, "nation", 600000, 8, -1)
			tune(c, "world", 4000000, 5, -1)
		}},
	}
}
