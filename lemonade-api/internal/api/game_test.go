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
	NewGame(repo, cfg, func() int64 { return 42 }, WithDevAuth()).Register(router)
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
	e.wantError(e.do("POST", "/api/login", "", map[string]string{"username": "ab"}), 400, "invalid_username")
	e.wantError(e.do("POST", "/api/login", "", map[string]string{"username": "a"}), 400, "invalid_username")
	// Three characters is the shortest allowed, so that is accepted.
	if rec := e.do("POST", "/api/login", "", map[string]string{"username": "Joe"}); rec.Code != 200 {
		t.Fatalf("a 3-character username was refused: %d %s", rec.Code, rec.Body)
	}
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

// A run in progress cannot be thrown away: it has to be given up (and so recorded) first.
func TestNewGameIsRefusedWhileARunIsActive(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 5})
	before := e.storedGame("joe12")

	e.wantError(e.do("POST", "/api/game/new", "joe12", nil), http.StatusConflict, "run_active")
	after := e.storedGame("joe12")
	if after.RunID != before.RunID || after.Inventory[domain.Lemon] != 5 {
		t.Fatal("a refused new game changed the run")
	}

	e.do("POST", "/api/game/give-up", "joe12", nil)
	rec := e.do("POST", "/api/game/new", "joe12", nil)
	v := decode[gameViewDTO](t, rec)
	if rec.Code != 200 || v.Capital != 1000 || v.Resources[0].Stock != 0 || v.Status != domain.StatusActive {
		t.Fatalf("after giving up: %d %+v", rec.Code, v)
	}
	if e.storedGame("joe12").RunID == before.RunID {
		t.Fatal("the new game reused the old run id")
	}
}

// setGame edits a stored game directly, to reach states that are slow to play into.
func (e *testEnv) setGame(user string, edit func(g *domain.Game)) {
	e.t.Helper()
	u, err := e.repo.FindUser(context.Background(), user)
	if err != nil {
		e.t.Fatal(err)
	}
	if _, err := e.repo.Mutate(context.Background(), u.ID, func(g *domain.Game) (domain.Effects, error) { edit(g); return domain.Effects{}, nil }); err != nil {
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

func (e *testEnv) storedGame(user string) domain.Game {
	e.t.Helper()
	u, err := e.repo.FindUser(context.Background(), user)
	if err != nil {
		e.t.Fatal(err)
	}
	g, err := e.repo.GetGame(context.Background(), u.ID)
	if err != nil {
		e.t.Fatal(err)
	}
	return g
}

func TestEveryRunHasItsOwnID(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	first := e.storedGame("joe12").RunID
	if len(first) != 36 {
		t.Fatalf("run id %q is not a uuid", first)
	}
	e.do("POST", "/api/game/give-up", "joe12", nil)
	e.do("POST", "/api/game/new", "joe12", nil)
	if second := e.storedGame("joe12").RunID; second == "" || second == first {
		t.Fatalf("new game kept run id %q", second)
	}
}

func TestOldSavesGetARunIDOnTheirFirstMutation(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	u, _ := e.repo.FindUser(context.Background(), "joe12")
	old := domain.NewGame(e.cfg, 1) // NewGame leaves RunID empty, like a game saved before runs
	if _, err := e.repo.ReplaceGame(context.Background(), u.ID, old); err != nil {
		t.Fatal(err)
	}
	if e.storedGame("joe12").RunID != "" {
		t.Fatal("setup: the old save should have no run id")
	}

	// A refused action saves nothing, so it does not assign one.
	e.wantError(e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 0}), http.StatusBadRequest, "invalid_quantity")
	if e.storedGame("joe12").RunID != "" {
		t.Fatal("a failed action assigned a run id")
	}

	e.do("POST", "/api/game/end-day", "joe12", nil)
	run := e.storedGame("joe12").RunID
	if len(run) != 36 || len(e.repo.Reports(run)) != 1 {
		t.Fatalf("run id %q, reports %v", run, e.repo.Reports(run))
	}
}

func TestEndDayStoresItsReportWithTheGame(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	run := e.storedGame("joe12").RunID
	for i := 0; i < 3; i++ {
		e.do("POST", "/api/game/end-day", "joe12", nil)
	}
	reports := e.repo.Reports(run)
	if len(reports) != 3 || reports[1].Day != 1 || reports[3].Day != 3 {
		t.Fatalf("reports: %+v", reports)
	}
	if len(reports[2].PriceChanges) != 5 {
		t.Fatalf("stored report lost its prices: %+v", reports[2])
	}
}

func TestBankruptcyRecordsExactlyOneRun(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	run := e.storedGame("joe12").RunID
	e.setGame("joe12", func(g *domain.Game) { g.Capital = 0 })

	rec := e.do("POST", "/api/game/end-day", "joe12", nil)
	if rec.Code != 200 {
		t.Fatal(rec.Body)
	}
	runs := e.repo.Runs()
	if len(runs) != 1 {
		t.Fatalf("%d runs recorded, want 1", len(runs))
	}
	r := runs[0]
	if r.RunID != run || r.EndedBy != "bankrupt" || r.Days != 1 || r.Score != r.NetWorth {
		t.Fatalf("%+v", r)
	}
	if len(e.repo.Reports(run)) != 1 {
		t.Fatal("the fatal day's report should be stored too")
	}
	// The game is over: another end-day is refused and records nothing more.
	e.wantError(e.do("POST", "/api/game/end-day", "joe12", nil), http.StatusConflict, "game_over")
	if len(e.repo.Runs()) != 1 {
		t.Fatal("a second end-day recorded another run")
	}
}

func TestGameViewCarriesNetWorth(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 5})
	nw := e.game("joe12").NetWorth
	// $890 cash, 5 lemons at the $18 bid, and $500 of buildings.
	if nw.Cash != 890 || nw.Stock != 90 || nw.Facilities != 500 || nw.Total != 1480 {
		t.Fatalf("%+v", nw)
	}
}

func TestGiveUp(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	run := e.storedGame("joe12").RunID
	e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 5})
	before := e.game("joe12")

	rec := e.do("POST", "/api/game/give-up", "joe12", nil)
	v := decode[gameViewDTO](t, rec)
	if rec.Code != 200 || v.Status != "gave_up" || v.Day != before.Day || v.Capital != before.Capital {
		t.Fatalf("%d %+v", rec.Code, v)
	}

	runs := e.repo.Runs()
	if len(runs) != 1 {
		t.Fatalf("%d runs recorded, want exactly 1", len(runs))
	}
	r := runs[0]
	// The score is the final net worth: $890 cash + 5 lemons at the $18 bid + $500 of buildings.
	if r.RunID != run || r.EndedBy != "gave_up" || r.Score != 1480 || r.NetWorth != 1480 || r.Days != before.Day {
		t.Fatalf("%+v", r)
	}
	if v.NetWorth.Total != r.Score {
		t.Fatalf("view net worth %d, record score %d", v.NetWorth.Total, r.Score)
	}

	// A second give-up and every other action are refused, and nothing more is recorded.
	e.wantError(e.do("POST", "/api/game/give-up", "joe12", nil), http.StatusConflict, "game_over")
	e.wantError(e.do("POST", "/api/game/end-day", "joe12", nil), http.StatusConflict, "game_over")
	e.wantError(e.do("POST", "/api/game/buy", "joe12", map[string]any{"resource": "lemon", "qty": 1}), http.StatusConflict, "game_over")
	e.wantError(e.do("POST", "/api/game/sell", "joe12", map[string]any{"resource": "lemon", "qty": 1}), http.StatusConflict, "game_over")
	e.wantError(e.do("POST", "/api/game/facilities/production/upgrade", "joe12", nil), http.StatusConflict, "game_over")
	e.wantError(e.do("POST", "/api/game/facilities/warehouse/expand", "joe12", map[string]any{"resource": "ice"}), http.StatusConflict, "game_over")
	if len(e.repo.Runs()) != 1 {
		t.Fatal("refused actions recorded another run")
	}

	// A new game starts a fresh run that can be given up again.
	e.do("POST", "/api/game/new", "joe12", nil)
	if e.game("joe12").Status != "active" {
		t.Fatal("new game is not active")
	}
}

func TestPastDayReports(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")

	// A fresh game has no history yet: an empty list, not an error.
	rec := e.do("GET", "/api/game/reports", "joe12", nil)
	if list := decode[[]reportSummaryDTO](t, rec); rec.Code != 200 || len(list) != 0 || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("empty list: %d %s", rec.Code, rec.Body)
	}

	for i := 0; i < 3; i++ {
		e.do("POST", "/api/game/end-day", "joe12", nil)
	}
	list := decode[[]reportSummaryDTO](t, e.do("GET", "/api/game/reports", "joe12", nil))
	if len(list) != 3 || list[0].Day != 1 || list[1].Day != 2 || list[2].Day != 3 {
		t.Fatalf("list: %+v", list)
	}
	if list[0].NewEvents == nil || list[0].CapitalBefore != 1000 || list[0].CapitalAfter != 970 {
		t.Fatalf("summary: %+v", list[0])
	}

	rec = e.do("GET", "/api/game/reports/2", "joe12", nil)
	full := decode[dayReportDTO](t, rec)
	if rec.Code != 200 || full.Day != 2 || len(full.PriceChanges) != 5 || full.CapitalBefore != list[1].CapitalBefore {
		t.Fatalf("full report: %d %+v", rec.Code, full)
	}

	e.wantError(e.do("GET", "/api/game/reports/9", "joe12", nil), http.StatusNotFound, "not_found")
	e.wantError(e.do("GET", "/api/game/reports/abc", "joe12", nil), http.StatusBadRequest, "invalid_day")
	e.wantError(e.do("GET", "/api/game/reports/0", "joe12", nil), http.StatusBadRequest, "invalid_day")
}

func TestPastDayReportsOfFinishedRunsBelongToTheirPlayer(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	e.login("amy34")
	run := e.storedGame("joe12").RunID
	e.do("POST", "/api/game/end-day", "joe12", nil)
	e.do("POST", "/api/game/end-day", "joe12", nil)
	e.do("POST", "/api/game/give-up", "joe12", nil)
	e.do("POST", "/api/game/new", "joe12", nil) // joe's current run is now a different one

	if now := decode[[]reportSummaryDTO](t, e.do("GET", "/api/game/reports", "joe12", nil)); len(now) != 0 {
		t.Fatalf("the new run should have no reports yet: %+v", now)
	}
	old := decode[[]reportSummaryDTO](t, e.do("GET", "/api/game/reports?runId="+run, "joe12", nil))
	if len(old) != 2 {
		t.Fatalf("finished run: %+v", old)
	}
	if rec := e.do("GET", "/api/game/reports/1?runId="+run, "joe12", nil); rec.Code != 200 {
		t.Fatalf("one day of a finished run: %d", rec.Code)
	}

	// Another player cannot read it, and a made-up run looks the same.
	e.wantError(e.do("GET", "/api/game/reports?runId="+run, "amy34", nil), http.StatusNotFound, "not_found")
	e.wantError(e.do("GET", "/api/game/reports/1?runId="+run, "amy34", nil), http.StatusNotFound, "not_found")
	e.wantError(e.do("GET", "/api/game/reports?runId=nope", "joe12", nil), http.StatusNotFound, "not_found")
}

func TestOldSavesHaveNoPastReportsUntilTheyPlay(t *testing.T) {
	e := newEnv(t)
	e.login("joe12")
	u, _ := e.repo.FindUser(context.Background(), "joe12")
	e.repo.ReplaceGame(context.Background(), u.ID, domain.NewGame(e.cfg, 1)) // no run id
	rec := e.do("GET", "/api/game/reports", "joe12", nil)
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	e.wantError(e.do("GET", "/api/game/reports/1", "joe12", nil), http.StatusNotFound, "not_found")
}

// giveUpAfterBuying plays a run to a known score: buying n lemons costs $22 each and
// is worth $18 each at the bid, so the net worth is $1,500 - 4n.
func (e *testEnv) giveUpAfterBuying(user string, n int) {
	e.t.Helper()
	if n > 0 {
		e.do("POST", "/api/game/buy", user, map[string]any{"resource": "lemon", "qty": n})
	}
	if rec := e.do("POST", "/api/game/give-up", user, nil); rec.Code != 200 {
		e.t.Fatalf("give up: %s", rec.Body)
	}
}

func TestGlobalBoardHasOneRowPerPlayerWithTheirBestRun(t *testing.T) {
	e := newEnv(t)
	for _, u := range []string{"ann12", "bob34", "cat56"} {
		e.login(u)
	}
	e.giveUpAfterBuying("ann12", 5) // 1480
	e.do("POST", "/api/game/new", "ann12", nil)
	e.giveUpAfterBuying("ann12", 0) // 1500: her best
	e.do("POST", "/api/game/new", "ann12", nil)
	e.giveUpAfterBuying("ann12", 9) // 1464
	e.giveUpAfterBuying("bob34", 0) // 1500, finished after ann's: ranks below her
	e.giveUpAfterBuying("cat56", 10)

	rec := e.do("GET", "/api/scores", "cat56", nil)
	if rec.Code != 200 {
		t.Fatal(rec.Body)
	}
	board := decode[scoresDTO](t, rec)
	if len(board.Rows) != 3 {
		t.Fatalf("rows: %+v", board.Rows)
	}
	want := []struct {
		user  string
		score int
	}{{"ann12", 1500}, {"bob34", 1500}, {"cat56", 1460}}
	for i, w := range want {
		r := board.Rows[i]
		if r.Rank != i+1 || r.Username != w.user || r.Score != w.score || r.Days != 1 || r.CreatedAt.IsZero() {
			t.Fatalf("row %d = %+v, want %s %d", i, r, w.user, w.score)
		}
		if r.IsMe != (w.user == "cat56") {
			t.Fatalf("row %d isMe = %v", i, r.IsMe)
		}
	}
	if board.Me == nil || board.Me.Rank != 3 || board.Me.Username != "cat56" || !board.Me.IsMe {
		t.Fatalf("me: %+v", board.Me)
	}
	if strings.Contains(rec.Body.String(), "runId") {
		t.Fatal("the global board must not expose run ids")
	}
}

func TestBoardShowsMeBelowTheLimitAndNothingForARookie(t *testing.T) {
	e := newEnv(t)
	for _, u := range []string{"ann12", "bob34", "cat56", "dan78"} {
		e.login(u)
	}
	e.giveUpAfterBuying("ann12", 0)
	e.giveUpAfterBuying("bob34", 1)
	e.giveUpAfterBuying("cat56", 2)

	board := decode[scoresDTO](t, e.do("GET", "/api/scores?limit=2", "cat56", nil))
	if len(board.Rows) != 2 || board.Me == nil || board.Me.Rank != 3 {
		t.Fatalf("limit 2: rows %d, me %+v", len(board.Rows), board.Me)
	}
	// dan has no finished run: the board loads, and there is no row for him.
	if dan := decode[scoresDTO](t, e.do("GET", "/api/scores", "dan78", nil)); dan.Me != nil || len(dan.Rows) != 3 {
		t.Fatalf("dan: %+v", dan)
	}
}

func TestBoardLimitIsClampedAndValidated(t *testing.T) {
	e := newEnv(t)
	e.login("ann12")
	e.giveUpAfterBuying("ann12", 0)
	for _, q := range []string{"limit=0", "limit=-5", "limit=1000", "limit=100"} {
		if rec := e.do("GET", "/api/scores?"+q, "ann12", nil); rec.Code != 200 || len(decode[scoresDTO](t, rec).Rows) != 1 {
			t.Fatalf("%s: %d %s", q, rec.Code, rec.Body)
		}
	}
	e.wantError(e.do("GET", "/api/scores?limit=lots", "ann12", nil), http.StatusBadRequest, "invalid_limit")
}

func TestActiveRunsNeverAppearOnTheBoard(t *testing.T) {
	e := newEnv(t)
	e.login("ann12")
	e.do("POST", "/api/game/buy", "ann12", map[string]any{"resource": "lemon", "qty": 3})
	board := decode[scoresDTO](t, e.do("GET", "/api/scores", "ann12", nil))
	if len(board.Rows) != 0 || board.Me != nil {
		t.Fatalf("an unfinished run is on the board: %+v", board)
	}
}

func TestMyRunsAreNewestFirstWithTheBestFlagged(t *testing.T) {
	e := newEnv(t)
	e.login("ann12")
	e.login("bob34")
	e.giveUpAfterBuying("ann12", 5) // 1480
	e.do("POST", "/api/game/new", "ann12", nil)
	e.giveUpAfterBuying("ann12", 0) // 1500
	e.do("POST", "/api/game/new", "ann12", nil)
	e.do("POST", "/api/game/end-day", "ann12", nil)
	e.giveUpAfterBuying("ann12", 8) // day 2, 1468
	e.giveUpAfterBuying("bob34", 0)

	runs := decode[[]runSummaryDTO](t, e.do("GET", "/api/runs", "ann12", nil))
	if len(runs) != 3 {
		t.Fatalf("runs: %+v", runs)
	}
	if runs[0].Score >= 1500 || runs[0].Days != 2 || runs[1].Score != 1500 || runs[2].Score != 1480 {
		t.Fatalf("order or scores: %+v", runs)
	}
	for i, r := range runs {
		if r.IsBest != (i == 1) || r.EndedBy != "gave_up" {
			t.Fatalf("run %d: %+v", i, r)
		}
	}
	if none := decode[[]runSummaryDTO](t, e.do("GET", "/api/runs", "bob34", nil)); len(none) != 1 {
		t.Fatalf("bob sees %d runs, want only his own", len(none))
	}
}

func TestRunDetailIsForItsOwnerOnly(t *testing.T) {
	e := newEnv(t)
	e.login("ann12")
	e.login("bob34")
	e.do("POST", "/api/game/buy", "ann12", map[string]any{"resource": "lemon", "qty": 4})
	e.do("POST", "/api/game/end-day", "ann12", nil)
	e.do("POST", "/api/game/end-day", "ann12", nil)
	run := e.storedGame("ann12").RunID
	e.do("POST", "/api/game/give-up", "ann12", nil)

	rec := e.do("GET", "/api/runs/"+run, "ann12", nil)
	if rec.Code != 200 {
		t.Fatal(rec.Body)
	}
	d := decode[runDetailDTO](t, rec)
	if d.RunID != run || d.EndedBy != "gave_up" || d.Days != 3 || !d.IsBest || d.Score != d.NetWorth {
		t.Fatalf("summary: %+v", d.runSummaryDTO)
	}
	if len(d.Timeline) == 0 || len(d.PriceLog) != 3 || len(d.BasePrices) != 5 || d.Stats.CasesBought != 4 {
		t.Fatalf("history missing: timeline %d, prices %d, stats %+v", len(d.Timeline), len(d.PriceLog), d.Stats)
	}
	if len(d.Reports) != 2 || d.Reports[0].Day != 1 || d.Reports[1].Day != 2 {
		t.Fatalf("report index: %+v", d.Reports)
	}

	e.wantError(e.do("GET", "/api/runs/"+run, "bob34", nil), http.StatusNotFound, "not_found")
	e.wantError(e.do("GET", "/api/runs/nope", "ann12", nil), http.StatusNotFound, "not_found")
	// A run still in progress is not a finished run.
	e.do("POST", "/api/game/new", "ann12", nil)
	e.wantError(e.do("GET", "/api/runs/"+e.storedGame("ann12").RunID, "ann12", nil), http.StatusNotFound, "not_found")
}

func TestScoreRoutesNeedALogin(t *testing.T) {
	e := newEnv(t)
	for _, path := range []string{"/api/scores", "/api/runs", "/api/runs/x"} {
		if rec := e.do("GET", path, "", nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s without a username: %d", path, rec.Code)
		}
		if rec := e.do("GET", path, "ghost", nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s for an unknown user: %d", path, rec.Code)
		}
	}
}

func TestGameViewCarriesTheRunIDAndPersonalBest(t *testing.T) {
	e := newEnv(t)
	e.login("ann12")
	v := e.game("ann12")
	if v.Best != nil || v.RunID == "" || v.RunID != e.storedGame("ann12").RunID {
		t.Fatalf("fresh player: best %+v runId %q", v.Best, v.RunID)
	}

	e.do("POST", "/api/game/end-day", "ann12", nil)
	rec := e.do("POST", "/api/game/give-up", "ann12", nil)
	v = decode[gameViewDTO](t, rec)
	if v.Best == nil || v.Best.RunID != v.RunID || v.Best.Score != v.NetWorth.Total || v.Best.Days != 2 {
		t.Fatalf("the finished run is the first personal best: %+v runId %q", v.Best, v.RunID)
	}

	// A worse next run leaves the best where it was; the callout compares run ids.
	e.do("POST", "/api/game/new", "ann12", nil)
	e.do("POST", "/api/game/buy", "ann12", map[string]any{"resource": "lemon", "qty": 9})
	first := v.Best
	rec = e.do("POST", "/api/game/give-up", "ann12", nil)
	v = decode[gameViewDTO](t, rec)
	if v.Best.RunID != first.RunID || v.Best.RunID == v.RunID {
		t.Fatalf("a worse run replaced the best: %+v", v.Best)
	}
}
