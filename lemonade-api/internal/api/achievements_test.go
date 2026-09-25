package api

import (
	"context"
	"net/http"
	"testing"

	"lemonade-api/internal/domain"
	"lemonade-api/internal/domain/content"
)

func (e *testEnv) achievements(user string) achievementsDTO {
	e.t.Helper()
	rec := e.do("GET", "/api/achievements", user, nil)
	if rec.Code != http.StatusOK {
		e.t.Fatalf("GET /api/achievements: %d %s", rec.Code, rec.Body)
	}
	return decode[achievementsDTO](e.t, rec)
}

func findAchievement(t *testing.T, list achievementsDTO, key string) achievementDTO {
	t.Helper()
	for _, a := range list.Achievements {
		if a.Key == key {
			return a
		}
	}
	t.Fatalf("no achievement %q", key)
	return achievementDTO{}
}

func TestAchievementsNeedALogin(t *testing.T) {
	e := newEnv(t)
	e.wantError(e.do("GET", "/api/achievements", "", nil), http.StatusUnauthorized, "unauthorized")
}

func TestListShowsEveryDefinitionAndMasksHiddenOnes(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	list := e.achievements("joe12")
	if list.Total != len(content.Achievements) || len(list.Achievements) != list.Total || list.UnlockedCount != 0 || len(list.Categories) == 0 {
		t.Fatalf("list = total %d, rows %d, unlocked %d", list.Total, len(list.Achievements), list.UnlockedCount)
	}
	shown := findAchievement(t, list, "nw_5k")
	if shown.Name == nil || *shown.Name != "Pocket money" || shown.Description == nil || shown.Unlocked || shown.UnlockedAt != nil {
		t.Fatalf("visible row = %+v", shown)
	}
	hidden := findAchievement(t, list, "just_ice")
	if !hidden.Hidden || hidden.Name != nil || hidden.Description != nil || hidden.Progress != nil {
		t.Fatalf("a locked hidden row leaks: %+v", hidden)
	}
	raw := e.do("GET", "/api/achievements", "joe12", nil).Body.String()
	if contains(raw, "Just ice") {
		t.Fatal("the hidden name is in the response")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestProgressShowsMeasurableCounts(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.setGame("joe12", func(g *domain.Game) { g.Stats.Produced = 620 })
	list := e.achievements("joe12")
	p := findAchievement(t, list, "made_1k").Progress
	if p == nil || p.Current != 620 || p.Target != 1000 {
		t.Fatalf("made_1k progress = %+v", p)
	}
	if findAchievement(t, list, "full_house").Progress != nil {
		t.Fatal("full_house has no progress bar")
	}
}

func TestAMutationReturnsWhatItUnlocked(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	if u := e.game("joe12").Unlocked; u == nil || len(u) != 0 {
		t.Fatalf("a plain read must return [] not %v", u)
	}

	// Building unlocks "Growing"; the toast carries key, name and tier.
	rec := e.do("POST", "/api/game/facilities/warehouse/expand", "joe12", map[string]any{"resource": "lemon"})
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Body)
	}
	v := decode[gameViewDTO](t, rec)
	if len(v.Unlocked) != 1 || v.Unlocked[0] != (unlockedDTO{Key: "first_expand", Name: "Growing", Tier: "bronze"}) {
		t.Fatalf("unlocked = %+v", v.Unlocked)
	}
	// Unlocking is once: the same action again returns nothing new.
	v = decode[gameViewDTO](t, e.do("POST", "/api/game/facilities/warehouse/expand", "joe12", map[string]any{"resource": "lemon"}))
	if len(v.Unlocked) != 0 {
		t.Fatalf("second expand unlocked %+v", v.Unlocked)
	}
	list := e.achievements("joe12")
	got := findAchievement(t, list, "first_expand")
	if !got.Unlocked || got.UnlockedAt == nil || got.RunID == nil || list.UnlockedCount != 1 {
		t.Fatalf("stored = %+v (%d)", got, list.UnlockedCount)
	}
}

func TestAFailedActionUnlocksNothing(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.setGame("joe12", func(g *domain.Game) { g.Capital = 50 })
	e.wantError(e.do("POST", "/api/game/facilities/production/expand", "joe12", nil), http.StatusConflict, "insufficient_funds")
	if n := e.achievements("joe12").UnlockedCount; n != 0 {
		t.Fatalf("unlocked %d after a refused action", n)
	}
}

func TestEndDayUnlocksAndRecordsGoalFacts(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.setGame("joe12", func(g *domain.Game) { g.Day = 6 })
	rec := e.do("POST", "/api/game/end-day", "joe12", nil)
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Body)
	}
	out := decode[endDayResponseDTO](t, rec)
	if len(out.Game.Unlocked) != 1 || out.Game.Unlocked[0].Key != "day_7" {
		t.Fatalf("unlocked = %+v", out.Game.Unlocked)
	}
	if got := e.storedGame("joe12").Goals; got.DaysClosed != 1 {
		t.Fatalf("goal facts after ending a day: %+v", got)
	}
}

func TestGivingUpRichUnlocksAndCountsTheRun(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.setGame("joe12", func(g *domain.Game) { g.Capital = 60000 })
	v := decode[gameViewDTO](t, e.do("POST", "/api/game/give-up", "joe12", nil))
	keys := map[string]bool{}
	for _, u := range v.Unlocked {
		keys[u.Key] = true
	}
	// A hidden achievement still shows its name once it unlocks.
	for _, k := range []string{"give_up_rich", "nw_5k", "nw_10k", "nw_25k", "top_10"} {
		if !keys[k] {
			t.Errorf("expected %s in %+v", k, v.Unlocked)
		}
	}
	if keys["new_best"] {
		t.Error("a first run is not a new personal best")
	}
	hidden := findAchievement(t, e.achievements("joe12"), "give_up_rich")
	if !hidden.Unlocked || hidden.Name == nil || *hidden.Name != "Quit while ahead" {
		t.Fatalf("unlocked hidden row = %+v", hidden)
	}
}

func TestSecondRunBeatingTheBestUnlocksNewBest(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.giveUpAfterBuying("joe12", 5)
	e.do("POST", "/api/game/new", "joe12", nil)
	e.giveUpAfterBuying("joe12", 0) // 1500 beats 1480
	if !findAchievement(t, e.achievements("joe12"), "new_best").Unlocked {
		t.Fatal("new_best should have unlocked")
	}
}

func TestTop10NeedsARankInTheTopTen(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 10; i++ {
		name := "rich" + string(rune('a'+i)) + "12"
		e.login(name)
		e.setGame(name, func(g *domain.Game) { g.Capital = 90000 })
		e.do("POST", "/api/game/give-up", name, nil)
	}
	e.login("poor12")
	e.do("POST", "/api/game/give-up", "poor12", nil)
	if findAchievement(t, e.achievements("poor12"), "top_10").Unlocked {
		t.Fatal("eleventh place is not top 10")
	}
}

func TestRunDetailListsWhatItUnlocked(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.setGame("joe12", func(g *domain.Game) { g.Capital = 6000 })
	v := decode[gameViewDTO](t, e.do("POST", "/api/game/give-up", "joe12", nil))
	if len(v.Unlocked) == 0 {
		t.Fatal("setup: nothing unlocked")
	}
	runs := decode[[]runSummaryDTO](t, e.do("GET", "/api/runs", "joe12", nil))
	d := decode[runDetailDTO](t, e.do("GET", "/api/runs/"+runs[0].RunID, "joe12", nil))
	if len(d.Achievements) != len(v.Unlocked) {
		t.Fatalf("run detail %+v, toast %+v", d.Achievements, v.Unlocked)
	}
}

func TestBoardRowsCarryTheAchievementCount(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.setGame("joe12", func(g *domain.Game) { g.Capital = 6000 })
	v := decode[gameViewDTO](t, e.do("POST", "/api/game/give-up", "joe12", nil))
	board := decode[scoresDTO](t, e.do("GET", "/api/scores", "joe12", nil))
	if len(board.Rows) != 1 || board.Rows[0].Achievements != len(v.Unlocked) || board.Me == nil || board.Me.Achievements != len(v.Unlocked) {
		t.Fatalf("board = %+v vs %d unlocked", board, len(v.Unlocked))
	}
}

func TestBackfillGrantsWhatStoredRunsProveOnceOnly(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	u, _ := e.repo.FindUser(context.Background(), "joe12")
	ctx := context.Background()
	// A finished run stored by an older build: no achievements were ever evaluated.
	if _, err := e.repo.Mutate(ctx, u.ID, func(g *domain.Game) (domain.Effects, error) {
		rec := domain.FinishRun(*g, e.cfg, domain.EndedByGaveUp)
		rec.Score, rec.NetWorth, rec.Days = 30000, 30000, 40
		rec.Stats.Produced = 1200
		return domain.Effects{Finished: &rec}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if n := e.achievements("joe12").UnlockedCount; n != 0 {
		t.Fatalf("setup: %d already unlocked", n)
	}

	n, err := BackfillAchievements(ctx, e.repo)
	if err != nil || n == 0 {
		t.Fatalf("backfill: %d, %v", n, err)
	}
	list := e.achievements("joe12")
	for _, k := range []string{"nw_25k", "day_30", "made_1k", "top_10"} {
		if !findAchievement(t, list, k).Unlocked {
			t.Errorf("%s should be granted", k)
		}
	}
	for _, k := range []string{"nw_100k", "day_60", "rain_profit", "comeback"} {
		if findAchievement(t, list, k).Unlocked {
			t.Errorf("%s cannot be proven from a stored run", k)
		}
	}
	before := list.UnlockedCount
	if again, err := BackfillAchievements(ctx, e.repo); err != nil || again != 0 {
		t.Fatalf("second backfill: %d, %v", again, err)
	}
	if e.achievements("joe12").UnlockedCount != before {
		t.Fatal("the backfill is not idempotent")
	}
}
