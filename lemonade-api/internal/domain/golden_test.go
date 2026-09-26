package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Golden bot runs. Every simulated player plays fixed seeds, and each end of day is
// written as one line that does not depend on how the game stores its data (keys are
// spelled out, maps are read in a fixed order). The file was recorded before the
// commodities became catalog data (late game Products A), so a refactor that claims
// "no gameplay change" must reproduce it byte for byte.
//
// To re-record after a deliberate gameplay change: UPDATE_GOLDEN=1 go test ./internal/domain -run TestGoldenBotRuns

const goldenFile = "testdata/golden_bots.txt"

// goldenKeys are the five original commodities, spelled out so the trace never
// depends on Resources or the catalog.
var goldenKeys = []Resource{"lemon", "sugar", "ice", "cup", "lemonade"}

// goldenSink collects trace lines while a golden run is in progress; nil otherwise.
var goldenSink *strings.Builder

// simEndDay is EndDay for the simulated players, tracing each day when a golden run
// is recording.
func simEndDay(g *Game, cfg Config) (DayReport, error) {
	r, err := EndDay(g, cfg)
	if goldenSink != nil && err == nil {
		goldenSink.WriteString(goldenLine(*g, cfg, r))
	}
	return r, err
}

func goldenLine(g Game, cfg Config, r DayReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "d%d %s cap=%d prod=%d melt=%d upk=%d forced=%d/%d pl=%d/%d wl=%d |",
		g.Day, g.Status, g.Capital, r.Produced, r.IceMelted, r.UpkeepPaid,
		r.ForcedSaleCases, r.ForcedSaleProceeds, g.ProductionLevel, g.ProductionQty, g.WarehouseLevel)
	q := Quotes(g, cfg)
	for _, k := range goldenKeys {
		p := 0.0
		if m := g.Market[k]; m != nil {
			p = m.Price
		}
		fmt.Fprintf(&b, " %s:inv=%d wq=%d cb=%d bp=%s sp=%s p=%s q=%d/%d/%d",
			k, g.Inventory[k], g.WarehouseQty[ClassOf(cfg, k)], g.CostBasis[k],
			strconv.FormatFloat(g.BuyPressure[k], 'g', -1, 64),
			strconv.FormatFloat(g.SellPressure[k], 'g', -1, 64),
			strconv.FormatFloat(p, 'g', -1, 64),
			q[k].Price, q[k].Bid, q[k].Ask)
	}
	var ev []string
	for _, e := range g.Events {
		ev = append(ev, fmt.Sprintf("%s:%d", e.Key, e.DaysLeft))
	}
	var changes []string
	for _, c := range r.PriceChanges {
		changes = append(changes, fmt.Sprintf("%s:%d>%d", c.Resource, c.Before, c.After))
	}
	sort.Strings(changes)
	fmt.Fprintf(&b, " | ev=%s pc=%s stats=%+v tl=%d\n", strings.Join(ev, ","), strings.Join(changes, ","), g.Stats, len(g.Timeline))
	return b.String()
}

func TestGoldenBotRuns(t *testing.T) {
	cfg := DefaultConfig()
	bots := []struct {
		name string
		run  func(Config, int64, int) simResult
	}{
		{"grower", grower}, {"sloppy", sloppy}, {"careless", careless}, {"idle", idle},
		{"spammer", spammer}, {"thresholder", thresholder}, {"hoarder", hoarder},
		{"thresholderStrict", thresholderStrict}, {"opportunist", opportunist},
	}
	var out strings.Builder
	goldenSink = &out
	defer func() { goldenSink = nil }()
	for _, bot := range bots {
		for seed := int64(1); seed <= 2; seed++ {
			fmt.Fprintf(&out, "== %s seed %d\n", bot.name, seed)
			bot.run(cfg, seed, 120)
		}
	}
	got := out.String()

	path := filepath.FromSlash(goldenFile)
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (record with UPDATE_GOLDEN=1): %v", err)
	}
	if got != string(want) {
		gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for i := 0; i < len(gl) && i < len(wl); i++ {
			if gl[i] != wl[i] {
				t.Fatalf("bot runs differ from the golden file at line %d:\n got: %s\nwant: %s", i+1, gl[i], wl[i])
			}
		}
		t.Fatalf("bot runs differ from the golden file in length: got %d lines, want %d", len(gl), len(wl))
	}
}
