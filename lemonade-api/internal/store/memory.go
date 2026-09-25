package store

import (
	"context"
	"sync"

	"lemonade-api/internal/domain"
)

// Memory is an in-memory Repository for tests. A single mutex stands in for the
// row lock, so Mutate calls are serialized.
type Memory struct {
	mu     sync.Mutex
	nextID uint
	users  map[string]domain.User
	games  map[uint]domain.Game

	reports map[string]map[int]domain.DayReport // by run ID, then day
	runs    []domain.RunRecord                  // finished runs, oldest first
}

func NewMemory() *Memory {
	return &Memory{
		nextID: 1,
		users:  map[string]domain.User{},
		games:  map[uint]domain.Game{},

		reports: map[string]map[int]domain.DayReport{},
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
