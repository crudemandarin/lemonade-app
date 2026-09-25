package domain

// How a run ended.
const (
	EndedByBankrupt = "bankrupt"
	EndedByGaveUp   = "gave_up"
)

// RunRecord is the permanent summary of a finished run: what the record and
// leaderboard pages show, plus the history for replaying the charts.
type RunRecord struct {
	RunID string
	// Days is the day the run ended on.
	Days     int
	Score    int
	NetWorth int
	Capital  int
	EndedBy  string
	Timeline []TimelinePoint
	Stats    Stats
	PriceLog []PricePoint
}

// Effects are what a mutation produced beyond changing the game. The store saves
// them in the same transaction as the game, so a run is never finished without its
// record and a day is never ended without its report.
type Effects struct {
	// Report is the day report of an EndDay.
	Report *DayReport
	// Finished is set when the run ended (bankruptcy or giving up).
	Finished *RunRecord
}

// FinishRun builds the record of a run that has just ended. The score is the
// final net worth; the day count is shown beside it, not folded into it.
func FinishRun(g Game, cfg Config, endedBy string) RunRecord {
	c := g.Clone()
	nw := NetWorth(g, cfg)
	return RunRecord{
		RunID:    g.RunID,
		Days:     g.Day,
		Score:    nw,
		NetWorth: nw,
		Capital:  g.Capital,
		EndedBy:  endedBy,
		Timeline: c.Timeline,
		Stats:    c.Stats,
		PriceLog: c.PriceLog,
	}
}
