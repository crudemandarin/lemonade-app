package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"lemonade-api/internal/domain"
)

// Memory is an in-memory Repository for tests. A single mutex stands in for the
// row lock, so Mutate calls are serialized.
type Memory struct {
	mu     sync.Mutex
	nextID uint
	users  map[string]domain.User
	games  map[uint]domain.Game

	reports  map[string]map[int]domain.DayReport // by run ID, then day
	runOwner map[string]uint                     // finished run ID -> user ID
	runs     []domain.RunRecord                  // finished runs, oldest first
	runMeta  map[string]runMeta                  // when each finished
	seq      int

	achievements map[uint]map[string]Achievement // by user, then key
	backfills    map[string]bool
}

func NewMemory() *Memory {
	return &Memory{
		nextID: 1,
		users:  map[string]domain.User{},
		games:  map[uint]domain.Game{},

		reports:  map[string]map[int]domain.DayReport{},
		runOwner: map[string]uint{},
		runMeta:  map[string]runMeta{},
	}
}

func (m *Memory) FindUser(_ context.Context, username string) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[username]
	if !ok {
		return domain.User{}, ErrNotFound
	}
	return u, nil
}

func (m *Memory) CreateUserWithGame(_ context.Context, username string, game domain.Game) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.users[username]; ok {
		return u, nil
	}
	u := domain.User{ID: m.nextID, Username: username}
	m.nextID++
	m.users[username] = u
	m.games[u.ID] = game.Clone()
	return u, nil
}

func (m *Memory) GetGame(_ context.Context, userID uint) (domain.Game, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.games[userID]
	if !ok {
		return domain.Game{}, ErrNotFound
	}
	return g.Clone(), nil
}

func (m *Memory) ReplaceGame(_ context.Context, userID uint, game domain.Game) (domain.Game, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.games[userID] = game.Clone()
	return game.Clone(), nil
}

func (m *Memory) Mutate(_ context.Context, userID uint, fn func(g *domain.Game) (domain.Effects, error)) (domain.Game, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stored, ok := m.games[userID]
	if !ok {
		return domain.Game{}, ErrNotFound
	}
	g := stored.Clone()
	effects, err := fn(&g)
	if err != nil {
		return domain.Game{}, err
	}
	m.games[userID] = g.Clone()
	if effects.Report != nil && g.RunID != "" {
		if m.reports[g.RunID] == nil {
			m.reports[g.RunID] = map[int]domain.DayReport{}
		}
		m.reports[g.RunID][effects.Report.Day] = *effects.Report
	}
	if r := effects.Finished; r != nil && !m.hasRun(r.RunID) {
		m.runs = append(m.runs, *r)
		m.runOwner[r.RunID] = userID
		m.seq++
		m.runMeta[r.RunID] = runMeta{createdAt: time.Now(), seq: m.seq}
	}
	if len(effects.Unlocked) > 0 {
		now := time.Now()
		grants := make([]Achievement, 0, len(effects.Unlocked))
		for _, key := range effects.Unlocked {
			grants = append(grants, Achievement{Key: key, RunID: g.RunID, UnlockedAt: now})
		}
		m.grant(userID, grants)
	}
	return g, nil
}

func (m *Memory) hasRun(runID string) bool {
	for _, r := range m.runs {
		if r.RunID == runID {
			return true
		}
	}
	return false
}

// Runs returns the finished runs, oldest first. Test helper: not part of Repository.
func (m *Memory) Runs() []domain.RunRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.RunRecord(nil), m.runs...)
}

// Reports returns the stored day reports of a run by day. Test helper: not part of Repository.
func (m *Memory) Reports(runID string) map[int]domain.DayReport {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[int]domain.DayReport{}
	for d, r := range m.reports[runID] {
		out[d] = r
	}
	return out
}

func (m *Memory) RunBelongsTo(_ context.Context, userID uint, runID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	owner, ok := m.runOwner[runID]
	return ok && owner == userID, nil
}

func (m *Memory) ListReports(_ context.Context, runID string) ([]domain.DayReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.DayReport, 0, len(m.reports[runID]))
	for _, r := range m.reports[runID] {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Day < out[j].Day })
	return out, nil
}

func (m *Memory) GetReport(_ context.Context, runID string, day int) (domain.DayReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reports[runID][day]
	if !ok {
		return domain.DayReport{}, ErrNotFound
	}
	return r, nil
}

// runMeta orders finished runs: by finish time, then by arrival (the clock can tie).
type runMeta struct {
	createdAt time.Time
	seq       int
}

// earlier reports whether a finished before b.
func (a runMeta) earlier(b runMeta) bool {
	if !a.createdAt.Equal(b.createdAt) {
		return a.createdAt.Before(b.createdAt)
	}
	return a.seq < b.seq
}

func (m *Memory) usernameOf(userID uint) string {
	for name, u := range m.users {
		if u.ID == userID {
			return name
		}
	}
	return ""
}

// board builds the ranked best-run-per-user list; callers hold the lock.
func (m *Memory) board(kind Board) []ScoreRow {
	// value is what a run ranks by; ok is false for a run that is off the board.
	value := func(r domain.RunRecord) (int, bool) {
		if kind == BoardDay100 {
			if r.NetWorthDay100 == nil {
				return 0, false
			}
			return *r.NetWorthDay100, true
		}
		return r.Score, true
	}
	best := map[uint]domain.RunRecord{}
	for _, r := range m.runs {
		v, ok := value(r)
		if !ok {
			continue
		}
		uid := m.runOwner[r.RunID]
		cur, has := best[uid]
		cv, _ := value(cur)
		if !has || v > cv ||
			(v == cv && m.runMeta[r.RunID].earlier(m.runMeta[cur.RunID])) {
			best[uid] = r
		}
	}
	rows := make([]ScoreRow, 0, len(best))
	for uid, r := range best {
		v, _ := value(r)
		row := ScoreRow{
			UserID: uid, Username: m.usernameOf(uid), RunID: r.RunID, Score: v,
			Days: r.Days, NetWorth: r.NetWorth, CreatedAt: m.runMeta[r.RunID].createdAt,
			Achievements: len(m.achievements[uid]), WonOnDay: r.WonOnDay,
		}
		if kind == BoardDay100 {
			row.Days, row.NetWorth = domain.BoardDay, v
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Score != rows[j].Score {
			return rows[i].Score > rows[j].Score
		}
		return m.runMeta[rows[i].RunID].earlier(m.runMeta[rows[j].RunID])
	})
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows
}

func (m *Memory) TopScores(ctx context.Context, limit int) ([]ScoreRow, error) {
	return m.TopBoard(ctx, BoardAllTime, limit)
}

func (m *Memory) TopBoard(_ context.Context, kind Board, limit int) ([]ScoreRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows := m.board(kind)
	if limit < len(rows) {
		rows = rows[:limit]
	}
	return rows, nil
}

func (m *Memory) BestScore(ctx context.Context, userID uint) (*ScoreRow, error) {
	return m.BestOnBoard(ctx, BoardAllTime, userID)
}

func (m *Memory) BestOnBoard(_ context.Context, kind Board, userID uint) (*ScoreRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.board(kind) {
		if r.UserID == userID {
			r := r
			return &r, nil
		}
	}
	return nil, nil
}

func (m *Memory) UserRuns(_ context.Context, userID uint) ([]RunSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []RunSummary{}
	for _, r := range m.runs {
		if m.runOwner[r.RunID] == userID {
			out = append(out, RunSummary{
				RunID: r.RunID, Score: r.Score, Days: r.Days, NetWorth: r.NetWorth,
				Capital: r.Capital, EndedBy: r.EndedBy, CreatedAt: m.runMeta[r.RunID].createdAt,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return m.runMeta[out[j].RunID].earlier(m.runMeta[out[i].RunID]) })
	return out, nil
}

func (m *Memory) GetRun(_ context.Context, userID uint, runID string) (RunDetail, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.runs {
		if r.RunID == runID && m.runOwner[runID] == userID {
			return RunDetail{RunRecord: r, CreatedAt: m.runMeta[runID].createdAt}, nil
		}
	}
	return RunDetail{}, ErrNotFound
}
