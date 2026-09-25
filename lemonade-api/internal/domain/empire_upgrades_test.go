package domain

import (
	"errors"
	"testing"
)

// Every territory upgrade is gated by the era the player has actually reached.
func TestTerritoryUpgradesAreEraGated(t *testing.T) {
	g, cfg := newTestGame()
	flush(&g)
	if err := BuyUpgrade(&g, cfg, "roadside_sign"); err != nil {
		t.Fatalf("era 1 sign: %v", err)
	}
	var lock *UpgradeLockedError
	if err := BuyUpgrade(&g, cfg, "billboard"); !errors.As(err, &lock) || lock.Code != "era" {
		t.Fatalf("billboard in era 1: %v", err)
	}
	if err := EnterTerritory(&g, cfg, "city"); err != nil {
		t.Fatal(err)
	}
	flush(&g)
	if err := BuyUpgrade(&g, cfg, "billboard"); err != nil {
		t.Fatalf("billboard in era 2: %v", err)
	}
}

// presence_bonus: a sign in the Neighborhood turns the contest around; the same game
// without it stays put at 40%.
func TestPresenceBonusMovesTheShareContest(t *testing.T) {
	run := func(keys ...string) float64 {
		g, cfg := newTestGame()
		own(&g, keys...)
		g.SellPressure[Lemonade] = 300 // sells its depth: no fill penalty
		for i := 0; i < 10; i++ {
			g.SellPressure[Lemonade] = 300
			flush(&g)
			if _, err := EndDay(&g, cfg); err != nil {
				t.Fatal(err)
			}
		}
		return g.Territories["neighborhood"].Share
	}
	if got := run(); got != 40 {
		t.Fatalf("without a sign: %v", got)
	}
	if got := run("roadside_sign"); got <= 40 {
		t.Fatalf("with a sign the share stayed %v", got)
	}
}

func TestPresenceBonusTargetsTheNamedTerritoryOrAll(t *testing.T) {
	g, cfg := newTestGame()
	own(&g, "roadside_sign", "billboard")
	if got := presenceBonus(g, cfg, "neighborhood"); got != 0.05 {
		t.Fatalf("neighborhood %v", got)
	}
	if got := presenceBonus(g, cfg, "city"); got != 0.10 {
		t.Fatalf("city %v", got)
	}
	if got := presenceBonus(g, cfg, "region"); got != 0 {
		t.Fatalf("region %v", got)
	}
	own(&g, "global_brand")
	if got := presenceBonus(g, cfg, "region"); got != 0.10 {
		t.Fatalf("region with the global brand %v", got)
	}
	if got := presenceBonus(g, cfg, "neighborhood"); got < 0.149 || got > 0.151 {
		t.Fatalf("neighborhood with sign and brand %v", got)
	}
}

func TestHubUpkeepDiscountCutsHubsOnly(t *testing.T) {
	g, cfg := newTestGame()
	g.Territories["city"] = TerritoryState{Entered: true, Share: 10}
	g.Territories["region"] = TerritoryState{Entered: true, Share: 8}
	plain := HubUpkeep(g, cfg) // (150+900) less a 10% synergy discount
	if plain != 945 {
		t.Fatalf("hub upkeep %d, want 945", plain)
	}
	before := WarehouseUpkeep(g, cfg) + ProductionUpkeep(g, cfg)
	own(&g, "delivery_fleet")
	if got := HubUpkeep(g, cfg); got != 709 { // 945 less 25%
		t.Fatalf("with the fleet: %d, want 709", got)
	}
	if WarehouseUpkeep(g, cfg)+ProductionUpkeep(g, cfg) != before {
		t.Fatal("the fleet changed building upkeep")
	}
}

func TestRivalIntelShowsTelegraphsTwoDaysAhead(t *testing.T) {
	g, cfg := newTestGame()
	if TelegraphVisibleDays(g, cfg) != 1 {
		t.Fatal("one day without intel")
	}
	own(&g, "rival_intel")
	if TelegraphVisibleDays(g, cfg) != 2 {
		t.Fatal("two days with intel")
	}
}

func TestPRTeamHalvesPriceWars(t *testing.T) {
	g, cfg := newTestGame()
	if priceWarDays(g, cfg) != 3 {
		t.Fatal("three days normally")
	}
	own(&g, "pr_team")
	if got := priceWarDays(g, cfg); got != 1 {
		t.Fatalf("with the PR team %d, want 1", got)
	}
}

func TestGlobalBrandUnlocksTheGlobalLeaderGoal(t *testing.T) {
	g, cfg := newTestGame()
	if HasUnlock(g, cfg, "global_leader") {
		t.Fatal("locked at the start")
	}
	own(&g, "global_brand")
	if !HasUnlock(g, cfg, "global_leader") {
		t.Fatal("global brand unlocks it")
	}
}

func TestRadioSpotAddsLemonadeDepth(t *testing.T) {
	g, cfg := newTestGame()
	base := freeDepth(g, cfg, Lemonade)
	own(&g, "radio_spot")
	if got := freeDepth(g, cfg, Lemonade); got != int(float64(base)*1.05+0.5) {
		t.Fatalf("depth %d, base %d", got, base)
	}
}
