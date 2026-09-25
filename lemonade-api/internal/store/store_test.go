package store

import (
	"context"
	"errors"
	"os"
	"reflect"
	"sync"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"lemonade-api/internal/domain"
)

// repoContract runs against every Repository implementation.
func repoContract(t *testing.T, repo Repository, username string) {
	t.Helper()
	ctx := context.Background()
	cfg := domain.DefaultConfig()

	if _, err := repo.FindUser(ctx, username); !errors.Is(err, ErrNotFound) {
		t.Fatalf("FindUser before create: %v", err)
	}

	game := domain.NewGame(cfg, 7)
	// Play a little so the timeline and stats have something to round-trip.
	for _, act := range []func() error{
		func() error { return domain.Buy(&game, cfg, domain.Lemon, 2) },
		func() error { return domain.Expand(&game, cfg, domain.Warehouse, domain.Ice) },
		func() error { return domain.Sell(&game, cfg, domain.Lemon, 1) },
	} {
		if err := act(); err != nil {
			t.Fatal(err)
		}
	}
	if len(game.Timeline) != 4 || game.Stats.CasesBought != 2 {
		t.Fatalf("test setup: timeline=%d stats=%+v", len(game.Timeline), game.Stats)
	}
	game.Capital = 50_000
	if err := domain.BuyUpgrade(&game, cfg, "freezer_1"); err != nil {
		t.Fatal(err)
	}
	game.IceOld, game.Carry["yield:lemonade"] = 7, 0.25
	game.Events = []domain.ActiveEvent{{
		Key: "heat_wave", Name: "Heat Wave", DaysLeft: 2,
		Multipliers: map[domain.Resource]float64{domain.Lemonade: 1.4},
	}}
	user, err := repo.CreateUserWithGame(ctx, username, game)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID == 0 || user.Username != username {
		t.Fatalf("user = %+v", user)
	}

	found, err := repo.FindUser(ctx, username)
	if err != nil || found != user {
		t.Fatalf("FindUser = %+v, %v", found, err)
	}

	got, err := repo.GetGame(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, game) {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", got, game)
	}

	// Mutate saves changes; an error from fn saves nothing.
	if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) (domain.Effects, error) {
		g.Capital = 123
		return domain.Effects{}, errors.New("boom")
	}); err == nil {
		t.Fatal("expected fn error to propagate")
	}
	if g, _ := repo.GetGame(ctx, user.ID); g.Capital != game.Capital {
		t.Fatalf("failed mutate was saved: capital = %d, want %d", g.Capital, game.Capital)
	}
	if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) (domain.Effects, error) {
		g.Capital = 123
		return domain.Effects{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if g, _ := repo.GetGame(ctx, user.ID); g.Capital != 123 {
		t.Fatalf("capital = %d, want 123", g.Capital)
	}

	// Concurrent mutations must serialize: no lost updates.
	var wg sync.WaitGroup
	const n = 20
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) (domain.Effects, error) {
				g.Capital++
				return domain.Effects{}, nil
			}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if g, _ := repo.GetGame(ctx, user.ID); g.Capital != 123+n {
		t.Fatalf("capital = %d, want %d (lost update)", g.Capital, 123+n)
	}

	// ReplaceGame overwrites.
	fresh := domain.NewGame(cfg, 99)
	if _, err := repo.ReplaceGame(ctx, user.ID, fresh); err != nil {
		t.Fatal(err)
	}
	if g, _ := repo.GetGame(ctx, user.ID); g.Seed != 99 || g.Capital != 1000 {
		t.Fatalf("replace failed: %+v", g)
	}

	// A second user gets a separate game; creating an existing user is idempotent.
	other, err := repo.CreateUserWithGame(ctx, username+"-2", domain.NewGame(cfg, 1))
	if err != nil || other.ID == user.ID {
		t.Fatalf("other = %+v, %v", other, err)
	}
	again, err := repo.CreateUserWithGame(ctx, username, domain.NewGame(cfg, 5))
	if err != nil || again.ID != user.ID {
		t.Fatalf("re-create = %+v, %v", again, err)
	}
	if g, _ := repo.GetGame(ctx, user.ID); g.Seed != 99 {
		t.Fatal("re-creating an existing user replaced their game")
	}

	if _, err := repo.GetGame(ctx, 99999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetGame unknown: %v", err)
	}
	if _, err := repo.Mutate(ctx, 99999, func(*domain.Game) (domain.Effects, error) { return domain.Effects{}, nil }); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Mutate unknown: %v", err)
	}
}

func TestMemoryRepository(t *testing.T) {
	repoContract(t, NewMemory(), "memuser")
}

func TestPostgresRepository(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgres(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		db.Exec("DELETE FROM games WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'pguser%')")
		db.Exec("DELETE FROM users WHERE username LIKE 'pguser%'")
	}
	cleanup()
	t.Cleanup(cleanup)
	repoContract(t, repo, "pguser")
}

// A row written before cost basis existed has NULL in that column. It must load,
// with a basis seeded from the stock and current prices, and save back cleanly.
func TestPostgresLoadsAnOldShapeRow(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgres(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		db.Exec("DELETE FROM games WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'pgold%')")
		db.Exec("DELETE FROM users WHERE username LIKE 'pgold%'")
	}
	cleanup()
	t.Cleanup(cleanup)

	ctx := context.Background()
	cfg := domain.DefaultConfig()
	game := domain.NewGame(cfg, 5)
	if err := domain.Buy(&game, cfg, domain.Lemon, 3); err != nil {
		t.Fatal(err)
	}
	user, err := repo.CreateUserWithGame(ctx, "pgold1", game)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE games SET cost_basis = NULL, price_log = NULL, buy_pressure = NULL, sell_pressure = NULL, upgrades = NULL, upgrade_spend = NULL, ice_old = NULL, carry = NULL WHERE user_id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}

	got, err := repo.GetGame(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CostBasis[domain.Lemon] != 3*20 { // 3 cases at the walked price of $20
		t.Fatalf("seeded basis = %v", got.CostBasis)
	}
	if len(got.BuyPressure) != 0 || len(got.SellPressure) != 0 {
		t.Fatalf("an old row should load with no pressure: %v %v", got.BuyPressure, got.SellPressure)
	}
	if got.Upgrades == nil || len(got.Upgrades) != 0 || got.IceOld != 0 || got.UpgradeSpend != 0 || len(got.Carry) != 0 {
		t.Fatalf("an old row should load with no upgrades: %+v %d", got.Upgrades, got.IceOld)
	}
	if len(got.PriceLog) != 1 || got.PriceLog[0].Day != 1 {
		t.Fatalf("seeded price log = %+v", got.PriceLog)
	}
	if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) (domain.Effects, error) {
		return domain.Effects{}, domain.Sell(g, cfg, domain.Lemon, 1)
	}); err != nil {
		t.Fatal(err)
	}
	if after, _ := repo.GetGame(ctx, user.ID); after.CostBasis[domain.Lemon] != 40 {
		t.Fatalf("basis after a sale = %v", after.CostBasis)
	}
}

// A row saved before territories has NULL there. It loads as the Neighborhood at 40% with
// its three catalog rivals, and an empire round-trips exactly.
func TestPostgresRoundTripsTheEmpire(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgres(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		db.Exec("DELETE FROM games WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'pgemp%')")
		db.Exec("DELETE FROM users WHERE username LIKE 'pgemp%'")
	}
	cleanup()
	t.Cleanup(cleanup)

	ctx := context.Background()
	cfg := domain.DefaultConfig()
	game := domain.NewGame(cfg, 7)
	game.Territories["city"] = domain.TerritoryState{Entered: true, Share: 12.5, CampaignDaysLeft: 3, CampaignBonus: 0.1}
	game.Rivals["zest_express"] = domain.RivalState{Share: 20, Valuation: 51234.5, Status: domain.RivalActive, Mood: domain.MoodHostile, TelegraphKey: "price_war", TelegraphDay: 9}
	game.Rivals["lil_lucy"] = domain.RivalState{Share: 0, Status: domain.RivalAcquired, PricePaid: 2500, Hostile: true}
	user, err := repo.CreateUserWithGame(ctx, "pgemp1", game)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetGame(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Territories["city"] != game.Territories["city"] || got.Rivals["zest_express"] != game.Rivals["zest_express"] || got.Rivals["lil_lucy"] != game.Rivals["lil_lucy"] {
		t.Fatalf("empire did not round-trip: %+v %+v", got.Territories, got.Rivals)
	}

	if err := db.Exec("UPDATE games SET territories = NULL, rivals = NULL WHERE user_id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	old, err := repo.GetGame(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if nh := old.Territories["neighborhood"]; !nh.Entered || nh.Share != 40 || len(old.Territories) != 1 {
		t.Fatalf("old row territories = %+v", old.Territories)
	}
	if len(old.Rivals) != 3 || old.Rivals["sour_sam"].Share != 25 || old.Rivals["lil_lucy"].Valuation != 2500 {
		t.Fatalf("old row rivals = %+v", old.Rivals)
	}
	if domain.Era(old, cfg) != 1 {
		t.Fatal("an old save is era 1")
	}
}

func TestPostgresSavesEffectsWithTheGame(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgres(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		db.Exec("DELETE FROM day_reports WHERE run_id LIKE 'pgfx-%'")
		db.Exec("DELETE FROM runs WHERE run_id LIKE 'pgfx-%'")
		db.Exec("DELETE FROM games WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'pgfx%')")
		db.Exec("DELETE FROM users WHERE username LIKE 'pgfx%'")
	}
	cleanup()
	t.Cleanup(cleanup)

	ctx := context.Background()
	cfg := domain.DefaultConfig()
	game := domain.NewGame(cfg, 3)
	game.RunID = "pgfx-run-1"
	user, err := repo.CreateUserWithGame(ctx, "pgfx1", game)
	if err != nil {
		t.Fatal(err)
	}
	count := func(table string) (n int64) {
		db.Table(table).Where("run_id = ?", "pgfx-run-1").Count(&n)
		return n
	}

	// Ending a day stores its report; doing it twice replaces, never duplicates.
	endDay := func(g *domain.Game) (domain.Effects, error) {
		report, err := domain.EndDay(g, cfg)
		return domain.Effects{Report: &report}, err
	}
	if _, err := repo.Mutate(ctx, user.ID, endDay); err != nil {
		t.Fatal(err)
	}
	if count("day_reports") != 1 {
		t.Fatalf("day_reports = %d", count("day_reports"))
	}

	// An error from fn saves neither the game nor its effects.
	if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) (domain.Effects, error) {
		report, _ := domain.EndDay(g, cfg)
		return domain.Effects{Report: &report}, errors.New("boom")
	}); err == nil {
		t.Fatal("expected the error")
	}
	if count("day_reports") != 1 {
		t.Fatalf("a failed mutation wrote a report: %d", count("day_reports"))
	}

	// A finished run is stored once, even if the same record arrives twice.
	finish := func(g *domain.Game) (domain.Effects, error) {
		rec := domain.FinishRun(*g, cfg, domain.EndedByGaveUp)
		return domain.Effects{Finished: &rec}, nil
	}
	for i := 0; i < 2; i++ {
		if _, err := repo.Mutate(ctx, user.ID, finish); err != nil {
			t.Fatal(err)
		}
	}
	var run runRow
	if err := db.Where("run_id = ?", "pgfx-run-1").First(&run).Error; err != nil {
		t.Fatal(err)
	}
	if count("runs") != 1 || run.UserID != user.ID || run.EndedBy != "gave_up" || run.Difficulty != 3 ||
		run.Score != run.NetWorth || len(run.Timeline) == 0 || len(run.PriceLog) == 0 {
		t.Fatalf("run row: %+v", run)
	}

	// Reports and finished runs read back, and only for their owner.
	if list, err := repo.ListReports(ctx, "pgfx-run-1"); err != nil || len(list) != 1 || list[0].Day != 1 || len(list[0].PriceChanges) != 5 {
		t.Fatalf("ListReports: %v %+v", err, list)
	}
	if r, err := repo.GetReport(ctx, "pgfx-run-1", 1); err != nil || r.Day != 1 {
		t.Fatalf("GetReport: %v %+v", err, r)
	}
	if _, err := repo.GetReport(ctx, "pgfx-run-1", 2); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing day: %v", err)
	}
	if own, _ := repo.RunBelongsTo(ctx, user.ID, "pgfx-run-1"); !own {
		t.Fatal("run should belong to its player")
	}
	if own, _ := repo.RunBelongsTo(ctx, user.ID+1000, "pgfx-run-1"); own {
		t.Fatal("run should not belong to someone else")
	}
	if list, err := repo.ListReports(ctx, "pgfx-nothing"); err != nil || list == nil || len(list) != 0 {
		t.Fatalf("empty list: %v %v", err, list)
	}

	// A game row written before runs existed has no run id and still loads.
	if err := db.Exec("UPDATE games SET run_id = NULL WHERE user_id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if g, err := repo.GetGame(ctx, user.ID); err != nil || g.RunID != "" {
		t.Fatalf("old row: %v, run id %q", err, g.RunID)
	}
}

func TestMemoryKeepsEffectsToo(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	cfg := domain.DefaultConfig()
	game := domain.NewGame(cfg, 3)
	game.RunID = "mem-run"
	user, _ := m.CreateUserWithGame(ctx, "memfx", game)
	for i := 0; i < 2; i++ {
		if _, err := m.Mutate(ctx, user.ID, func(g *domain.Game) (domain.Effects, error) {
			rec := domain.FinishRun(*g, cfg, domain.EndedByGaveUp)
			return domain.Effects{Finished: &rec}, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	if len(m.Runs()) != 1 {
		t.Fatalf("%d runs", len(m.Runs()))
	}
}

// scoresContract runs against every Repository. Scores are huge so the test's rows
// outrank anything already in a shared database.
func scoresContract(t *testing.T, repo Repository, prefix string) {
	t.Helper()
	ctx := context.Background()
	cfg := domain.DefaultConfig()
	const base = 9_000_000

	users := map[string]domain.User{}
	for _, name := range []string{"ann", "bob", "cat", "dan"} {
		g := domain.NewGame(cfg, 1)
		g.RunID = prefix + "-play-" + name
		u, err := repo.CreateUserWithGame(ctx, prefix+name, g)
		if err != nil {
			t.Fatal(err)
		}
		users[name] = u
	}
	finish := func(name, runID string, score, days int, endedBy string) {
		t.Helper()
		rec := domain.RunRecord{
			RunID: prefix + "-" + runID, Days: days, Score: base + score, NetWorth: base + score,
			Capital: 100, EndedBy: endedBy,
			Timeline: []domain.TimelinePoint{{Day: 1, Kind: domain.PointStart, Capital: 1000}},
			PriceLog: []domain.PricePoint{{Day: 1, Prices: map[domain.Resource]int{domain.Lemon: 20, domain.Sugar: 10, domain.Ice: 10, domain.Cup: 10, domain.Lemonade: 90}}},
		}
		if _, err := repo.Mutate(ctx, users[name].ID, func(g *domain.Game) (domain.Effects, error) {
			return domain.Effects{Finished: &rec}, nil
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Nothing finished yet: no rows for them, and no personal best.
	if best, err := repo.BestScore(ctx, users["ann"].ID); err != nil || best != nil {
		t.Fatalf("best before any run: %v %v", best, err)
	}
	if runs, err := repo.UserRuns(ctx, users["ann"].ID); err != nil || runs == nil || len(runs) != 0 {
		t.Fatalf("runs before any run: %v %v", runs, err)
	}

	// ann has three runs (her best is 500); bob ties her best but finished later; cat is lower.
	finish("ann", "a1", 300, 5, domain.EndedByBankrupt)
	finish("ann", "a2", 500, 9, domain.EndedByGaveUp)
	finish("ann", "a3", 100, 3, domain.EndedByBankrupt)
	finish("bob", "b1", 500, 12, domain.EndedByGaveUp)
	finish("cat", "c1", 200, 4, domain.EndedByBankrupt)

	top, err := repo.TopScores(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) < 3 {
		t.Fatalf("top = %+v", top)
	}
	// One row per user (their best), ranked by score, and the earlier finish wins the tie.
	want := []struct {
		name  string
		score int
		days  int
	}{{"ann", 500, 9}, {"bob", 500, 12}, {"cat", 200, 4}}
	for i, w := range want {
		row := top[i]
		if row.Rank != i+1 || row.Username != prefix+w.name || row.Score != base+w.score || row.Days != w.days {
			t.Fatalf("row %d = %+v, want %s %d on day %d", i, row, w.name, w.score, w.days)
		}
	}
	seen := map[string]bool{}
	for _, r := range top {
		if seen[r.Username] {
			t.Fatalf("%s appears twice", r.Username)
		}
		seen[r.Username] = true
	}

	// The limit cuts the list but not a user's own rank.
	if two, _ := repo.TopScores(ctx, 2); len(two) != 2 {
		t.Fatalf("limit 2 returned %d", len(two))
	}
	me, err := repo.BestScore(ctx, users["cat"].ID)
	if err != nil || me == nil || me.Rank != 3 || me.Score != base+200 || me.Username != prefix+"cat" {
		t.Fatalf("cat's best = %+v, %v", me, err)
	}
	if me, _ := repo.BestScore(ctx, users["ann"].ID); me == nil || me.Rank != 1 || me.RunID != prefix+"-a2" {
		t.Fatalf("ann's best = %+v", me)
	}
	if best, _ := repo.BestScore(ctx, users["dan"].ID); best != nil {
		t.Fatalf("dan has no finished run: %+v", best)
	}

	// A user's history is newest first and complete.
	runs, err := repo.UserRuns(ctx, users["ann"].ID)
	if err != nil || len(runs) != 3 {
		t.Fatalf("ann's runs: %v %+v", err, runs)
	}
	if runs[0].RunID != prefix+"-a3" || runs[1].RunID != prefix+"-a2" || runs[2].RunID != prefix+"-a1" {
		t.Fatalf("order: %+v", runs)
	}
	if runs[1].EndedBy != domain.EndedByGaveUp || runs[1].Days != 9 || runs[1].Score != base+500 {
		t.Fatalf("summary: %+v", runs[1])
	}

	// One run in full, for its owner only.
	run, err := repo.GetRun(ctx, users["ann"].ID, prefix+"-a2")
	if err != nil || run.Score != base+500 || len(run.Timeline) != 1 || len(run.PriceLog) != 1 || run.CreatedAt.IsZero() {
		t.Fatalf("run detail: %v %+v", err, run)
	}
	if _, err := repo.GetRun(ctx, users["bob"].ID, prefix+"-a2"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another user's run: %v", err)
	}
	if _, err := repo.GetRun(ctx, users["ann"].ID, prefix+"-missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown run: %v", err)
	}
}

func TestMemoryScores(t *testing.T) {
	scoresContract(t, NewMemory(), "mem")
}

func TestPostgresScores(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgres(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		db.Exec("DELETE FROM runs WHERE run_id LIKE 'pgsc-%'")
		db.Exec("DELETE FROM games WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'pgsc%')")
		db.Exec("DELETE FROM users WHERE username LIKE 'pgsc%'")
	}
	cleanup()
	t.Cleanup(cleanup)
	scoresContract(t, repo, "pgsc")
}

// The market remembers what the player traded, so it has to survive a save.
func TestPostgresKeepsPricePressure(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgres(db)
	if err := repo.Migrate(); err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		db.Exec("DELETE FROM games WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'pgpr%')")
		db.Exec("DELETE FROM users WHERE username LIKE 'pgpr%'")
	}
	cleanup()
	t.Cleanup(cleanup)

	ctx := context.Background()
	game := domain.NewGame(domain.DefaultConfig(), 2)
	game.BuyPressure[domain.Lemon] = 12.5
	game.SellPressure[domain.Lemonade] = 3
	user, err := repo.CreateUserWithGame(ctx, "pgpr1", game)
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetGame(ctx, user.ID)
	if err != nil || got.BuyPressure[domain.Lemon] != 12.5 || got.SellPressure[domain.Lemonade] != 3 {
		t.Fatalf("pressure after a round trip: %v %v (%v)", got.BuyPressure, got.SellPressure, err)
	}
}
