package domain

// BoardDay is the day the time-capped leaderboard measures: the best net worth a
// player had on arriving at this day (late game Goals B, decision 30 rules).
const BoardDay = 100

// RecordMilestones snapshots the net worth the first time the run reaches BoardDay.
// It changes nothing the rules read, takes the snapshot at most once (a run past the
// day keeps its first value), and never for a run that is not active. The API layer
// calls it after EndDay, in the same transaction as the save.
func RecordMilestones(g *Game, cfg Config) {
	if g.Status != StatusActive || g.NetWorthDay100 != nil || g.Day < BoardDay {
		return
	}
	nw := NetWorth(*g, cfg)
	g.NetWorthDay100 = &nw
}
