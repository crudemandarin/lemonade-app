package api

import (
	"net/http"
	"testing"

	"lemonade-api/internal/domain"
)

func TestRecipesAreListedAndLocksExplained(t *testing.T) {
	e := newEnv(t)
	e.login("rec1")
	v := e.game("rec1")
	byKey := map[string]recipeDTO{}
	for _, r := range v.Recipes {
		byKey[r.Key] = r
	}
	if byKey["lemonade"].State != "known" || byKey["limeade"].State != "available" {
		t.Fatalf("lemonade known, limeade available: %+v %+v", byKey["lemonade"], byKey["limeade"])
	}
	mint := byKey["mint_lemonade"]
	if mint.State != "locked" || mint.LockCode != "era" || mint.LockedReason != "Reach era 2 first" {
		t.Fatalf("mint lemonade in era 1: %+v", mint)
	}
	if sb := byKey["strawberry_lemonade"]; sb.State != "locked" || sb.LockedReason != "Reach era 3 first" {
		t.Fatalf("strawberry lemonade: %+v", sb)
	}
	if len(v.Plan) != 1 || v.Plan[0].Recipe != "lemonade" || v.Plan[0].Target != 0 {
		t.Fatalf("default plan: %+v", v.Plan)
	}
	for _, r := range v.Resources {
		known := r.Resource == "lemon" || r.Resource == "sugar" || r.Resource == "ice" || r.Resource == "cup" || r.Resource == "lemonade"
		if r.Unlocked != known {
			t.Fatalf("%s unlocked = %v, want %v", r.Resource, r.Unlocked, known)
		}
	}
}

func TestLearnARecipeThenBuyItsGoodsAndPlanIt(t *testing.T) {
	e := newEnv(t)
	e.login("rec2")
	e.wantError(e.do("POST", "/api/game/buy", "rec2", map[string]any{"resource": "lime", "qty": 1}), http.StatusConflict, "commodity_locked")
	e.wantError(e.do("POST", "/api/game/recipes/mint_lemonade/learn", "rec2", nil), http.StatusConflict, "recipe_locked")
	e.wantError(e.do("POST", "/api/game/recipes/nope/learn", "rec2", nil), http.StatusNotFound, "unknown_recipe")
	e.setGame("rec2", func(g *domain.Game) { g.Capital = 500 })
	e.wantError(e.do("POST", "/api/game/recipes/limeade/learn", "rec2", nil), http.StatusConflict, "insufficient_funds")
	e.setGame("rec2", func(g *domain.Game) { g.Capital = 5000 })

	rec := e.do("POST", "/api/game/recipes/limeade/learn", "rec2", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("learn: %d %s", rec.Code, rec.Body)
	}
	v := decode[gameViewDTO](t, rec)
	if v.Capital != 4200 {
		t.Fatalf("capital %d, want 5000 - 800", v.Capital)
	}
	e.wantError(e.do("POST", "/api/game/recipes/limeade/learn", "rec2", nil), http.StatusConflict, "recipe_known")
	if rec := e.do("POST", "/api/game/buy", "rec2", map[string]any{"resource": "lime", "qty": 1}); rec.Code != http.StatusOK {
		t.Fatalf("buy limes after learning: %d %s", rec.Code, rec.Body)
	}

	rec = e.do("POST", "/api/game/production-plan", "rec2", map[string]any{"rows": []map[string]any{{"recipe": "limeade", "target": 20}, {"recipe": "lemonade", "target": 0}}})
	if rec.Code != http.StatusOK {
		t.Fatalf("plan: %d %s", rec.Code, rec.Body)
	}
	v = decode[gameViewDTO](t, rec)
	if len(v.Plan) != 2 || v.Plan[0].Recipe != "limeade" || v.Plan[0].Target != 20 || len(v.Projection.Plan) != 2 {
		t.Fatalf("plan in view: %+v projection %+v", v.Plan, v.Projection.Plan)
	}
	e.wantError(e.do("POST", "/api/game/production-plan", "rec2", map[string]any{"rows": []map[string]any{{"recipe": "mint_lemonade"}}}), http.StatusBadRequest, "invalid_plan")
	e.wantError(e.do("POST", "/api/game/production-plan", "rec2", map[string]any{"rows": []map[string]any{{"recipe": "lemonade"}, {"recipe": "lemonade"}}}), http.StatusBadRequest, "invalid_plan")

	// An empty list restores the default plan.
	rec = e.do("POST", "/api/game/production-plan", "rec2", map[string]any{"rows": []map[string]any{}})
	if v = decode[gameViewDTO](t, rec); len(v.Plan) != 1 || v.Plan[0].Recipe != "lemonade" {
		t.Fatalf("default plan after clearing: %+v", v.Plan)
	}
}

func TestPerishablesShowInTheProjectionAndTheReport(t *testing.T) {
	e := newEnv(t)
	e.login("rec3")
	e.setGame("rec3", func(g *domain.Game) {
		g.Capital = 100_000
		g.Recipes["limeade"] = true
	})
	e.do("POST", "/api/game/buy", "rec3", map[string]any{"resource": "lime", "qty": 5})
	var lime resourceViewDTO
	for _, r := range e.game("rec3").Resources {
		if r.Resource == "lime" {
			lime = r
		}
	}
	if lime.ShelfDays != 3 {
		t.Fatalf("limes keep 3 days: %+v", lime)
	}
	// Two nights pass with nothing using them; on the third the limes go off.
	for i := 0; i < 2; i++ {
		rec := e.do("POST", "/api/game/end-day", "rec3", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("end day: %d %s", rec.Code, rec.Body)
		}
	}
	v := e.game("rec3")
	if v.Projection.WillSpoil["lime"] != 5 {
		t.Fatalf("the projection should warn that 5 limes will spoil: %+v", v.Projection.WillSpoil)
	}
	rec := e.do("POST", "/api/game/end-day", "rec3", nil)
	out := decode[endDayResponseDTO](t, rec)
	if out.Report.Spoiled["lime"] != 5 {
		t.Fatalf("the report should say 5 limes spoiled: %+v", out.Report.Spoiled)
	}
}
