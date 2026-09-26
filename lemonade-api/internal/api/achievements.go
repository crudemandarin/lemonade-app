package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"lemonade-api/internal/domain"
	"lemonade-api/internal/domain/content"
	"lemonade-api/internal/store"
)

// Achievements are evaluated after every successful mutation, inside its
// transaction (see mutateWithEffects), and listed by GET /api/achievements.

// unlockedDTO is one achievement a mutation just unlocked, for the toast.
type unlockedDTO struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Tier string `json:"tier"`
}

type achievementCategoryDTO struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type progressDTO struct {
	Current int `json:"current"`
	Target  int `json:"target"`
}

// achievementDTO is one row of the achievements page. A hidden achievement that is
// still locked has a null name and description.
type achievementDTO struct {
	Key         string       `json:"key"`
	Name        *string      `json:"name"`
	Description *string      `json:"description"`
	Category    string       `json:"category"`
	Tier        string       `json:"tier"`
	Hidden      bool         `json:"hidden"`
	Unlocked    bool         `json:"unlocked"`
	UnlockedAt  *time.Time   `json:"unlockedAt"`
	RunID       *string      `json:"runId"`
	Progress    *progressDTO `json:"progress"`
}

type achievementsDTO struct {
	Categories   []achievementCategoryDTO `json:"categories"`
	Achievements []achievementDTO         `json:"achievements"`
	// UnlockedCount and Total count the rows of the table.
	UnlockedCount int `json:"unlockedCount"`
	Total         int `json:"total"`
}

var achievementByKey = func() map[string]content.AchievementDef {
	m := make(map[string]content.AchievementDef, len(content.Achievements))
	for _, d := range content.Achievements {
		m[d.Key] = d
	}
	return m
}()

// toUnlocked maps unlocked keys to toast rows ([] when none, never null).
func toUnlocked(keys []string) []unlockedDTO {
	out := make([]unlockedDTO, 0, len(keys))
	for _, k := range keys {
		if d, ok := achievementByKey[k]; ok {
			out = append(out, unlockedDTO{Key: d.Key, Name: d.Name, Tier: d.Tier})
		}
	}
	return out
}

// playerHistory is what achievements need from outside the game, loaded before the
// mutation takes its lock. The board part is only loaded for mutations that can end
// a run (end day, give up).
type playerHistory struct {
	unlocked map[string]bool
	runs     int
	best     *store.ScoreRow
	top      []store.ScoreRow
}

// boardDepth is how many board rows are read to rank a finishing run for top 10:
// one extra, in case the player is already among them.
const boardDepth = 11

func (h *Game) loadHistory(ctx context.Context, userID uint, mayFinish bool) (playerHistory, error) {
	list, err := h.repo.ListAchievements(ctx, userID)
	if err != nil {
		return playerHistory{}, err
	}
	ph := playerHistory{unlocked: make(map[string]bool, len(list))}
	for _, a := range list {
		ph.unlocked[a.Key] = true
	}
	if !mayFinish {
		return ph, nil
	}
	if ph.runs, err = h.repo.RunCount(ctx, userID); err != nil {
		return ph, err
	}
	if ph.best, err = h.repo.BestScore(ctx, userID); err != nil {
		return ph, err
	}
	ph.top, err = h.repo.TopScores(ctx, boardDepth)
	return ph, err
}

// achievementContext builds the evaluation context once the mutation has run.
func (ph playerHistory) achievementContext(userID uint, finished *domain.RunRecord, now time.Time) domain.AchievementContext {
	ctx := domain.AchievementContext{Unlocked: ph.unlocked, RunsFinished: ph.runs}
	if ph.best != nil {
		prev := ph.best.Score
		ctx.PreviousBest = &prev
	}
	if finished == nil {
		return ctx
	}
	score := finished.Score
	ctx.FinishedScore = &score
	ctx.RunsFinished++
	ctx.BoardRank = boardRank(ph, userID, score, now)
	return ctx
}

// boardRank is the player's rank once a run scoring score counts, by the board's
// rules (decision 30): higher score first, an earlier finish winning a tie. It is
// exact within the rows read, and past them it is only known to be below them.
func boardRank(ph playerHistory, userID uint, score int, now time.Time) int {
	mine, at := score, now
	if ph.best != nil && ph.best.Score >= score {
		mine, at = ph.best.Score, ph.best.CreatedAt
	}
	ahead := 0
	for _, r := range ph.top {
		if r.UserID == userID {
			continue
		}
		if r.Score > mine || (r.Score == mine && !r.CreatedAt.After(at)) {
			ahead++
		}
	}
	return ahead + 1
}

// listAchievements serves every definition with the caller's unlocked state, and
// progress toward the measurable ones in the current game.
func (h *Game) listAchievements(c *gin.Context) {
	ctx, me := c.Request.Context(), currentUser(c).ID
	list, err := h.repo.ListAchievements(ctx, me)
	if err != nil {
		abortErr(c, err)
		return
	}
	got := make(map[string]store.Achievement, len(list))
	for _, a := range list {
		got[a.Key] = a
	}
	g, err := h.repo.GetGame(ctx, me)
	if err != nil {
		abortErr(c, err)
		return
	}
	runs, err := h.repo.RunCount(ctx, me)
	if err != nil {
		abortErr(c, err)
		return
	}
	actx := domain.AchievementContext{RunsFinished: runs}

	out := achievementsDTO{
		Categories:   make([]achievementCategoryDTO, 0, len(content.AchievementCategories)),
		Achievements: make([]achievementDTO, 0, len(content.Achievements)),
		Total:        len(content.Achievements),
	}
	for _, cat := range content.AchievementCategories {
		out.Categories = append(out.Categories, achievementCategoryDTO{Key: cat.Key, Name: cat.Name})
	}
	for _, d := range content.Achievements {
		row := achievementDTO{Key: d.Key, Category: d.Category, Tier: d.Tier, Hidden: d.Hidden}
		a, unlocked := got[d.Key]
		if unlocked || !d.Hidden {
			name, desc := d.Name, d.Description
			row.Name, row.Description = &name, &desc
		}
		if unlocked {
			out.UnlockedCount++
			at, run := a.UnlockedAt, a.RunID
			row.Unlocked, row.UnlockedAt = true, &at
			if run != "" {
				row.RunID = &run
			}
		} else if !d.Hidden {
			if cur, target, ok := domain.Progress(d.Check, g, h.cfg, actx); ok {
				row.Progress = &progressDTO{Current: cur, Target: target}
			}
		}
		out.Achievements = append(out.Achievements, row)
	}
	c.JSON(http.StatusOK, out)
}

// runAchievements lists what one run unlocked, in the table's order.
func runAchievements(list []store.Achievement, runID string) []unlockedDTO {
	in := make(map[string]bool)
	for _, a := range list {
		if a.RunID == runID {
			in[a.Key] = true
		}
	}
	var keys []string
	for _, d := range content.Achievements {
		if in[d.Key] {
			keys = append(keys, d.Key)
		}
	}
	return toUnlocked(keys)
}

// AchievementsBackfill names the one-off retroactive grant.
const AchievementsBackfill = "achievements_v1"

// BackfillAchievements grants every player what their stored runs prove (see
// domain.ProvenByRuns), dated when the proving run finished. It runs once: a marker
// row records it, and granting is idempotent anyway, so a crash half way is safe to
// repeat. It returns how many achievements were granted.
func BackfillAchievements(ctx context.Context, repo store.Repository) (int, error) {
	if done, err := repo.BackfillDone(ctx, AchievementsBackfill); err != nil || done {
		return 0, err
	}
	runs, err := repo.FinishedRuns(ctx)
	if err != nil {
		return 0, err
	}
	top, err := repo.TopScores(ctx, 10)
	if err != nil {
		return 0, err
	}
	rank := make(map[uint]int, len(top))
	for _, r := range top {
		rank[r.UserID] = r.Rank
	}

	granted := 0
	for start := 0; start < len(runs); {
		end := start
		for end < len(runs) && runs[end].UserID == runs[start].UserID {
			end++
		}
		mine := runs[start:end]
		records := make([]domain.RunRecord, len(mine))
		for i, r := range mine {
			records[i] = r.RunRecord
		}
		var grants []store.Achievement
		for _, p := range domain.ProvenByRuns(content.Achievements, records, rank[mine[0].UserID]) {
			grants = append(grants, store.Achievement{Key: p.Key, RunID: mine[p.Run].RunID, UnlockedAt: mine[p.Run].CreatedAt})
		}
		n, err := repo.GrantAchievements(ctx, mine[0].UserID, grants)
		if err != nil {
			return granted, err
		}
		granted += n
		start = end
	}
	return granted, repo.MarkBackfillDone(ctx, AchievementsBackfill)
}
