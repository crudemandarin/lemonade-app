package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"lemonade-api/internal/store"
)

const (
	defaultScoreLimit = 20
	maxScoreLimit     = 100
)

// bestDTO is the player's top finished run: shown on the game page, and how the
// result screen knows a run just set a new personal best.
type bestDTO struct {
	Score int    `json:"score"`
	Days  int    `json:"days"`
	RunID string `json:"runId"`
}

// scoreRowDTO is one line of the global board. It carries no run ID: other
// players' runs are not viewable.
type scoreRowDTO struct {
	Rank      int       `json:"rank"`
	Username  string    `json:"username"`
	Score     int       `json:"score"`
	Days      int       `json:"days"`
	NetWorth  int       `json:"netWorth"`
	CreatedAt time.Time `json:"createdAt"`
	// IsMe marks the caller's own row.
	IsMe bool `json:"isMe"`
	// Achievements is how many the player has unlocked, shown as a badge.
	Achievements int `json:"achievements"`
}

type scoresDTO struct {
	// Board is the board these rows belong to: all_time or day_100. On day_100 a row's
	// score is the net worth on arriving at day 100, and its days is always 100.
	Board string        `json:"board"`
	Rows  []scoreRowDTO `json:"rows"`
	// Me is the caller's own best row and rank, even when it is below the rows shown; null with no finished run.
	Me *scoreRowDTO `json:"me"`
}

type runSummaryDTO struct {
	RunID     string    `json:"runId"`
	Score     int       `json:"score"`
	Days      int       `json:"days"`
	NetWorth  int       `json:"netWorth"`
	Capital   int       `json:"capital"`
	EndedBy   string    `json:"endedBy"`
	CreatedAt time.Time `json:"createdAt"`
	IsBest    bool      `json:"isBest"`
}

// runDetailDTO is one finished run in full, for its owner.
type runDetailDTO struct {
	runSummaryDTO
	Stats      statsDTO           `json:"stats"`
	Timeline   []timelinePointDTO `json:"timeline"`
	PriceLog   []pricePointDTO    `json:"priceLog"`
	BasePrices []int              `json:"basePrices"`
	// Commodities is the catalog the arrays above follow.
	Commodities []commodityDTO `json:"commodities"`
	// Reports is the light list of the run's ended days, as GET /game/reports returns.
	Reports []reportSummaryDTO `json:"reports"`
	// Achievements are the ones this run unlocked.
	Achievements []unlockedDTO `json:"achievements"`
}

func toScoreRow(r store.ScoreRow, callerID uint) scoreRowDTO {
	return scoreRowDTO{
		Rank: r.Rank, Username: r.Username, Score: r.Score, Days: r.Days,
		NetWorth: r.NetWorth, CreatedAt: r.CreatedAt, IsMe: r.UserID == callerID, Achievements: r.Achievements,
	}
}

// scores serves the global board: one row per player (their best run), best first.
func (h *Game) scores(c *gin.Context) {
	limit := defaultScoreLimit
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			abort(c, http.StatusBadRequest, "invalid_limit", "Limit must be a whole number.")
			return
		}
		limit = min(max(n, 1), maxScoreLimit)
	}

	board := store.Board(c.DefaultQuery("board", string(store.BoardAllTime)))
	if !board.Valid() {
		abort(c, http.StatusBadRequest, "invalid_board", "Board must be all_time or day_100.")
		return
	}

	ctx, me := c.Request.Context(), currentUser(c).ID
	rows, err := h.repo.TopBoard(ctx, board, limit)
	if err != nil {
		abortErr(c, err)
		return
	}
	out := scoresDTO{Board: string(board), Rows: make([]scoreRowDTO, 0, len(rows))}
	for _, r := range rows {
		out.Rows = append(out.Rows, toScoreRow(r, me))
	}
	best, err := h.repo.BestOnBoard(ctx, board, me)
	if err != nil {
		abortErr(c, err)
		return
	}
	if best != nil {
		row := toScoreRow(*best, me)
		out.Me = &row
	}
	c.JSON(http.StatusOK, out)
}

// myRuns lists the caller's finished runs, newest first, with their best flagged.
func (h *Game) myRuns(c *gin.Context) {
	ctx, me := c.Request.Context(), currentUser(c).ID
	runs, err := h.repo.UserRuns(ctx, me)
	if err != nil {
		abortErr(c, err)
		return
	}
	best, err := h.repo.BestScore(ctx, me)
	if err != nil {
		abortErr(c, err)
		return
	}
	out := make([]runSummaryDTO, 0, len(runs))
	for _, r := range runs {
		out = append(out, toRunSummary(r, best))
	}
	c.JSON(http.StatusOK, out)
}

func toRunSummary(r store.RunSummary, best *store.ScoreRow) runSummaryDTO {
	return runSummaryDTO{
		RunID: r.RunID, Score: r.Score, Days: r.Days, NetWorth: r.NetWorth, Capital: r.Capital,
		EndedBy: r.EndedBy, CreatedAt: r.CreatedAt, IsBest: best != nil && best.RunID == r.RunID,
	}
}

// myRun serves one finished run in full. Only its owner may read it: anyone else
// gets the same 404 as for a run that does not exist.
func (h *Game) myRun(c *gin.Context) {
	ctx, me := c.Request.Context(), currentUser(c).ID
	run, err := h.repo.GetRun(ctx, me, c.Param("runId"))
	if err != nil {
		abortErr(c, err)
		return
	}
	best, err := h.repo.BestScore(ctx, me)
	if err != nil {
		abortErr(c, err)
		return
	}
	reports, err := h.repo.ListReports(ctx, run.RunID)
	if err != nil {
		abortErr(c, err)
		return
	}
	summaries := make([]reportSummaryDTO, 0, len(reports))
	for _, r := range reports {
		summaries = append(summaries, toReportSummary(r))
	}
	unlocked, err := h.repo.ListAchievements(ctx, me)
	if err != nil {
		abortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, runDetailDTO{
		runSummaryDTO: toRunSummary(store.RunSummary{
			RunID: run.RunID, Score: run.Score, Days: run.Days, NetWorth: run.NetWorth,
			Capital: run.Capital, EndedBy: run.EndedBy, CreatedAt: run.CreatedAt,
		}, best),
		Stats:        statsDTO(run.Stats),
		Timeline:     timelinePointDTOs(run.Timeline, h.cfg),
		PriceLog:     priceLogDTOs(run.PriceLog, h.cfg),
		BasePrices:   basePrices(h.cfg),
		Commodities:  toCommodityDTOs(h.cfg),
		Reports:      summaries,
		Achievements: runAchievements(unlocked, run.RunID),
	})
}
