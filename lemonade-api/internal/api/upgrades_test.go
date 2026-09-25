package api

import (
	"context"
	"net/http"
	"testing"

	"lemonade-api/internal/domain"
)

func (e *testEnv) upgrades(user string) upgradesResponseDTO {
	e.t.Helper()
	rec := e.do("GET", "/api/game/upgrades", user, nil)
	if rec.Code != http.StatusOK {
		e.t.Fatalf("GET upgrades: %d %s", rec.Code, rec.Body)
	}
	return decode[upgradesResponseDTO](e.t, rec)
}

func upgradeByKey(t *testing.T, r upgradesResponseDTO, key string) upgradeDTO {
	t.Helper()
	for _, u := range r.Upgrades {
		if u.Key == key {
			return u
		}
	}
	t.Fatalf("no upgrade %q", key)
	return upgradeDTO{}
}

func TestUpgradesListShowsStatesAndReasons(t *testing.T) {
	e := newEnv(t)
	e.login("ada")
	r := e.upgrades("ada")
	if r.Era != 1 || len(r.Upgrades) != len(e.cfg.Upgrades) || len(r.Categories) == 0 {
		t.Fatalf("list: %+v", r)
	}
	if u := upgradeByKey(t, r, "order_book"); u.State != "available" || u.Cost != 500 {
		t.Errorf("order_book: %+v", u)
	}
	u := upgradeByKey(t, r, "freezer_2")
	if u.State != "locked" || u.LockCode != "era" || u.LockedReason != "Reach era 2 first." {
		t.Errorf("freezer_2: %+v", u)
	}
}

func TestBuyUpgradeEndpoint(t *testing.T) {
	e := newEnv(t)
	e.login("ada")

	rec := e.do("POST", "/api/game/upgrades/order_book/buy", "ada", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("buy: %d %s", rec.Code, rec.Body)
	}
	v := decode[gameViewDTO](t, rec)
	if v.Capital != 500 || len(v.Features) != 1 || v.Features[0] != "repeat_trades" {
		t.Errorf("view after buy: capital %d features %v", v.Capital, v.Features)
	}
	r := e.upgrades("ada")
	if upgradeByKey(t, r, "order_book").State != "owned" || r.OwnedCount != 1 || r.Spent != 500 {
		t.Errorf("list after buy: %+v", r)
	}

	e.wantError(e.do("POST", "/api/game/upgrades/order_book/buy", "ada", nil), http.StatusConflict, "upgrade_owned")
	e.wantError(e.do("POST", "/api/game/upgrades/freezer_2/buy", "ada", nil), http.StatusConflict, "upgrade_locked")
	e.wantError(e.do("POST", "/api/game/upgrades/nope/buy", "ada", nil), http.StatusNotFound, "unknown_upgrade")
	e.wantError(e.do("POST", "/api/game/upgrades/citrus_press/buy", "ada", nil), http.StatusConflict, "insufficient_funds")
	e.wantError(e.do("POST", "/api/game/upgrades/order_book/buy", "", nil), http.StatusUnauthorized, "unauthorized")
}

func TestUpgradeEffectsShowInTheViewAndReport(t *testing.T) {
	e := newEnv(t)
	e.login("ada")
	user, _ := e.repo.FindUser(context.Background(), "ada")
	if _, err := e.repo.Mutate(context.Background(), user.ID, func(g *domain.Game) (domain.Effects, error) {
		g.Capital = 50_000
		for _, k := range []string{"freezer_1", "bookkeeper", "market_analyst", "weather_radio"} {
			g.Upgrades[k] = 1
		}
		g.Inventory[domain.Ice], g.WarehouseQty[domain.Ice] = 5, 3
		return domain.Effects{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	v := e.game("ada")
	if v.IceKeepCases != 20 || v.Projection.IceKept != 5 || v.Projection.IceToMelt != 0 {
		t.Errorf("freezer in view: keep %d projection %+v", v.IceKeepCases, v.Projection)
	}
	if v.Resources[0].MovingAverage == nil {
		t.Error("market analyst should add a moving average")
	}
	rec := e.do("POST", "/api/game/end-day", "ada", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("end day: %d %s", rec.Code, rec.Body)
	}
	resp := decode[endDayResponseDTO](t, rec)
	if resp.Report.IceKept != 5 || resp.Report.UpgradeUpkeep != 35 || resp.Report.Pnl == nil {
		t.Errorf("report: kept %d upgrade upkeep %d pnl %v", resp.Report.IceKept, resp.Report.UpgradeUpkeep, resp.Report.Pnl)
	}
}

func TestViewWithoutUpgradesHasNoFeatures(t *testing.T) {
	e := newEnv(t)
	e.login("ada")
	v := e.game("ada")
	if v.Features == nil || len(v.Features) != 0 || v.Forecast == nil || v.Resources[0].MovingAverage != nil {
		t.Errorf("features %v forecast %v", v.Features, v.Forecast)
	}
}
