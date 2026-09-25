package domain

import "testing"

func TestNewGameEntersTheNeighborhoodWithItsRivals(t *testing.T) {
	g, cfg := newTestGame()
	nh := g.Territories["neighborhood"]
	if !nh.Entered || nh.Share != 40 {
		t.Fatalf("neighborhood = %+v, want entered at 40%%", nh)
	}
	for _, k := range []string{"lil_lucy", "sour_sam", "squeeze_box"} {
		if r := g.Rivals[k]; r.Status != RivalActive || r.Share == 0 || r.Valuation == 0 {
			t.Errorf("rival %s = %+v", k, r)
		}
	}
	if Era(g, cfg) != 1 {
		t.Errorf("era = %d, want 1", Era(g, cfg))
	}
}

// A game saved before territories has nil maps; seeding gives it exactly the start.
func TestSeedEmpireFillsAnOldSave(t *testing.T) {
	g, cfg := newTestGame()
	g.Territories, g.Rivals = nil, nil
	SeedEmpire(&g, cfg)
	if g.Territories["neighborhood"].Share != 40 || g.Rivals["sour_sam"].Share != 25 {
		t.Fatalf("seeded %+v %+v", g.Territories, g.Rivals)
	}
	before := g.Clone()
	SeedEmpire(&g, cfg) // idempotent
	if before.Territories["neighborhood"] != g.Territories["neighborhood"] || len(before.Rivals) != len(g.Rivals) {
		t.Fatal("seeding twice changed the game")
	}
}

// Reach with no territory beyond the start is the phase 0 depth, whether or not the maps exist.
func TestReachIsPhaseZeroDepthAtTheStartShare(t *testing.T) {
	g, cfg := newTestGame()
	for level, want := range []int{80, 280, 480, 960, 960, 960, 960} {
		g.WarehouseLevel = level + 1
		for _, r := range Resources {
			if got := freeDepth(g, cfg, r); got != want {
				t.Fatalf("level %d %s: %d, want %d", level+1, r, got, want)
			}
		}
		old := g.Clone()
		old.Territories = nil
		if freeDepth(old, cfg, Lemonade) != want {
			t.Fatalf("level %d: an old save differs", level+1)
		}
	}
}

func TestNeighborhoodReachScalesWithShare(t *testing.T) {
	g, cfg := newTestGame()
	nh := g.Territories["neighborhood"]
	nh.Share = 60
	g.Territories["neighborhood"] = nh
	if got := freeDepth(g, cfg, Lemonade); got != 120 {
		t.Fatalf("60%% share gives depth %d, want 120", got)
	}
}

func TestOtherTerritoriesAddDemandTimesShare(t *testing.T) {
	g, cfg := newTestGame()
	g.Territories["city"] = TerritoryState{Entered: true, Share: 10}
	if got := freeDepth(g, cfg, Lemonade); got != 80+80 {
		t.Fatalf("depth with the city at 10%% = %d, want 160", got)
	}
	if got := ReachIn(g, cfg, "city", Lemonade); got != 80 {
		t.Fatalf("city part = %d, want 80", got)
	}
}

func TestEraIsTheHighestTerritoryEntered(t *testing.T) {
	g, cfg := newTestGame()
	for i, k := range []string{"city", "region", "nation", "world"} {
		g.Territories[k] = TerritoryState{Entered: true, Share: 5}
		if got := Era(g, cfg); got != i+2 {
			t.Fatalf("after %s era = %d, want %d", k, got, i+2)
		}
	}
}

func TestTiersFiveToSevenUnlockByEra(t *testing.T) {
	g, cfg := newTestGame()
	for i, tc := range []struct {
		territory string
		cap       int
	}{{"", 4}, {"city", 4}, {"region", 4}, {"nation", 5}, {"world", 7}} {
		if tc.territory != "" {
			g.Territories[tc.territory] = TerritoryState{Entered: true, Share: 5}
		}
		if got := levelCap(g, cfg); got != tc.cap {
			t.Fatalf("step %d (%s): level cap %d, want %d", i, tc.territory, got, tc.cap)
		}
	}
}

func TestOldSavesAreGrandfatheredAboveTheCap(t *testing.T) {
	g, cfg := newTestGame()
	g.ProductionLevel, g.WarehouseLevel = 4, 4
	if got := levelCap(g, cfg); got < 4 {
		t.Fatalf("cap %d below the held level", got)
	}
}

func TestUpgradeStopsAtTheEraCap(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 10_000_000
	g.ProductionLevel, g.WarehouseLevel = 4, 4
	if err := Upgrade(&g, cfg, Production); err != ErrMaxLevel {
		t.Fatalf("level 4 to 5 without the nation: %v, want ErrMaxLevel", err)
	}
	g.Territories["nation"] = TerritoryState{Entered: true, Share: 5}
	if err := Upgrade(&g, cfg, Production); err != nil {
		t.Fatalf("with the nation: %v", err)
	}
}

func TestBuildingCapGrowsWithTerritories(t *testing.T) {
	g, cfg := newTestGame()
	if buildingCap(g, cfg) != 10 {
		t.Fatalf("cap %d, want 10", buildingCap(g, cfg))
	}
	want := 10
	for _, k := range []string{"city", "region", "nation", "world"} {
		g.Territories[k] = TerritoryState{Entered: true, Share: 5}
		if k == "world" {
			want += 20
		} else {
			want += 10
		}
		if got := buildingCap(g, cfg); got != want {
			t.Fatalf("after %s cap %d, want %d", k, got, want)
		}
	}
}

func TestHubUpkeepHasASynergyDiscount(t *testing.T) {
	g, cfg := newTestGame()
	if HubUpkeep(g, cfg) != 0 {
		t.Fatal("the neighborhood has no hub")
	}
	g.Territories["city"] = TerritoryState{Entered: true, Share: 10}
	if got := HubUpkeep(g, cfg); got != 143 { // 150 less 5%
		t.Fatalf("one hub beyond the first: %d, want 143", got)
	}
	g.Territories["region"] = TerritoryState{Entered: true, Share: 8}
	g.Territories["nation"] = TerritoryState{Entered: true, Share: 5}
	g.Territories["world"] = TerritoryState{Entered: true, Share: 3}
	if got, want := HubUpkeep(g, cfg), 28840; got != want {
		t.Fatalf("four hubs: %d, want %d (20%% cap)", got, want)
	}
	if TotalUpkeep(g, cfg) != WarehouseUpkeep(g, cfg)+ProductionUpkeep(g, cfg)+HubUpkeep(g, cfg) {
		t.Fatal("hub upkeep is not in the total")
	}
}
