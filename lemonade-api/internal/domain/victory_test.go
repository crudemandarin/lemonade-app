package domain

import (
	"testing"

	"lemonade-api/internal/domain/content"
)

func TestVictoryNeedsHalfTheWorldOrTheFinalBuyout(t *testing.T) {
	cfg := DefaultConfig()
	world := cfg.Territories[len(cfg.Territories)-1].Key
	fresh := NewGame(cfg, 1)
	edit := func(f func(g *Game)) Game {
		g := fresh.Clone()
		f(&g)
		return g
	}
	cases := []struct {
		name string
		g    Game
		want bool
	}{
		{"a new game", fresh, false},
		{"49.9% of the World", edit(func(g *Game) { g.Territories[world] = TerritoryState{Entered: true, Share: 49.9} }), false},
		{"50% of the World", edit(func(g *Game) { g.Territories[world] = TerritoryState{Entered: true, Share: 50} }), true},
		{"share of a world not entered", edit(func(g *Game) { g.Territories[world] = TerritoryState{Share: 80} }), false},
		{"the final rival bought", edit(func(g *Game) { g.Rivals[FinalRival] = RivalState{Status: RivalAcquired} }), true},
	}
	for _, c := range cases {
		if got := VictoryConditionMet(c.g, cfg); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRecordVictoryOnceAndTheRunKeepsPlaying(t *testing.T) {
	cfg := DefaultConfig()
	g := NewGame(cfg, 1)
	world := cfg.Territories[len(cfg.Territories)-1].Key
	g.Territories[world] = TerritoryState{Entered: true, Share: 55}
	g.Day = 120
	RecordVictory(&g, cfg)
	if g.WonOnDay != 120 || g.WonNetWorth != NetWorth(g, cfg) {
		t.Fatalf("won on %d with %d", g.WonOnDay, g.WonNetWorth)
	}
	if g.Status != StatusActive {
		t.Fatalf("winning must not end the run, status %s", g.Status)
	}
	g.Day, g.Capital = 140, g.Capital+50_000
	RecordVictory(&g, cfg)
	if g.WonOnDay != 120 {
		t.Fatalf("the win day must not move, got %d", g.WonOnDay)
	}
	rec := FinishRun(g, cfg, EndedByGaveUp)
	if rec.WonOnDay != 120 || rec.Score != NetWorth(g, cfg) {
		t.Fatalf("the record should carry the win day and score the final net worth: %+v", rec)
	}
}

func TestVictoryAchievements(t *testing.T) {
	cfg := DefaultConfig()
	fresh := NewGame(cfg, 1)
	byShare, byBuyout := fresh.Clone(), fresh.Clone()
	byShare.WonOnDay = 100
	byBuyout.WonOnDay = 100
	byBuyout.Rivals[FinalRival] = RivalState{Status: RivalAcquired}
	for _, c := range []struct {
		name string
		p    content.Predicate
		g    Game
		want bool
	}{
		{"victory before winning", content.Won(), fresh, false},
		{"victory by share", content.Won(), byShare, true},
		{"underdog by share", content.WonWithoutBuyout(), byShare, true},
		{"no underdog after the buyout", content.WonWithoutBuyout(), byBuyout, false},
		{"victory by buyout", content.Won(), byBuyout, true},
	} {
		if got := holdsFor(c.p, fresh, c.g, AchievementContext{}); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
