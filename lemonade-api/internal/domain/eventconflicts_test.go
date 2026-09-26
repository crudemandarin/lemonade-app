package domain

import (
	"math/rand"
	"testing"
)

// 9b: conflicts are derived from the multipliers.
func TestConflictsDerivedFromMultipliers(t *testing.T) {
	cfg := DefaultConfig()
	var got [][2]string
	for i, a := range cfg.Events {
		for _, b := range cfg.Events[i+1:] {
			if conflicts(a, b) != conflicts(b, a) {
				t.Errorf("conflicts(%s,%s) is not symmetric", a.Key, b.Key)
			}
			if conflicts(a, b) {
				got = append(got, [2]string{a.Key, b.Key})
			}
		}
	}
	// The bumper crop (era 2) cheapens the lemons a blight makes dear.
	want := map[[2]string]bool{{"heat_wave", "rainy_week"}: true, {"rainy_week", "holiday"}: true, {"lemon_blight", "bumper_crop"}: true}
	if len(got) != len(want) {
		t.Fatalf("conflicting pairs = %v, want %v", got, want)
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("unexpected conflict %v", p)
		}
	}

	up := EventDef{Key: "up", Multipliers: map[Resource]float64{Cup: 1.2, Lemon: 0.9}}
	down := EventDef{Key: "down", Multipliers: map[Resource]float64{Lemon: 1.1}}
	same := EventDef{Key: "same", Multipliers: map[Resource]float64{Cup: 1.5}}
	if !conflicts(up, down) {
		t.Error("opposite lemon multipliers should conflict")
	}
	if conflicts(up, same) {
		t.Error("same-direction multipliers should not conflict")
	}
}

func TestHolidayAndRainyWeekNeverEligibleTogether(t *testing.T) {
	cfg := DefaultConfig()
	for _, pair := range [][2]string{{"rainy_week", "holiday"}, {"holiday", "rainy_week"}} {
		for _, d := range eligibleEvents(cfg.Events, []ActiveEvent{{Key: pair[0]}}) {
			if d.Key == pair[1] {
				t.Errorf("%s eligible while %s active", pair[1], pair[0])
			}
		}
	}
	ok := false
	for _, d := range eligibleEvents(cfg.Events, []ActiveEvent{{Key: "heat_wave"}}) {
		ok = ok || d.Key == "holiday"
	}
	if !ok {
		t.Error("holiday should still be eligible during a heat wave")
	}
}

func TestNoOpposingEventsEverOverlap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EventChance = 1
	byKey := map[string]EventDef{}
	for _, d := range cfg.Events {
		byKey[d.Key] = d
	}
	sawHeatAndHoliday := false
	for seed := int64(1); seed <= 20; seed++ {
		g := NewGame(cfg, seed)
		rng := rand.New(rand.NewSource(seed))
		for day := 0; day < 300; day++ {
			tickEvents(&g, rng, cfg)
			for i, a := range g.Events {
				for _, b := range g.Events[i+1:] {
					if conflicts(byKey[a.Key], byKey[b.Key]) {
						t.Fatalf("seed %d day %d: %s and %s overlap", seed, day, a.Key, b.Key)
					}
				}
			}
			k := eventKeys(g.Events)
			sawHeatAndHoliday = sawHeatAndHoliday || (k["heat_wave"] && k["holiday"])
		}
	}
	if !sawHeatAndHoliday {
		t.Error("heat wave and holiday never co-occurred, so the rule is over-blocking")
	}
}
