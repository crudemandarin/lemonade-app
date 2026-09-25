package domain

import "testing"

func TestDay100SnapshotIsTakenOnArrivingAndOnlyOnce(t *testing.T) {
	g, cfg := newTestGame()
	g.Day, g.Capital = 99, 5000
	RecordMilestones(&g, cfg)
	if g.NetWorthDay100 != nil {
		t.Fatal("no snapshot before day 100")
	}

	g.Day = 100
	RecordMilestones(&g, cfg)
	if g.NetWorthDay100 == nil || *g.NetWorthDay100 != NetWorth(g, cfg) {
		t.Fatalf("snapshot = %v, want the net worth on arrival", g.NetWorthDay100)
	}
	first := *g.NetWorthDay100

	g.Capital, g.Day = 90000, 130
	RecordMilestones(&g, cfg)
	if *g.NetWorthDay100 != first {
		t.Fatalf("a later call replaced the snapshot: %d, was %d", *g.NetWorthDay100, first)
	}
}

func TestNoSnapshotForAFinishedRun(t *testing.T) {
	g, cfg := newTestGame()
	g.Day, g.Status = 100, StatusBankrupt
	RecordMilestones(&g, cfg)
	if g.NetWorthDay100 != nil {
		t.Fatal("a bankrupt run must not be snapshotted")
	}
}

func TestSnapshotThroughRealPlayAndItsRecord(t *testing.T) {
	g, cfg := newTestGame()
	g.Capital = 1_000_000 // survives 99 idle days of upkeep
	for g.Day < BoardDay {
		before := g.Clone()
		report, err := EndDay(&g, cfg)
		if err != nil {
			t.Fatal(err)
		}
		RecordDayFacts(before, &g, cfg, report)
		RecordMilestones(&g, cfg)
		if g.Day < BoardDay && g.NetWorthDay100 != nil {
			t.Fatalf("snapshot too early on day %d", g.Day)
		}
	}
	if g.NetWorthDay100 == nil || *g.NetWorthDay100 != NetWorth(g, cfg) {
		t.Fatalf("snapshot on arrival = %v", g.NetWorthDay100)
	}
	rec := FinishRun(g, cfg, EndedByGaveUp)
	if rec.NetWorthDay100 == nil || *rec.NetWorthDay100 != *g.NetWorthDay100 {
		t.Fatalf("the record lost the snapshot: %v", rec.NetWorthDay100)
	}

	// A run that ended before day 100 has none.
	early, _ := newTestGame()
	if FinishRun(early, cfg, EndedByGaveUp).NetWorthDay100 != nil {
		t.Fatal("an early run has no day 100 value")
	}
}

func TestCloneDoesNotShareTheSnapshot(t *testing.T) {
	g, _ := newTestGame()
	v := 7
	g.NetWorthDay100 = &v
	c := g.Clone()
	*c.NetWorthDay100 = 9
	if *g.NetWorthDay100 != 7 {
		t.Fatal("clone shares the snapshot pointer")
	}
}
