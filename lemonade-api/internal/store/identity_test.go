package store

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"lemonade-api/internal/domain"
)

// identityContract covers Firebase-linked accounts and runs against every Repository.
// prefix keeps names and UIDs unique so the Postgres run can clean up after itself.
func identityContract(t *testing.T, repo Repository, prefix string) {
	t.Helper()
	ctx := context.Background()
	game := domain.NewGame(domain.DefaultConfig(), 3)

	if _, err := repo.FindUserByUID(ctx, prefix+"-uid-a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("FindUserByUID before create: %v", err)
	}

	// A new profile links the UID and creates the game.
	a, err := repo.CreateProfile(ctx, prefix+"alice", prefix+"-uid-a", "Alice@Example.com", game)
	if err != nil || a.Username != prefix+"alice" {
		t.Fatalf("CreateProfile = %+v, %v", a, err)
	}
	if got, err := repo.FindUserByUID(ctx, prefix+"-uid-a"); err != nil || got != a {
		t.Fatalf("FindUserByUID = %+v, %v", got, err)
	}
	if _, err := repo.GetGame(ctx, a.ID); err != nil {
		t.Fatalf("profile has no game: %v", err)
	}

	// Taken username (case is normalized by the caller); UID already linked.
	if _, err := repo.CreateProfile(ctx, prefix+"alice", prefix+"-uid-b", "", game); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("duplicate username: %v", err)
	}
	if _, err := repo.CreateProfile(ctx, prefix+"other", prefix+"-uid-a", "", game); !errors.Is(err, ErrAlreadyLinked) {
		t.Fatalf("uid already linked: %v", err)
	}
	if _, err := repo.FindUser(ctx, prefix+"other"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("failed profile left a user behind: %v", err)
	}

	// Claim a legacy (uid-less) user, keeping their game.
	legacy, err := repo.CreateUserWithGame(ctx, prefix+"legacy", game)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Mutate(ctx, legacy.ID, func(g *domain.Game) (domain.Effects, error) {
		g.Capital = 4321
		return domain.Effects{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	claimed, err := repo.ClaimUser(ctx, prefix+"legacy", prefix+"-uid-c", "c@example.com")
	if err != nil || claimed.ID != legacy.ID {
		t.Fatalf("ClaimUser = %+v, %v", claimed, err)
	}
	if g, _ := repo.GetGame(ctx, legacy.ID); g.Capital != 4321 {
		t.Fatalf("claim lost the game: capital = %d", g.Capital)
	}
	if got, err := repo.FindUserByUID(ctx, prefix+"-uid-c"); err != nil || got.ID != legacy.ID {
		t.Fatalf("FindUserByUID after claim = %+v, %v", got, err)
	}

	// Claim errors.
	if _, err := repo.ClaimUser(ctx, prefix+"nobody", prefix+"-uid-d", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown username: %v", err)
	}
	if _, err := repo.ClaimUser(ctx, prefix+"legacy", prefix+"-uid-d", ""); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("already claimed: %v", err)
	}
	if _, err := repo.ClaimUser(ctx, prefix+"alice", prefix+"-uid-d", ""); !errors.Is(err, ErrAlreadyClaimed) {
		t.Fatalf("claiming a Firebase-created profile: %v", err)
	}
	spare, _ := repo.CreateUserWithGame(ctx, prefix+"spare", game)
	if _, err := repo.ClaimUser(ctx, prefix+"spare", prefix+"-uid-a", ""); !errors.Is(err, ErrAlreadyLinked) {
		t.Fatalf("uid already linked: %v", err)
	}
	if _, err := repo.FindUserByUID(ctx, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty uid must never match legacy users: %v", err)
	}

	// Two simultaneous claims of one legacy user: exactly one wins.
	race, _ := repo.CreateUserWithGame(ctx, prefix+"race", game)
	var wg sync.WaitGroup
	results := make([]error, 8)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, results[i] = repo.ClaimUser(ctx, prefix+"race", prefix+"-uid-r"+string(rune('0'+i)), "")
		}()
	}
	wg.Wait()
	wins := 0
	for _, err := range results {
		switch {
		case err == nil:
			wins++
		case !errors.Is(err, ErrAlreadyClaimed):
			t.Fatalf("racing claim: %v", err)
		}
	}
	if wins != 1 {
		t.Fatalf("%d claims won, want 1", wins)
	}
	_ = race
	_ = spare
}

func TestMemoryIdentity(t *testing.T) {
	identityContract(t, NewMemory(), "mem")
}

func TestPostgresIdentity(t *testing.T) {
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
		db.Exec("DELETE FROM games WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'pgid%')")
		db.Exec("DELETE FROM users WHERE username LIKE 'pgid%'")
	}
	cleanup()
	t.Cleanup(cleanup)
	identityContract(t, repo, "pgid")
}
