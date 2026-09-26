package api

import (
	"context"
	"net/http"
	"testing"

	"lemonade-api/internal/domain"
)

// funded logs in and gives the player cash for the empire tests.
func (e *testEnv) funded(user string, cash int) uint {
	e.t.Helper()
	rec := e.do("POST", "/api/login", "", map[string]string{"username": user})
	id := decode[userDTO](e.t, rec).ID
	if _, err := e.repo.Mutate(context.Background(), id, func(g *domain.Game) (domain.Effects, error) {
		g.Capital = cash
		return domain.Effects{}, nil
	}); err != nil {
		e.t.Fatal(err)
	}
	return id
}

func (e *testEnv) empire(user string) empireDTO {
	e.t.Helper()
	rec := e.do("GET", "/api/game/empire", user, nil)
	if rec.Code != http.StatusOK {
		e.t.Fatalf("GET empire: %d %s", rec.Code, rec.Body)
	}
	return decode[empireDTO](e.t, rec)
}

func TestEmpireStartsInTheNeighborhood(t *testing.T) {
	e := newEnv(t)
	e.login("ann")
	em := e.empire("ann")
	if em.Era != 1 || len(em.Territories) != 5 || em.Reach != 80 {
		t.Fatalf("empire = era %d, %d territories, reach %d", em.Era, len(em.Territories), em.Reach)
	}
	nh := em.Territories[0]
	if !nh.Entered || nh.Share != 40 || len(nh.Rivals) != 3 || nh.EnterBlocked != "already_entered" {
		t.Fatalf("neighborhood = %+v", nh)
	}
	if em.Territories[1].EnterBlocked != "insufficient_funds" || em.Territories[2].EnterBlocked != "territory_locked" {
		t.Fatalf("blocks: %q %q", em.Territories[1].EnterBlocked, em.Territories[2].EnterBlocked)
	}
	if em.NextGoal == nil || em.NextGoal.Key != "city" || em.NextGoal.Cost != 15000 {
		t.Fatalf("next goal = %+v", em.NextGoal)
	}
	v := e.game("ann")
	if v.Era != 1 || v.NextGoal == nil || v.NextGoal.Key != "city" || v.Resources[4].Reach != 80 {
		t.Fatalf("view era/goal/reach = %d %+v %d", v.Era, v.NextGoal, v.Resources[4].Reach)
	}
}

func TestEnterBuyOutAndCampaignOverHTTP(t *testing.T) {
	e := newEnv(t)
	e.funded("bob", 100_000)

	e.wantError(e.do("POST", "/api/game/territories/region/enter", "bob", nil), http.StatusConflict, "territory_locked")
	e.wantError(e.do("POST", "/api/game/territories/atlantis/enter", "bob", nil), http.StatusNotFound, "not_found")
	if rec := e.do("POST", "/api/game/territories/city/enter", "bob", nil); rec.Code != http.StatusOK {
		t.Fatalf("enter city: %d %s", rec.Code, rec.Body)
	}
	e.wantError(e.do("POST", "/api/game/territories/city/enter", "bob", nil), http.StatusConflict, "already_entered")
	v := e.game("bob")
	if v.Era != 2 || v.Capital != 85_000 || v.Resources[4].Reach != 160 {
		t.Fatalf("after entering: era %d cash %d reach %d", v.Era, v.Capital, v.Resources[4].Reach)
	}

	e.wantError(e.do("POST", "/api/game/rivals/sour_sam/buyout", "bob", nil), http.StatusConflict, "rival_refuses")
	if rec := e.do("POST", "/api/game/rivals/lil_lucy/buyout", "bob", nil); rec.Code != http.StatusOK {
		t.Fatalf("buyout: %d %s", rec.Code, rec.Body)
	}
	e.wantError(e.do("POST", "/api/game/rivals/lil_lucy/buyout", "bob", nil), http.StatusConflict, "rival_gone")
	if rec := e.do("POST", "/api/game/rivals/sour_sam/buyout", "bob", map[string]bool{"hostile": true}); rec.Code != http.StatusOK {
		t.Fatalf("hostile buyout: %d %s", rec.Code, rec.Body)
	}
	e.wantError(e.do("POST", "/api/game/rivals/nobody/buyout", "bob", nil), http.StatusNotFound, "not_found")

	e.wantError(e.do("POST", "/api/game/territories/city/campaign", "bob", map[string]int{"level": 9}), http.StatusBadRequest, "invalid_campaign")
	if rec := e.do("POST", "/api/game/territories/city/campaign", "bob", map[string]int{"level": 1}); rec.Code != http.StatusOK {
		t.Fatalf("campaign: %d %s", rec.Code, rec.Body)
	}
	e.wantError(e.do("POST", "/api/game/territories/city/campaign", "bob", map[string]int{"level": 1}), http.StatusConflict, "campaign_running")

	em := e.empire("bob")
	if got := em.Territories[0].Share; got != 40+15+25 {
		t.Fatalf("neighborhood share %v after two buyouts", got)
	}
	if em.Territories[1].CampaignDaysLeft != 5 || em.Acquisitions == 0 {
		t.Fatalf("city campaign %d, acquisitions %d", em.Territories[1].CampaignDaysLeft, em.Acquisitions)
	}
	if v := e.game("bob"); v.NetWorth.Acquisitions != em.Acquisitions {
		t.Fatalf("net worth acquisitions %d vs %d", v.NetWorth.Acquisitions, em.Acquisitions)
	}
}

func TestBuyoutWithoutCashIs409(t *testing.T) {
	e := newEnv(t)
	e.funded("cyd", 50)
	e.wantError(e.do("POST", "/api/game/rivals/lil_lucy/buyout", "cyd", nil), http.StatusConflict, "insufficient_funds")
	e.wantError(e.do("POST", "/api/game/territories/city/enter", "cyd", nil), http.StatusConflict, "insufficient_funds")
}

// The empire belongs to its owner: another player's territories are never visible or actionable.
func TestEmpireIsPerPlayer(t *testing.T) {
	e := newEnv(t)
	e.funded("dee", 100_000)
	e.login("eve")
	if rec := e.do("POST", "/api/game/territories/city/enter", "dee", nil); rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if got := e.empire("eve"); got.Era != 1 || got.Territories[1].Entered {
		t.Fatalf("eve sees dee's empire: %+v", got.Territories[1])
	}
	if rec := e.do("GET", "/api/game/empire", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestWinningIsRecordedAndTheRunKeepsPlaying(t *testing.T) {
	e := newEnv(t)
	e.login("cyc1")
	cfg := domain.DefaultConfig()
	world := cfg.Territories[len(cfg.Territories)-1].Key
	if v := e.game("cyc1"); v.WonOnDay != 0 || v.Cycle != nil {
		t.Fatalf("a new game has not won and has no cycle: %+v", v)
	}
	e.setGame("cyc1", func(g *domain.Game) {
		g.Territories[world] = domain.TerritoryState{Entered: true, Share: 52}
	})

	// The next action records the win, unlocks the achievements and leaves the run active.
	rec := e.do("POST", "/api/game/buy", "cyc1", map[string]any{"resource": "lemon", "qty": 1})
	if rec.Code != http.StatusOK {
		t.Fatalf("buy: %d %s", rec.Code, rec.Body)
	}
	v := decode[gameViewDTO](t, rec)
	if v.WonOnDay != 1 || v.WonNetWorth <= 0 || v.Status != "active" {
		t.Fatalf("view after winning: won day %d, net worth %d, status %s", v.WonOnDay, v.WonNetWorth, v.Status)
	}
	got := map[string]bool{}
	for _, u := range v.Unlocked {
		got[u.Key] = true
	}
	if !got["victory"] || !got["underdog"] {
		t.Fatalf("unlocked %v, want victory and underdog", got)
	}

	// Giving up ends the run, scored by net worth, and the board marks it as a win.
	if rec := e.do("POST", "/api/game/give-up", "cyc1", nil); rec.Code != http.StatusOK {
		t.Fatalf("give up: %d %s", rec.Code, rec.Body)
	}
	board := decode[scoresDTO](t, e.do("GET", "/api/scores", "cyc1", nil))
	if len(board.Rows) != 1 || board.Rows[0].WonOnDay != 1 {
		t.Fatalf("board rows = %+v", board.Rows)
	}
}

func TestTheEconomistShowsHowLongACycleLasts(t *testing.T) {
	e := newEnv(t)
	e.login("cyc2")
	e.setGame("cyc2", func(g *domain.Game) { g.Cycle = domain.CycleState{Key: "boom", DaysLeft: 9} })
	c := e.game("cyc2").Cycle
	if c == nil || c.Key != "boom" || c.DaysLeft != nil {
		t.Fatalf("without the economist a cycle is named but its length is not: %+v", c)
	}
	e.setGame("cyc2", func(g *domain.Game) { g.Upgrades["economist"] = 1 })
	c = e.game("cyc2").Cycle
	if c == nil || c.DaysLeft == nil || *c.DaysLeft != 9 {
		t.Fatalf("with the economist the days left are sent: %+v", c)
	}
}
