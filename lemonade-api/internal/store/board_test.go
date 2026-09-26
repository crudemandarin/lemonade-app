package store

import (
	"context"
	"strings"
	"testing"

	"lemonade-api/internal/domain"
)

// day100Contract runs against every Repository. Values are huge so this test's rows
// outrank anything already in a shared database; only its own rows are inspected.
func day100Contract(t *testing.T, repo Repository, prefix string) {
	t.Helper()
	ctx := context.Background()
	cfg := domain.DefaultConfig()
	const base = 20_000_000

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
	// wonOn are the runs that won the game, by day.
	wonOn := map[string]int{"a2": 120}
	// finish stores a run; day100 < 0 means it ended before day 100.
	finish := func(name, runID string, score, day100 int) {
		t.Helper()
		rec := domain.RunRecord{
			RunID: prefix + "-" + runID, Days: 150, Score: base + score, NetWorth: base + score,
			Capital: 100, EndedBy: domain.EndedByGaveUp, WonOnDay: wonOn[runID],
			Timeline: []domain.TimelinePoint{{Day: 1, Kind: domain.PointStart, Capital: 1000}},
		}
		if day100 >= 0 {
			v := base + day100
			rec.NetWorthDay100 = &v
		}
		if _, err := repo.Mutate(ctx, users[name].ID, func(g *domain.Game) (domain.Effects, error) {
			return domain.Effects{Finished: &rec}, nil
		}); err != nil {
			t.Fatal(err)
		}
	}

	// ann's best all-time run (score 900) is not her best day-100 run (a1: 500).
	finish("ann", "a1", 700, 500)
	finish("ann", "a2", 900, 300)
	finish("bob", "b1", 800, 500) // ties ann's best, finished later: ranks below her
	finish("cat", "c1", 950, -1)  // never reached day 100: not on the board

	mine := func(rows []ScoreRow) []ScoreRow {
		var out []ScoreRow
		for _, r := range rows {
			if strings.HasPrefix(r.Username, prefix) {
				out = append(out, r)
			}
		}
		return out
	}
	top, err := repo.TopBoard(ctx, BoardDay100, 50)
	if err != nil {
		t.Fatal(err)
	}
	rows := mine(top)
	if len(rows) != 2 {
		t.Fatalf("day-100 rows = %+v, want ann and bob only", rows)
	}
	if rows[0].Username != prefix+"ann" || rows[0].Score != base+500 || rows[0].RunID != prefix+"-a1" || rows[0].Days != domain.BoardDay {
		t.Fatalf("first row = %+v", rows[0])
	}
	if rows[1].Username != prefix+"bob" || rows[1].Score != base+500 || rows[1].Rank != rows[0].Rank+1 {
		t.Fatalf("tie must go to the earlier finish: %+v", rows)
	}

	if me, err := repo.BestOnBoard(ctx, BoardDay100, users["ann"].ID); err != nil || me == nil || me.RunID != prefix+"-a1" || me.Score != base+500 {
		t.Fatalf("ann on the day-100 board = %+v, %v", me, err)
	}
	if me, _ := repo.BestOnBoard(ctx, BoardDay100, users["cat"].ID); me != nil {
		t.Fatalf("cat never reached day 100: %+v", me)
	}
	if me, _ := repo.BestOnBoard(ctx, BoardDay100, users["dan"].ID); me != nil {
		t.Fatalf("dan has no runs: %+v", me)
	}

	// The all-time board is unchanged by all this: ann's 900 and cat's 950 count there.
	if me, _ := repo.BestScore(ctx, users["ann"].ID); me == nil || me.Score != base+900 {
		t.Fatalf("all-time best = %+v", me)
	}
	if me, _ := repo.BestScore(ctx, users["cat"].ID); me == nil || me.Score != base+950 {
		t.Fatalf("cat all-time = %+v", me)
	}

	// The won mark belongs to the row's run: a2 won (her all-time best), a1 (her day-100 run) did not.
	if me, _ := repo.BestScore(ctx, users["ann"].ID); me == nil || me.WonOnDay != 120 {
		t.Fatalf("ann's all-time row should carry the win day: %+v", me)
	}
	if me, _ := repo.BestOnBoard(ctx, BoardDay100, users["ann"].ID); me == nil || me.WonOnDay != 0 {
		t.Fatalf("ann's day-100 run did not win: %+v", me)
	}
	if me, _ := repo.BestScore(ctx, users["bob"].ID); me == nil || me.WonOnDay != 0 {
		t.Fatalf("bob never won: %+v", me)
	}
}

func TestMemoryDay100Board(t *testing.T) { day100Contract(t, NewMemory(), "mem") }

func TestPostgresDay100Board(t *testing.T) {
	db, repo := openTestPostgres(t)
	cleanup := func() {
		db.Exec("DELETE FROM runs WHERE run_id LIKE 'pgd100-%'")
		db.Exec("DELETE FROM games WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'pgd100%')")
		db.Exec("DELETE FROM users WHERE username LIKE 'pgd100%'")
	}
	cleanup()
	t.Cleanup(cleanup)
	day100Contract(t, repo, "pgd100")

	// A run stored before the column existed has NULL there and stays off the board.
	if err := db.Exec("UPDATE runs SET net_worth_day100 = NULL WHERE run_id = 'pgd100-a1'").Error; err != nil {
		t.Fatal(err)
	}
	u, _ := repo.FindUser(context.Background(), "pgd100ann")
	me, err := repo.BestOnBoard(context.Background(), BoardDay100, u.ID)
	if err != nil || me == nil || me.RunID != "pgd100-a2" {
		t.Fatalf("with a1 old-shaped, ann's day-100 run is a2: %+v %v", me, err)
	}
}

// The snapshot survives a game save and load, and an old row loads without one.
func TestPostgresKeepsTheDay100Snapshot(t *testing.T) {
	db, repo := openTestPostgres(t)
	cleanup := func() {
		db.Exec("DELETE FROM games WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'pgsnap%')")
		db.Exec("DELETE FROM users WHERE username LIKE 'pgsnap%'")
	}
	cleanup()
	t.Cleanup(cleanup)

	ctx := context.Background()
	game := domain.NewGame(domain.DefaultConfig(), 3)
	user, err := repo.CreateUserWithGame(ctx, "pgsnap1", game)
	if err != nil {
		t.Fatal(err)
	}
	if g, _ := repo.GetGame(ctx, user.ID); g.NetWorthDay100 != nil {
		t.Fatalf("a new game has no snapshot: %v", *g.NetWorthDay100)
	}
	if _, err := repo.Mutate(ctx, user.ID, func(g *domain.Game) (domain.Effects, error) {
		v := 12345
		g.NetWorthDay100 = &v
		return domain.Effects{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if g, _ := repo.GetGame(ctx, user.ID); g.NetWorthDay100 == nil || *g.NetWorthDay100 != 12345 {
		t.Fatalf("snapshot after a round trip: %v", g.NetWorthDay100)
	}
	if err := db.Exec("UPDATE games SET net_worth_day100 = NULL WHERE user_id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if g, err := repo.GetGame(ctx, user.ID); err != nil || g.NetWorthDay100 != nil {
		t.Fatalf("old row: %v %v", g.NetWorthDay100, err)
	}
}
