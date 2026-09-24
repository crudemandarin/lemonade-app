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
	if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) error {
		g.Capital = 123
		return errors.New("boom")
	}); err == nil {
		t.Fatal("expected fn error to propagate")
	}
	if g, _ := repo.GetGame(ctx, user.ID); g.Capital != game.Capital {
		t.Fatalf("failed mutate was saved: capital = %d, want %d", g.Capital, game.Capital)
	}
	if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) error {
		g.Capital = 123
		return nil
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
			if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) error {
				g.Capital++
				return nil
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
	if _, err := repo.Mutate(ctx, 99999, func(*domain.Game) error { return nil }); !errors.Is(err, ErrNotFound) {
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
