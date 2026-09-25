package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"lemonade-api/internal/domain"
)

// achievementsContract runs against every Repository.
func achievementsContract(t *testing.T, repo Repository, prefix string) {
	t.Helper()
	ctx := context.Background()
	cfg := domain.DefaultConfig()

	game := domain.NewGame(cfg, 4)
	game.RunID = prefix + "-run-1"
	user, err := repo.CreateUserWithGame(ctx, prefix+"ach", game)
	if err != nil {
		t.Fatal(err)
	}
	if list, err := repo.ListAchievements(ctx, user.ID); err != nil || list == nil || len(list) != 0 {
		t.Fatalf("before any unlock: %v %v", list, err)
	}

	// A failed mutation grants nothing.
	if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) (domain.Effects, error) {
		return domain.Effects{Unlocked: []string{"day_7"}}, errors.New("boom")
	}); err == nil {
		t.Fatal("expected the error")
	}
	if list, _ := repo.ListAchievements(ctx, user.ID); len(list) != 0 {
		t.Fatalf("a failed mutation granted %v", list)
	}

	// A successful one stores them with the run; granting again is a no-op.
	for i := 0; i < 2; i++ {
		if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) (domain.Effects, error) {
			return domain.Effects{Unlocked: []string{"day_7", "nw_5k"}}, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	list, err := repo.ListAchievements(ctx, user.ID)
	if err != nil || len(list) != 2 {
		t.Fatalf("after unlocking: %v %+v", err, list)
	}
	for _, a := range list {
		if a.RunID != prefix+"-run-1" || a.UnlockedAt.IsZero() {
			t.Fatalf("unlock row %+v", a)
		}
	}

	// Grants outside a mutation (the backfill) keep what is there and count only new ones.
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	n, err := repo.GrantAchievements(ctx, user.ID, []Achievement{
		{Key: "day_7", RunID: "other", UnlockedAt: when},
		{Key: "runs_5", RunID: prefix + "-old", UnlockedAt: when},
	})
	if err != nil || n != 1 {
		t.Fatalf("grant: %d %v", n, err)
	}
	list, _ = repo.ListAchievements(ctx, user.ID)
	if len(list) != 3 || list[0].Key != "runs_5" || !list[0].UnlockedAt.Equal(when) {
		t.Fatalf("oldest first, backfilled date kept: %+v", list)
	}
	for _, a := range list {
		if a.Key == "day_7" && a.RunID != prefix+"-run-1" {
			t.Fatal("a second grant overwrote the first")
		}
	}

	// The board counts them, and runs are counted and listed for the backfill.
	if c, err := repo.RunCount(ctx, user.ID); err != nil || c != 0 {
		t.Fatalf("run count before finishing: %d %v", c, err)
	}
	if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) (domain.Effects, error) {
		rec := domain.FinishRun(*g, cfg, domain.EndedByGaveUp)
		rec.Score = 8_000_000 // outrank anything else in a shared database
		return domain.Effects{Finished: &rec}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if c, _ := repo.RunCount(ctx, user.ID); c != 1 {
		t.Fatalf("run count = %d", c)
	}
	best, err := repo.BestScore(ctx, user.ID)
	if err != nil || best == nil || best.Achievements != 3 {
		t.Fatalf("board row: %+v %v", best, err)
	}
	runs, err := repo.FinishedRuns(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range runs {
		if r.RunID == prefix+"-run-1" {
			found = r.UserID == user.ID && r.EndedBy == domain.EndedByGaveUp && r.Days == 1 && !r.CreatedAt.IsZero()
		}
	}
	if !found {
		t.Fatalf("finished run missing from %d runs", len(runs))
	}

	// The backfill marker.
	name := prefix + "-job"
	if done, err := repo.BackfillDone(ctx, name); err != nil || done {
		t.Fatalf("fresh job: %v %v", done, err)
	}
	for i := 0; i < 2; i++ {
		if err := repo.MarkBackfillDone(ctx, name); err != nil {
			t.Fatal(err)
		}
	}
	if done, _ := repo.BackfillDone(ctx, name); !done {
		t.Fatal("job not marked")
	}
}

func TestMemoryAchievements(t *testing.T) {
	achievementsContract(t, NewMemory(), "mem")
}

func openTestPostgres(t *testing.T) (*gorm.DB, *Postgres) {
	t.Helper()
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
	return db, repo
}

func TestPostgresAchievements(t *testing.T) {
	db, repo := openTestPostgres(t)
	cleanup := func() {
		db.Exec("DELETE FROM achievements WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'pgach%')")
		db.Exec("DELETE FROM runs WHERE run_id LIKE 'pgach-%'")
		db.Exec("DELETE FROM backfills WHERE name LIKE 'pgach-%'")
		db.Exec("DELETE FROM games WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'pgach%')")
		db.Exec("DELETE FROM users WHERE username LIKE 'pgach%'")
	}
	cleanup()
	t.Cleanup(cleanup)
	achievementsContract(t, repo, "pgach")
}

// A game row saved before goal facts existed has NULL there: it loads as all zero,
// and the facts round-trip once they are set.
func TestPostgresGoalFactsOnAnOldShapeRow(t *testing.T) {
	db, repo := openTestPostgres(t)
	cleanup := func() {
		db.Exec("DELETE FROM games WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'pggoal%')")
		db.Exec("DELETE FROM users WHERE username LIKE 'pggoal%'")
	}
	cleanup()
	t.Cleanup(cleanup)

	ctx := context.Background()
	cfg := domain.DefaultConfig()
	game := domain.NewGame(cfg, 9)
	if err := domain.Buy(&game, cfg, domain.Lemon, 2); err != nil {
		t.Fatal(err)
	}
	user, err := repo.CreateUserWithGame(ctx, "pggoal1", game)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE games SET goals = NULL WHERE user_id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetGame(ctx, user.ID)
	if err != nil || got.Goals.LastTradeDay != 0 || got.Goals.Sold != nil {
		t.Fatalf("old row goals = %+v, %v", got.Goals, err)
	}

	if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) (domain.Effects, error) {
		g.Goals.EventsSeen = []string{"holiday"}
		g.Goals.EventSales = map[string]int{"holiday/lemonade": 3}
		g.Goals.LongestIdle = 4
		return domain.Effects{}, domain.Sell(g, cfg, domain.Lemon, 1)
	}); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetGame(ctx, user.ID)
	if got.Goals.LongestIdle != 4 || got.Goals.Sold["lemon"] != 1 || got.Goals.EventSales["holiday/lemonade"] != 3 ||
		len(got.Goals.EventsSeen) != 1 || got.Goals.LastTradeDay != 1 {
		t.Fatalf("goals after a round trip: %+v", got.Goals)
	}
}
