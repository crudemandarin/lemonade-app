package store

import (
	"context"
	"errors"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lemonade-api/internal/domain"
)

// Achievements and the one-off backfill, for both repositories.

func (p *Postgres) ListAchievements(ctx context.Context, userID uint) ([]Achievement, error) {
	var rows []achievementRow
	if err := p.db.WithContext(ctx).Where("user_id = ?", userID).Order("unlocked_at, key").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Achievement, 0, len(rows))
	for _, r := range rows {
		out = append(out, Achievement{Key: r.Key, RunID: r.RunID, UnlockedAt: r.UnlockedAt})
	}
	return out, nil
}

func (p *Postgres) GrantAchievements(ctx context.Context, userID uint, grants []Achievement) (int, error) {
	if len(grants) == 0 {
		return 0, nil
	}
	rows := make([]achievementRow, 0, len(grants))
	for _, a := range grants {
		rows = append(rows, achievementRow{UserID: userID, Key: a.Key, RunID: a.RunID, UnlockedAt: a.UnlockedAt})
	}
	res := p.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&rows)
	return int(res.RowsAffected), res.Error
}

func (p *Postgres) RunCount(ctx context.Context, userID uint) (int, error) {
	var n int64
	err := p.db.WithContext(ctx).Model(&runRow{}).Where("user_id = ?", userID).Count(&n).Error
	return int(n), err
}

func (p *Postgres) FinishedRuns(ctx context.Context) ([]FinishedRun, error) {
	var rows []runRow
	err := p.db.WithContext(ctx).Select("user_id, run_id, days, score, net_worth, capital, ended_by, stats, created_at").
		Order("user_id, created_at, id").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]FinishedRun, 0, len(rows))
	for _, r := range rows {
		out = append(out, FinishedRun{UserID: r.UserID, CreatedAt: r.CreatedAt, RunRecord: domain.RunRecord{
			RunID: r.RunID, Days: r.Days, Score: r.Score, NetWorth: r.NetWorth, Capital: r.Capital,
			EndedBy: r.EndedBy, Stats: domain.Stats(r.Stats),
		}})
	}
	return out, nil
}

func (p *Postgres) BackfillDone(ctx context.Context, name string) (bool, error) {
	var row backfillRow
	err := p.db.WithContext(ctx).Where("name = ?", name).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (p *Postgres) MarkBackfillDone(ctx context.Context, name string) error {
	return p.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&backfillRow{Name: name, DoneAt: time.Now()}).Error
}

// memory

func (m *Memory) ListAchievements(_ context.Context, userID uint) ([]Achievement, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Achievement{}
	for _, a := range m.achievements[userID] {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].UnlockedAt.Equal(out[j].UnlockedAt) {
			return out[i].UnlockedAt.Before(out[j].UnlockedAt)
		}
		return out[i].Key < out[j].Key
	})
	return out, nil
}

// grant stores achievements the player does not have yet; callers hold the lock.
func (m *Memory) grant(userID uint, grants []Achievement) int {
	if m.achievements == nil {
		m.achievements = map[uint]map[string]Achievement{}
	}
	if m.achievements[userID] == nil {
		m.achievements[userID] = map[string]Achievement{}
	}
	n := 0
	for _, a := range grants {
		if _, ok := m.achievements[userID][a.Key]; !ok {
			m.achievements[userID][a.Key] = a
			n++
		}
	}
	return n
}

func (m *Memory) GrantAchievements(_ context.Context, userID uint, grants []Achievement) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.grant(userID, grants), nil
}

func (m *Memory) RunCount(_ context.Context, userID uint) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, r := range m.runs {
		if m.runOwner[r.RunID] == userID {
			n++
		}
	}
	return n, nil
}

func (m *Memory) FinishedRuns(_ context.Context) ([]FinishedRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]FinishedRun, 0, len(m.runs))
	for _, r := range m.runs {
		rec := r
		rec.Timeline, rec.PriceLog = nil, nil
		out = append(out, FinishedRun{UserID: m.runOwner[r.RunID], CreatedAt: m.runMeta[r.RunID].createdAt, RunRecord: rec})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UserID != out[j].UserID {
			return out[i].UserID < out[j].UserID
		}
		return m.runMeta[out[i].RunID].earlier(m.runMeta[out[j].RunID])
	})
	return out, nil
}

func (m *Memory) BackfillDone(_ context.Context, name string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.backfills[name], nil
}

func (m *Memory) MarkBackfillDone(_ context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.backfills == nil {
		m.backfills = map[string]bool{}
	}
	m.backfills[name] = true
	return nil
}
