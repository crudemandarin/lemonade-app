package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"lemonade-api/internal/domain"
	"lemonade-api/internal/store"
)

type testEnv struct {
	t      *testing.T
	router *gin.Engine
	repo   *store.Memory
	cfg    domain.Config
}

func newEnv(t *testing.T) *testEnv {
	gin.SetMode(gin.TestMode)
	repo := store.NewMemory()
	cfg := domain.DefaultConfig()
	router := gin.New()
	NewGame(repo, cfg, func() int64 { return 42 }).Register(router)
	return &testEnv{t: t, router: router, repo: repo, cfg: cfg}
}

// do sends a request and returns the recorder. body may be nil; user may be empty.
func (e *testEnv) do(method, path, user string, body any) *httptest.ResponseRecorder {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			e.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if user != "" {
		req.Header.Set("X-Username", user)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) login(user string) {
	e.t.Helper()
	if rec := e.do("POST", "/api/login", "", map[string]string{"username": user}); rec.Code != http.StatusOK {
		e.t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body, err)
	}
	return v
}

func (e *testEnv) game(user string) gameViewDTO {
	e.t.Helper()
	rec := e.do("GET", "/api/game", user, nil)
	if rec.Code != http.StatusOK {
		e.t.Fatalf("GET /api/game: %d %s", rec.Code, rec.Body)
	}
	return decode[gameViewDTO](e.t, rec)
}

func (e *testEnv) wantError(rec *httptest.ResponseRecorder, status int, code string) {
	e.t.Helper()
	if rec.Code != status {
		e.t.Fatalf("status = %d, want %d (%s)", rec.Code, status, rec.Body)
	}
	got := decode[errorDTO](e.t, rec)
	if got.Error != code || got.Message == "" {
		e.t.Fatalf("error body = %+v, want code %q with a message", got, code)
	}
}

func TestLoginCreatesThenResumes(t *testing.T) {
	e := newEnv(t)

	first := e.do("POST", "/api/login", "", map[string]string{"username": "joe12"})
	if first.Code != http.StatusOK {
		t.Fatalf("status = %d", first.Code)
	}
	u1 := decode[userDTO](t, first)
	if u1.Username != "joe12" || u1.ID == 0 {
		t.Fatalf("user = %+v", u1)
	}

	if rec := e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 2}); rec.Code != http.StatusOK {
		t.Fatalf("buy: %s", rec.Body)
	}
	capital := e.game("joe12").Capital

	u2 := decode[userDTO](t, e.do("POST", "/api/login", "", map[string]string{"username": "joe12"}))
	if u2.ID != u1.ID {
		t.Fatal("known username created a new user")
	}
	if got := e.game("joe12").Capital; got != capital {
		t.Fatalf("login reset the game: capital %d, want %d", got, capital)
	}

	e.login("ann12")
	if got := e.game("ann12").Capital; got != 1000 {
		t.Fatalf("other user's capital = %d, want 1000", got)
	}
}

func TestLoginValidation(t *testing.T) {
	e := newEnv(t)
	e.wantError(e.do("POST", "/api/login", "", map[string]string{"username": "   "}), 400, "invalid_username")
	e.wantError(e.do("POST", "/api/login", "", map[string]string{"username": strings.Repeat("a", 41)}), 400, "invalid_username")
	e.wantError(e.do("POST", "/api/login", "", map[string]string{"username": "abcd"}), 400, "invalid_username")
	for _, bad := range []string{"jo ee", "josée", "日本語日本", "ab\tcd"} {
		e.wantError(e.do("POST", "/api/login", "", map[string]string{"username": bad}), 400, "invalid_username")
	}

	req := httptest.NewRequest("POST", "/api/login", bytes.NewBufferString("not json"))
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	e.wantError(rec, 400, "invalid_request")
}

func TestUsernameIsCaseInsensitive(t *testing.T) {
	e := newEnv(t)
	first := decode[userDTO](t, e.do("POST", "/api/login", "", map[string]string{"username": "Joe12"}))
	if first.Username != "joe12" {
		t.Fatalf("username = %q, want it lowercased", first.Username)
	}
	second := decode[userDTO](t, e.do("POST", "/api/login", "", map[string]string{"username": "JOE12"}))
	if second.ID != first.ID {
		t.Fatal("JOE and Joe are different users")
	}
	if rec := e.do("GET", "/api/game", "JoE12", nil); rec.Code != http.StatusOK {
		t.Fatalf("mixed-case X-Username header: status %d, want 200", rec.Code)
	}
}

func TestUsernameMiddleware(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")

	paths := []struct{ method, path string }{
		{"GET", "/api/game"},
		{"POST", "/api/game/new"},
		{"POST", "/api/game/buy"},
		{"POST", "/api/game/sell"},
		{"POST", "/api/game/facilities/warehouse/expand"},
		{"POST", "/api/game/facilities/production/upgrade"},
		{"POST", "/api/game/end-day"},
	}
	for _, p := range paths {
		e.wantError(e.do(p.method, p.path, "", nil), 401, "unauthorized")
		e.wantError(e.do(p.method, p.path, "stranger", nil), 401, "unauthorized")
	}
}

func TestGameViewMatchesContract(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	v := e.game("joe12")

	if v.Day != 1 || v.Capital != 1000 || v.Status != domain.StatusActive || v.UpkeepPerDay != 30 {
		t.Fatalf("view = %+v", v)
	}
	if len(v.Resources) != 5 {
		t.Fatalf("resources = %d", len(v.Resources))
	}
	wantOrder := []domain.Resource{"lemon", "sugar", "ice", "cup", "lemonade"}
	wantPrice := []int{20, 10, 10, 10, 90}
	wantBid := []int{18, 9, 9, 9, 81}
	wantAsk := []int{22, 11, 11, 11, 99}
	for i, r := range v.Resources {
		if r.Resource != wantOrder[i] || r.Price != wantPrice[i] || r.Bid != wantBid[i] || r.Ask != wantAsk[i] {
			t.Errorf("resource %d = %+v", i, r)
		}
		if r.Stock != 0 || r.Capacity != 10 || r.PreviousPrice != nil || len(r.History) != 1 {
			t.Errorf("resource %d = %+v", i, r)
		}
	}

	wh := v.Facilities.Warehouse
	if wh.TierName != "Pantry" || wh.Level != 1 || wh.MaxLevel != 4 || wh.Buildings != 5 || wh.MaxCount != 10 ||
		wh.SizePerBuilding != 10 || wh.ExpandCost != 100 || wh.UpkeepPerDay != 10 || len(wh.Resources) != 5 {
		t.Errorf("warehouse = %+v", wh)
	}
	if up := wh.Upgrade; up == nil || up.TierName != "Garage" || up.CostPerBuilding != 100 || up.TotalCost != 500 || up.UpkeepIncrease != 20 || up.SizePerBuilding != 20 {
		t.Errorf("warehouse upgrade = %+v", wh.Upgrade)
	}
	pr := v.Facilities.Production
	if pr.TierName != "Kitchen" || pr.Buildings != 1 || pr.ExpandCost != 500 || pr.UpkeepPerDay != 20 || pr.RatePerDay != 10 || pr.SizePerBuilding != 10 {
		t.Errorf("production = %+v", pr)
	}
	if up := pr.Upgrade; up == nil || up.TierName != "Food Truck" || up.TotalCost != 1000 || up.UpkeepIncrease != 30 || up.SizePerBuilding != 20 {
		t.Errorf("production upgrade = %+v", pr.Upgrade)
	}

	// Raw JSON: null previousPrice / upgrade at max level, [] (not null) for events.
	raw := e.do("GET", "/api/game", "joe12", nil).Body.String()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	if ev, ok := m["events"].([]any); !ok || len(ev) != 0 {
		t.Errorf("events = %v, want []", m["events"])
	}
	first := m["resources"].([]any)[0].(map[string]any)
	if v, present := first["previousPrice"]; !present || v != nil {
		t.Errorf("previousPrice = %v (present=%v), want null", v, present)
	}
}

func TestBuyAndSell(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")

	rec := e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 5})
	if rec.Code != 200 {
		t.Fatalf("buy: %s", rec.Body)
	}
	v := decode[gameViewDTO](t, rec)
	if v.Capital != 1000-5*22 || v.Resources[0].Stock != 5 {
		t.Fatalf("after buy: capital=%d stock=%d", v.Capital, v.Resources[0].Stock)
	}

	rec = e.do("POST", "/api/game/sell", "joe12", map[string]any{"resource": "lemon", "qty": 2})
	v = decode[gameViewDTO](t, rec)
	if v.Capital != 1000-5*22+2*18 || v.Resources[0].Stock != 3 {
		t.Fatalf("after sell: capital=%d stock=%d", v.Capital, v.Resources[0].Stock)
	}
}

func TestTradeErrors(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")

	tests := []struct {
		name   string
		path   string
		body   any
		status int
		code   string
	}{
		{"insufficient stock", "/api/game/sell", map[string]any{"resource": "lemonade", "qty": 1}, 409, "insufficient_stock"},
		{"capacity exceeded", "/api/game/buy", map[string]any{"resource": "lemon", "qty": 11}, 409, "capacity_exceeded"},
		{"zero qty", "/api/game/buy", map[string]any{"resource": "lemon", "qty": 0}, 400, "invalid_quantity"},
		{"negative qty", "/api/game/sell", map[string]any{"resource": "lemon", "qty": -3}, 400, "invalid_quantity"},
		{"unknown resource", "/api/game/buy", map[string]any{"resource": "gold", "qty": 1}, 400, "invalid_resource"},
		{"missing body", "/api/game/buy", nil, 400, "invalid_request"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e.wantError(e.do("POST", tt.path, "joe12", tt.body), tt.status, tt.code)
		})
	}
	if v := e.game("joe12"); v.Capital != 1000 {
		t.Fatalf("failed requests changed capital to %d", v.Capital)
	}
}

func TestGameViewCarriesTheTimelineAndStats(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")

	v := e.game("joe12")
	if len(v.Timeline) != 1 || v.Timeline[0].Kind != "start" || v.Timeline[0].Capital != 1000 || len(v.Timeline[0].Stock) != 5 {
		t.Fatalf("fresh timeline = %+v", v.Timeline)
	}
	if v.Stats.PeakCapital != 1000 || v.Stats.PeakDay != 1 {
		t.Fatalf("fresh stats = %+v", v.Stats)
	}

	if rec := e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 3}); rec.Code != 200 {
		t.Fatalf("buy: %s", rec.Body)
	}
	if rec := e.do("POST", "/api/game/facilities/warehouse/expand", "joe12", map[string]any{"resource": "ice"}); rec.Code != 200 {
		t.Fatalf("expand: %s", rec.Body)
	}
	rec := e.do("GET", "/api/game", "joe12", nil)
	v = decode[gameViewDTO](t, rec)
	if len(v.Timeline) != 3 {
		t.Fatalf("timeline = %+v", v.Timeline)
	}
	buy, expand := v.Timeline[1], v.Timeline[2]
	if buy.Kind != "buy" || buy.Resource != "lemon" || buy.Qty != 3 || buy.Amount != 66 || buy.Capital != 934 || buy.Stock[0] != 3 {
		t.Errorf("buy point = %+v", buy)
	}
	if expand.Kind != "expand" || expand.Facility != "warehouse" || expand.Resource != "ice" || expand.Amount != 100 || expand.Capital != 834 {
		t.Errorf("expand point = %+v", expand)
	}
	if v.Stats.CasesBought != 3 || v.Stats.Spent != 66 || v.Stats.FacilitiesBought != 1 || v.Stats.FacilitySpend != 100 {
		t.Errorf("stats = %+v", v.Stats)
	}

	// The JSON keys are the contract with the frontend.
	var raw struct {
		Timeline []map[string]any `json:"timeline"`
		Stats    map[string]any   `json:"stats"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"day", "kind", "qty", "amount", "produced", "capital", "stock"} {
		if _, ok := raw.Timeline[1][key]; !ok {
			t.Errorf("timeline point is missing %q", key)
		}
	}
	if _, ok := raw.Timeline[0]["resource"]; ok {
		t.Error("a point with no resource should omit the key")
	}
	for _, key := range []string{"casesBought", "casesSold", "spent", "earned", "facilitiesBought", "upgrades", "facilitySpend", "produced", "upkeepPaid", "peakCapital", "peakDay"} {
		if _, ok := raw.Stats[key]; !ok {
			t.Errorf("stats is missing %q", key)
		}
	}
}

func TestOlderGamesWithoutATimelineGetAStartPoint(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.setGame("joe12", func(g *domain.Game) {
		g.Timeline = nil // saved before the timeline existed
		g.Day, g.Capital = 9, 4321
	})
	v := e.game("joe12")
	if len(v.Timeline) != 1 || v.Timeline[0].Day != 9 || v.Timeline[0].Capital != 4321 {
		t.Fatalf("timeline = %+v", v.Timeline)
	}
}

func TestBuyInsufficientFunds(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.setGame("joe12", func(g *domain.Game) { g.Capital = 50 })

	e.wantError(e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemonade", "qty": 1}), 409, "insufficient_funds")
	if v := e.game("joe12"); v.Capital != 50 || v.Resources[4].Stock != 0 {
		t.Fatalf("failed buy changed the game: capital=%d stock=%d", v.Capital, v.Resources[4].Stock)
	}
}

func TestExpandAndUpgrade(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")

	rec := e.do("POST", "/api/game/facilities/warehouse/expand", "joe12", map[string]any{"resource": "ice"})
	if rec.Code != 200 {
		t.Fatalf("expand warehouse: %s", rec.Body)
	}
	v := decode[gameViewDTO](t, rec)
	if v.Capital != 900 || v.Facilities.Warehouse.Buildings != 6 || v.Resources[2].Capacity != 20 || v.Resources[0].Capacity != 10 {
		t.Fatalf("after expand: %+v", v)
	}

	rec = e.do("POST", "/api/game/facilities/production/expand", "joe12", map[string]any{})
	if rec.Code != 200 {
		t.Fatalf("expand production: %s", rec.Body)
	}
	v = decode[gameViewDTO](t, rec)
	if v.Capital != 400 || v.Facilities.Production.Buildings != 2 || v.Facilities.Production.RatePerDay != 20 {
		t.Fatalf("after production expand: %+v", v)
	}

	rec = e.do("POST", "/api/game/facilities/production/upgrade", "joe12", nil)
	e.wantError(rec, 409, "insufficient_funds") // 2 buildings x $1000

	rec = e.do("POST", "/api/game/facilities/warehouse/upgrade", "joe12", nil)
	e.wantError(rec, 409, "insufficient_funds") // 6 buildings x $100 = $600 > $400
}

func TestUpgradeSuccessAndMaxLevel(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.setGame("joe12", func(g *domain.Game) { g.Capital = 1_000_000 })

	for level := 1; level < 4; level++ {
		if rec := e.do("POST", "/api/game/facilities/warehouse/upgrade", "joe12", nil); rec.Code != 200 {
			t.Fatalf("upgrade from level %d: %s", level, rec.Body)
		}
	}
	v := e.game("joe12")
	if wh := v.Facilities.Warehouse; wh.Level != 4 || wh.TierName != "Industrial Warehouse" || wh.Upgrade != nil {
		t.Fatalf("warehouse = %+v", wh)
	}
	e.wantError(e.do("POST", "/api/game/facilities/warehouse/upgrade", "joe12", nil), 409, "max_level")
}

func TestExpandErrors(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")

	e.wantError(e.do("POST", "/api/game/facilities/garage/expand", "joe12", map[string]any{}), 400, "invalid_facility_type")
	e.wantError(e.do("POST", "/api/game/facilities/garage/upgrade", "joe12", nil), 400, "invalid_facility_type")
	e.wantError(e.do("POST", "/api/game/facilities/warehouse/expand", "joe12", map[string]any{"resource": "gold"}), 400, "invalid_resource")
	e.wantError(e.do("POST", "/api/game/facilities/warehouse/expand", "joe12", map[string]any{}), 400, "invalid_resource")

	e.setGame("joe12", func(g *domain.Game) { g.Capital = 1_000_000; g.ProductionQty = 10 })
	e.wantError(e.do("POST", "/api/game/facilities/production/expand", "joe12", map[string]any{}), 409, "max_quantity")
}

func TestEndDayLoop(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	for _, r := range []string{"lemon", "sugar", "ice", "cup"} {
		if rec := e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": r, "qty": 10}); rec.Code != 200 {
			t.Fatalf("buy %s: %s", r, rec.Body)
		}
	}

	rec := e.do("POST", "/api/game/end-day", "joe12", nil)
	if rec.Code != 200 {
		t.Fatalf("end-day: %s", rec.Body)
	}
	res := decode[endDayResponseDTO](t, rec)
	if res.Report.Day != 1 || res.Report.Produced != 10 || res.Report.IceMelted != 0 || res.Report.UpkeepPaid != 30 {
		t.Fatalf("report = %+v", res.Report)
	}
	if res.Game.Day != 2 || res.Game.Resources[4].Stock != 10 || res.Game.Resources[2].Stock != 0 {
		t.Fatalf("game = day %d lemonade %d ice %d", res.Game.Day, res.Game.Resources[4].Stock, res.Game.Resources[2].Stock)
	}
	if res.Game.Resources[0].PreviousPrice == nil {
		t.Fatal("previousPrice should be set after a day passes")
	}
	if res.Report.PriceChanges == nil || res.Report.NewEvents == nil || res.Report.ExpiredEvents == nil {
		t.Fatal("report lists must serialize as [], not null")
	}
	if got := len(res.Game.Resources[0].History); got != 2 {
		t.Fatalf("history length = %d, want 2", got)
	}

	sell := e.do("POST", "/api/game/sell", "joe12", map[string]any{"resource": "lemonade", "qty": 10})
	if sell.Code != 200 {
		t.Fatalf("sell lemonade: %s", sell.Body)
	}
}

func TestEndDayReportsForcedSale(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.setGame("joe12", func(g *domain.Game) {
		g.Capital = 0
		g.Inventory[domain.Lemonade] = 3
	})

	rec := e.do("POST", "/api/game/end-day", "joe12", nil)
	if rec.Code != 200 {
		t.Fatalf("end-day: %s", rec.Body)
	}
	res := decode[endDayResponseDTO](t, rec)
	if res.Report.ForcedSaleCases != 1 || res.Report.ForcedSaleProceeds != 81 || res.Report.Bankrupt {
		t.Fatalf("report = %+v", res.Report)
	}
	if res.Report.UpkeepPaid != 30 || res.Game.Capital != 51 || res.Game.Resources[4].Stock != 2 {
		t.Fatalf("paid=%d capital=%d lemonade=%d", res.Report.UpkeepPaid, res.Game.Capital, res.Game.Resources[4].Stock)
	}
	// The JSON keys are part of the contract with the frontend.
	var raw map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"forcedSaleCases", "forcedSaleProceeds"} {
		if _, ok := raw["report"][key]; !ok {
			t.Errorf("report is missing %q", key)
		}
	}
}

func TestBankruptcyBlocksActionsUntilNewGame(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.setGame("joe12", func(g *domain.Game) { g.Capital = 0 })

	rec := e.do("POST", "/api/game/end-day", "joe12", nil)
	if rec.Code != 200 {
		t.Fatalf("end-day: %s", rec.Body)
	}
	res := decode[endDayResponseDTO](t, rec)
	if !res.Report.Bankrupt || res.Game.Status != domain.StatusBankrupt {
		t.Fatalf("expected bankrupt: %+v", res)
	}

	for _, p := range []struct {
		path string
		body any
	}{
		{"/api/game/buy", map[string]any{"resource": "lemon", "qty": 1}},
		{"/api/game/sell", map[string]any{"resource": "lemon", "qty": 1}},
		{"/api/game/facilities/production/expand", map[string]any{}},
		{"/api/game/facilities/warehouse/upgrade", nil},
		{"/api/game/end-day", nil},
	} {
		e.wantError(e.do("POST", p.path, "joe12", p.body), 409, "game_over")
	}

	rec = e.do("POST", "/api/game/new", "joe12", nil)
	if rec.Code != 200 {
		t.Fatalf("new game: %s", rec.Body)
	}
	if v := decode[gameViewDTO](t, rec); v.Day != 1 || v.Capital != 1000 || v.Status != domain.StatusActive {
		t.Fatalf("new game view = %+v", v)
	}
	if v := e.game("joe12"); v.Status != domain.StatusActive {
		t.Fatal("new game was not persisted")
	}
}

func TestNewGameReplacesActiveGame(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 5})

	rec := e.do("POST", "/api/game/new", "joe12", nil)
	v := decode[gameViewDTO](t, rec)
	if v.Capital != 1000 || v.Resources[0].Stock != 0 {
		t.Fatalf("view = %+v", v)
	}
}

// setGame edits a stored game directly, to reach states that are slow to play into.
func (e *testEnv) setGame(user string, edit func(g *domain.Game)) {
	e.t.Helper()
	u, err := e.repo.FindUser(context.Background(), user)
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := e.repo.Mutate(context.Background(), u.ID, func(g *domain.Game) error { edit(g); return nil }); err != nil {
		e.t.Fatal(err)
	}
}

func TestGameViewCarriesTheProjection(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")

	v := e.game("joe12")
	if v.Projection.LemonadeToProduce != 0 || v.Projection.IceToMelt != 0 || v.Projection.LimitedBy != "lemon" {
		t.Fatalf("empty warehouse: %+v", v.Projection)
	}

	// 4 of each input; 3 more ice than the recipe needs would be wasted.
	for _, r := range []string{"lemon", "sugar", "cup"} {
		e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": r, "qty": 4})
	}
	rec := e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "ice", "qty": 7})
	v = decode[gameViewDTO](t, rec)
	if v.Projection.LemonadeToProduce != 4 || v.Projection.IceToMelt != 3 || v.Projection.LimitedBy != "lemon" {
		t.Fatalf("after buying: %+v", v.Projection)
	}
	if !strings.Contains(rec.Body.String(), `"projection":{"lemonadeToProduce":4,"iceToMelt":3,"limitedBy":"lemon"}`) {
		t.Fatalf("unexpected JSON shape: %s", rec.Body)
	}
}

func TestClampedTrades(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")

	// Without clamp an oversize buy fails whole.
	rec := e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 20})
	e.wantError(rec, http.StatusConflict, "capacity_exceeded")

	// With clamp it buys what fits: $1000 buys 45 at $22, but a Pantry holds 10.
	rec = e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 1_000_000, "clamp": true})
	v := decode[gameViewDTO](t, rec)
	if rec.Code != 200 || v.Resources[0].Stock != 10 || v.Capital != 1000-10*22 {
		t.Fatalf("clamped buy: %d stock=%d capital=%d", rec.Code, v.Resources[0].Stock, v.Capital)
	}
	e.wantError(e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 5, "clamp": true}), http.StatusConflict, "capacity_exceeded")

	rec = e.do("POST", "/api/game/sell", "joe12", map[string]any{"resource": "lemon", "qty": 1_000_000, "clamp": true})
	v = decode[gameViewDTO](t, rec)
	if rec.Code != 200 || v.Resources[0].Stock != 0 {
		t.Fatalf("clamped sell: %d stock=%d", rec.Code, v.Resources[0].Stock)
	}
	e.wantError(e.do("POST", "/api/game/sell", "joe12", map[string]any{"resource": "lemon", "qty": 5, "clamp": true}), http.StatusConflict, "insufficient_stock")
}

func TestSellFacility(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")

	// Every group starts at one building, so nothing is sellable yet.
	v := e.game("joe12")
	if v.Facilities.Production.CanSell || v.Facilities.Production.Reason != "min_facility" || v.Facilities.Production.SellValue != 250 {
		t.Fatalf("production sale info: %+v", v.Facilities.Production.saleDTO)
	}
	e.wantError(e.do("POST", "/api/game/facilities/production/sell", "joe12", nil), http.StatusConflict, "min_facility")
	e.wantError(e.do("POST", "/api/game/facilities/warehouse/sell", "joe12", map[string]any{"resource": "lemon"}), http.StatusConflict, "min_facility")
	e.wantError(e.do("POST", "/api/game/facilities/warehouse/sell", "joe12", map[string]any{"resource": "nope"}), http.StatusBadRequest, "invalid_resource")

	// Two lemon warehouses; fill them past what one holds.
	e.do("POST", "/api/game/facilities/warehouse/expand", "joe12", map[string]any{"resource": "lemon"})
	e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 14})
	v = e.game("joe12")
	lemon := v.Facilities.Warehouse.Resources[0]
	if lemon.CanSell || lemon.Reason != "stock_exceeds_capacity" || lemon.CasesToSell != 4 {
		t.Fatalf("lemon sale info: %+v", lemon.saleDTO)
	}
	rec := e.do("POST", "/api/game/facilities/warehouse/sell", "joe12", map[string]any{"resource": "lemon"})
	e.wantError(rec, http.StatusConflict, "stock_exceeds_capacity")
	if !strings.Contains(rec.Body.String(), "Sell 4 cases first") {
		t.Fatalf("message: %s", rec.Body)
	}

	e.do("POST", "/api/game/sell", "joe12", map[string]any{"resource": "lemon", "qty": 4})
	before := e.game("joe12")
	rec = e.do("POST", "/api/game/facilities/warehouse/sell", "joe12", map[string]any{"resource": "lemon"})
	v = decode[gameViewDTO](t, rec)
	if rec.Code != 200 || v.Capital != before.Capital+50 || v.Facilities.Warehouse.Resources[0].Count != 1 {
		t.Fatalf("sell: %d capital %d -> %d", rec.Code, before.Capital, v.Capital)
	}
	if v.UpkeepPerDay != before.UpkeepPerDay-2 || v.Stats.FacilitiesSold != 1 || v.Stats.FacilityProceeds != 50 {
		t.Fatalf("upkeep=%d stats=%+v", v.UpkeepPerDay, v.Stats)
	}
	if last := v.Timeline[len(v.Timeline)-1]; last.Kind != "facility_sold" || last.Resource != "lemon" || last.Amount != 50 {
		t.Fatalf("timeline: %+v", last)
	}
	if !strings.Contains(rec.Body.String(), `"sellBlockedReason":"min_facility"`) {
		t.Fatalf("view lacks flattened sale fields: %s", rec.Body)
	}
}

func TestGameViewShowsAverageCost(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")

	e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 4})
	v := e.game("joe12")
	lemon := v.Resources[0]
	// 4 × $22 ask = $88 → $22 each; the bid is $18, so 4 × 18 - 88 = -16.
	if lemon.AvgCost != 22 || lemon.UnrealizedGain != -16 {
		t.Fatalf("lemon: avg=%d gain=%d", lemon.AvgCost, lemon.UnrealizedGain)
	}
	if sugar := v.Resources[1]; sugar.AvgCost != 0 || sugar.UnrealizedGain != 0 {
		t.Fatalf("sugar with no stock: %+v", sugar)
	}

	// Sell everything: no stock, no cost.
	e.do("POST", "/api/game/sell", "joe12", map[string]any{"resource": "lemon", "qty": 4})
	if l := e.game("joe12").Resources[0]; l.AvgCost != 0 || l.UnrealizedGain != 0 {
		t.Fatalf("after selling out: %+v", l)
	}
}

func TestGameViewCarriesThePriceLog(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")

	v := e.game("joe12")
	if len(v.PriceLog) != 1 || v.PriceLog[0].Day != 1 || len(v.PriceLog[0].Prices) != 5 || v.PriceLog[0].Prices[4] != 90 {
		t.Fatalf("start log: %+v", v.PriceLog)
	}
	if got := v.BasePrices; len(got) != 5 || got[0] != 20 || got[4] != 90 {
		t.Fatalf("base prices: %v", got)
	}

	for i := 0; i < 3; i++ {
		e.do("POST", "/api/game/end-day", "joe12", nil)
	}
	v = e.game("joe12")
	if len(v.PriceLog) != 4 || v.PriceLog[3].Day != 4 {
		t.Fatalf("after 3 days: %+v", v.PriceLog)
	}
	if v.PriceLog[3].Prices[4] != v.Resources[4].Price {
		t.Fatalf("last logged lemonade %d, shown %d", v.PriceLog[3].Prices[4], v.Resources[4].Price)
	}
	if v.PriceLog[0].Events == nil {
		t.Fatal("events must serialise as [], not null")
	}
}
