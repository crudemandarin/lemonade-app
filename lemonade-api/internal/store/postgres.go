package store

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"lemonade-api/internal/domain"
)

type userRow struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"uniqueIndex;not null"`
	CreatedAt time.Time
}

func (userRow) TableName() string { return "users" }

type marketRow struct {
	Price             float64 `json:"price"`
	PreviousEffective *int    `json:"previousEffective"`
	History           []int   `json:"history"`
}

// pointRow is one timeline point, with short keys because there can be hundreds.
type pointRow struct {
	Day      int    `json:"d"`
	Kind     string `json:"k"`
	Resource string `json:"r,omitempty"`
	Facility string `json:"f,omitempty"`
	Qty      int    `json:"q,omitempty"`
	Amount   int    `json:"a,omitempty"`
	Produced int    `json:"p,omitempty"`
	Capital  int    `json:"c"`
	Stock    counts `json:"s"`
}

type statsRow struct {
	CasesBought      int `json:"casesBought"`
	CasesSold        int `json:"casesSold"`
	Spent            int `json:"spent"`
	Earned           int `json:"earned"`
	FacilitiesBought int `json:"facilitiesBought"`
	Upgrades         int `json:"upgrades"`
	FacilitySpend    int `json:"facilitySpend"`
	FacilitiesSold   int `json:"facilitiesSold"`
	FacilityProceeds int `json:"facilityProceeds"`
	Produced         int `json:"produced"`
	UpkeepPaid       int `json:"upkeepPaid"`
	PeakCapital      int `json:"peakCapital"`
	PeakDay          int `json:"peakDay"`
}

type priceRow struct {
	Day    int      `json:"day"`
	Prices counts   `json:"prices"`
	Events []string `json:"events"`
}

type eventRow struct {
	Key         string             `json:"key"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Multipliers map[string]float64 `json:"multipliers"`
	DaysLeft    int                `json:"daysLeft"`
}

// gameRow keeps queryable scalars as columns and carried-along state as JSONB.
type gameRow struct {
	ID              uint   `gorm:"primaryKey"`
	UserID          uint   `gorm:"uniqueIndex;not null"`
	Seed            int64  `gorm:"not null"`
	Day             int    `gorm:"not null"`
	Capital         int    `gorm:"not null"`
	Status          string `gorm:"not null"`
	WarehouseLevel  int    `gorm:"not null"`
	ProductionLevel int    `gorm:"not null"`
	ProductionQty   int    `gorm:"not null"`
	RunID           string // empty on rows saved before runs existed

	Inventory map[string]int `gorm:"type:jsonb;serializer:json"`
	// BuyPressure and SellPressure are NULL on rows saved before market depth existed (zero).
	BuyPressure  map[string]float64 `gorm:"type:jsonb;serializer:json"`
	SellPressure map[string]float64 `gorm:"type:jsonb;serializer:json"`
	// CostBasis is NULL on rows saved before it existed; fromRow seeds those.
	CostBasis    map[string]int       `gorm:"type:jsonb;serializer:json"`
	WarehouseQty map[string]int       `gorm:"type:jsonb;serializer:json"`
	Market       map[string]marketRow `gorm:"type:jsonb;serializer:json"`
	Events       []eventRow           `gorm:"type:jsonb;serializer:json"`
	Timeline     []pointRow           `gorm:"type:jsonb;serializer:json"`
	Stats        statsRow             `gorm:"type:jsonb;serializer:json"`
	// PriceLog is NULL on rows saved before it existed; fromRow seeds those.
	PriceLog []priceRow `gorm:"type:jsonb;serializer:json"`
	// Upgrades, UpgradeSpend, IceOld and Carry are NULL on rows saved before upgrades
	// existed: nothing owned, nothing kept, no fractions carried.
	Upgrades     map[string]int `gorm:"type:jsonb;serializer:json"`
	UpgradeSpend int
	IceOld       int
	Carry        map[string]float64 `gorm:"type:jsonb;serializer:json"`
	// Goals are the achievement facts; NULL (all zero) on rows saved before they existed.
	Goals domain.GoalStats `gorm:"type:jsonb;serializer:json"`
	// NetWorthDay100 is NULL until the run reaches day 100 (and on older rows).
	NetWorthDay100 *int `gorm:"column:net_worth_day100"`

	UpdatedAt time.Time
}

func (gameRow) TableName() string { return "games" }

// dayReportRow is one ended day of one run. Reports are written when the day ends
// and never loaded with the game, so the game row does not grow.
type dayReportRow struct {
	RunID   string           `gorm:"primaryKey"`
	Day     int              `gorm:"primaryKey;autoIncrement:false"`
	Payload domain.DayReport `gorm:"type:jsonb;serializer:json"`
}

func (dayReportRow) TableName() string { return "day_reports" }

// runRow is a finished run. Difficulty is reserved (3 is the default level).
type runRow struct {
	ID         uint       `gorm:"primaryKey"`
	UserID     uint       `gorm:"not null;index:idx_runs_user_created,priority:1"`
	RunID      string     `gorm:"not null;uniqueIndex"`
	Difficulty int        `gorm:"not null;default:3;index:idx_runs_difficulty_score,priority:1"`
	Days       int        `gorm:"not null"`
	Score      int        `gorm:"not null;index:idx_runs_difficulty_score,priority:2,sort:desc"`
	NetWorth   int        `gorm:"not null"`
	Capital    int        `gorm:"not null"`
	EndedBy    string     `gorm:"not null"`
	Timeline   []pointRow `gorm:"type:jsonb;serializer:json"`
	Stats      statsRow   `gorm:"type:jsonb;serializer:json"`
	PriceLog   []priceRow `gorm:"type:jsonb;serializer:json"`
	// NetWorthDay100 is the net worth on arriving at day 100; NULL when the run ended
	// earlier and for runs stored before the day-100 board existed.
	NetWorthDay100 *int      `gorm:"column:net_worth_day100"`
	CreatedAt      time.Time `gorm:"index:idx_runs_user_created,priority:2,sort:desc"`
}

func (runRow) TableName() string { return "runs" }

// achievementRow is one achievement a player has unlocked, with the run it came from.
// The key is unique per player, so unlocking twice is a no-op.
type achievementRow struct {
	UserID     uint   `gorm:"primaryKey;autoIncrement:false"`
	Key        string `gorm:"primaryKey"`
	RunID      string
	UnlockedAt time.Time `gorm:"not null"`
}

func (achievementRow) TableName() string { return "achievements" }

// backfillRow marks a one-off data job as done.
type backfillRow struct {
	Name   string `gorm:"primaryKey"`
	DoneAt time.Time
}

func (backfillRow) TableName() string { return "backfills" }

// Postgres implements Repository on GORM.
type Postgres struct {
	db *gorm.DB
}

func NewPostgres(db *gorm.DB) *Postgres {
	return &Postgres{db: db}
}

// Migrate creates or updates the users and games tables.
func (p *Postgres) Migrate() error {
	return p.db.AutoMigrate(&userRow{}, &gameRow{}, &dayReportRow{}, &runRow{}, &achievementRow{}, &backfillRow{})
}

func (p *Postgres) FindUser(ctx context.Context, username string) (domain.User, error) {
	var row userRow
	err := p.db.WithContext(ctx).Where("username = ?", username).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{ID: row.ID, Username: row.Username}, nil
}

func (p *Postgres) CreateUserWithGame(ctx context.Context, username string, game domain.Game) (domain.User, error) {
	var user domain.User
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := userRow{Username: username}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// Lost a race with a concurrent login: the user and their game already exist.
			if err := tx.Where("username = ?", username).First(&row).Error; err != nil {
				return err
			}
			user = domain.User{ID: row.ID, Username: row.Username}
			return nil
		}
		g := toRow(game)
		g.UserID = row.ID
		if err := tx.Create(&g).Error; err != nil {
			return err
		}
		user = domain.User{ID: row.ID, Username: row.Username}
		return nil
	})
	return user, err
}

func (p *Postgres) GetGame(ctx context.Context, userID uint) (domain.Game, error) {
	var row gameRow
	err := p.db.WithContext(ctx).Where("user_id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Game{}, ErrNotFound
	}
	if err != nil {
		return domain.Game{}, err
	}
	return fromRow(row), nil
}

func (p *Postgres) ReplaceGame(ctx context.Context, userID uint, game domain.Game) (domain.Game, error) {
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing gameRow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&existing).Error
		row := toRow(game)
		row.UserID = userID
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return tx.Create(&row).Error
		case err != nil:
			return err
		default:
			row.ID = existing.ID
			return tx.Save(&row).Error
		}
	})
	if err != nil {
		return domain.Game{}, err
	}
	return game.Clone(), nil
}

func (p *Postgres) RunBelongsTo(ctx context.Context, userID uint, runID string) (bool, error) {
	var n int64
	err := p.db.WithContext(ctx).Model(&runRow{}).Where("user_id = ? AND run_id = ?", userID, runID).Count(&n).Error
	return n > 0, err
}

func (p *Postgres) ListReports(ctx context.Context, runID string) ([]domain.DayReport, error) {
	var rows []dayReportRow
	if err := p.db.WithContext(ctx).Where("run_id = ?", runID).Order("day").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.DayReport, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Payload)
	}
	return out, nil
}

func (p *Postgres) GetReport(ctx context.Context, runID string, day int) (domain.DayReport, error) {
	var row dayReportRow
	err := p.db.WithContext(ctx).Where("run_id = ? AND day = ?", runID, day).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.DayReport{}, ErrNotFound
	}
	return row.Payload, err
}

func (p *Postgres) Mutate(ctx context.Context, userID uint, fn func(g *domain.Game) (domain.Effects, error)) (domain.Game, error) {
	var out domain.Game
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row gameRow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		g := fromRow(row)
		effects, err := fn(&g)
		if err != nil {
			return err
		}

		updated := toRow(g)
		updated.ID = row.ID
		updated.UserID = row.UserID
		if err := tx.Save(&updated).Error; err != nil {
			return err
		}
		if err := saveEffects(tx, userID, g, effects); err != nil {
			return err
		}
		out = g
		return nil
	})
	return out, err
}

// saveEffects writes a day report and a finished run inside the mutation's
// transaction. Both are idempotent per run, so a retry cannot duplicate them.
func saveEffects(tx *gorm.DB, userID uint, g domain.Game, e domain.Effects) error {
	if e.Report != nil && g.RunID != "" {
		row := dayReportRow{RunID: g.RunID, Day: e.Report.Day, Payload: *e.Report}
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
			return err
		}
	}
	if r := e.Finished; r != nil {
		row := runRow{
			UserID: userID, RunID: r.RunID, Difficulty: 3, Days: r.Days, Score: r.Score,
			NetWorth: r.NetWorth, Capital: r.Capital, EndedBy: r.EndedBy,
			Timeline: make([]pointRow, 0, len(r.Timeline)), Stats: statsRow(r.Stats),
			PriceLog: make([]priceRow, 0, len(r.PriceLog)), NetWorthDay100: r.NetWorthDay100,
		}
		for _, p := range r.Timeline {
			row.Timeline = append(row.Timeline, pointRow{
				Day: p.Day, Kind: string(p.Kind), Resource: string(p.Resource), Facility: string(p.Facility),
				Qty: p.Qty, Amount: p.Amount, Produced: p.Produced, Capital: p.Capital, Stock: toCounts(p.Stock),
			})
		}
		for _, pp := range r.PriceLog {
			row.PriceLog = append(row.PriceLog, priceRow{Day: pp.Day, Prices: toCounts(pp.Prices), Events: pp.Events})
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
	}
	if len(e.Unlocked) > 0 {
		now := time.Now()
		rows := make([]achievementRow, 0, len(e.Unlocked))
		for _, key := range e.Unlocked {
			rows = append(rows, achievementRow{UserID: userID, Key: key, RunID: g.RunID, UnlockedAt: now})
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error; err != nil {
			return err
		}
	}
	return nil
}

func toRow(g domain.Game) gameRow {
	row := gameRow{
		Seed:            g.Seed,
		Day:             g.Day,
		Capital:         g.Capital,
		Status:          string(g.Status),
		WarehouseLevel:  g.WarehouseLevel,
		ProductionLevel: g.ProductionLevel,
		ProductionQty:   g.ProductionQty,
		RunID:           g.RunID,
		Inventory:       make(map[string]int, len(g.Inventory)),
		WarehouseQty:    make(map[string]int, len(g.WarehouseQty)),
		CostBasis:       make(map[string]int, len(g.CostBasis)),
		BuyPressure:     make(map[string]float64, len(g.BuyPressure)),
		SellPressure:    make(map[string]float64, len(g.SellPressure)),
		Market:          make(map[string]marketRow, len(g.Market)),
		Events:          make([]eventRow, 0, len(g.Events)),
		Timeline:        make([]pointRow, 0, len(g.Timeline)),
		PriceLog:        make([]priceRow, 0, len(g.PriceLog)),
		Stats:           statsRow(g.Stats),
		Upgrades:        make(map[string]int, len(g.Upgrades)),
		UpgradeSpend:    g.UpgradeSpend,
		IceOld:          g.IceOld,
		Carry:           make(map[string]float64, len(g.Carry)),
		Goals:           g.Goals,
		NetWorthDay100:  g.NetWorthDay100,
	}
	for k, v := range g.Upgrades {
		row.Upgrades[k] = v
	}
	for k, v := range g.Carry {
		row.Carry[k] = v
	}
	for _, p := range g.Timeline {
		row.Timeline = append(row.Timeline, pointRow{
			Day: p.Day, Kind: string(p.Kind), Resource: string(p.Resource), Facility: string(p.Facility),
			Qty: p.Qty, Amount: p.Amount, Produced: p.Produced, Capital: p.Capital, Stock: toCounts(p.Stock),
		})
	}
	for _, pp := range g.PriceLog {
		row.PriceLog = append(row.PriceLog, priceRow{Day: pp.Day, Prices: toCounts(pp.Prices), Events: pp.Events})
	}
	for r, n := range g.Inventory {
		row.Inventory[string(r)] = n
	}
	for r, n := range g.WarehouseQty {
		row.WarehouseQty[string(r)] = n
	}
	for r, n := range g.CostBasis {
		row.CostBasis[string(r)] = n
	}
	for r, v := range g.BuyPressure {
		row.BuyPressure[string(r)] = v
	}
	for r, v := range g.SellPressure {
		row.SellPressure[string(r)] = v
	}
	for r, m := range g.Market {
		row.Market[string(r)] = marketRow{
			Price:             m.Price,
			PreviousEffective: m.PreviousEffective,
			History:           m.History,
		}
	}
	for _, e := range g.Events {
		mult := make(map[string]float64, len(e.Multipliers))
		for r, v := range e.Multipliers {
			mult[string(r)] = v
		}
		row.Events = append(row.Events, eventRow{
			Key:         e.Key,
			Name:        e.Name,
			Description: e.Description,
			Multipliers: mult,
			DaysLeft:    e.DaysLeft,
		})
	}
	return row
}

func fromRow(row gameRow) domain.Game {
	g := domain.Game{
		Seed:            row.Seed,
		Day:             row.Day,
		Capital:         row.Capital,
		Status:          domain.Status(row.Status),
		WarehouseLevel:  row.WarehouseLevel,
		ProductionLevel: row.ProductionLevel,
		ProductionQty:   row.ProductionQty,
		RunID:           row.RunID,
		Inventory:       make(map[domain.Resource]int, len(row.Inventory)),
		WarehouseQty:    make(map[domain.Resource]int, len(row.WarehouseQty)),
		Market:          make(map[domain.Resource]*domain.ResourceMarket, len(row.Market)),
		Stats:           domain.Stats(row.Stats),
		Upgrades:        make(map[string]int, len(row.Upgrades)),
		UpgradeSpend:    row.UpgradeSpend,
		IceOld:          row.IceOld,
		Carry:           make(map[string]float64, len(row.Carry)),
		Goals:           row.Goals,
		NetWorthDay100:  row.NetWorthDay100,
	}
	for k, v := range row.Upgrades {
		g.Upgrades[k] = v
	}
	for k, v := range row.Carry {
		g.Carry[k] = v
	}
	for _, p := range row.Timeline {
		g.Timeline = append(g.Timeline, domain.TimelinePoint{
			Day: p.Day, Kind: domain.PointKind(p.Kind), Resource: domain.Resource(p.Resource), Facility: domain.FacilityType(p.Facility),
			Qty: p.Qty, Amount: p.Amount, Produced: p.Produced, Capital: p.Capital, Stock: fromCounts(p.Stock),
		})
	}
	for _, pp := range row.PriceLog {
		g.PriceLog = append(g.PriceLog, domain.PricePoint{Day: pp.Day, Prices: fromCounts(pp.Prices), Events: pp.Events})
	}
	for r, n := range row.Inventory {
		g.Inventory[domain.Resource(r)] = n
	}
	for r, n := range row.WarehouseQty {
		g.WarehouseQty[domain.Resource(r)] = n
	}
	g.BuyPressure = make(map[domain.Resource]float64, len(row.BuyPressure))
	for r, v := range row.BuyPressure {
		g.BuyPressure[domain.Resource(r)] = v
	}
	g.SellPressure = make(map[domain.Resource]float64, len(row.SellPressure))
	for r, v := range row.SellPressure {
		g.SellPressure[domain.Resource(r)] = v
	}
	if row.CostBasis != nil {
		g.CostBasis = make(map[domain.Resource]int, len(row.CostBasis))
		for r, n := range row.CostBasis {
			g.CostBasis[domain.Resource(r)] = n
		}
	}
	for r, m := range row.Market {
		g.Market[domain.Resource(r)] = &domain.ResourceMarket{
			Price:             m.Price,
			PreviousEffective: m.PreviousEffective,
			History:           m.History,
		}
	}
	for _, e := range row.Events {
		mult := make(map[domain.Resource]float64, len(e.Multipliers))
		for r, v := range e.Multipliers {
			mult[domain.Resource(r)] = v
		}
		g.Events = append(g.Events, domain.ActiveEvent{
			Key:         e.Key,
			Name:        e.Name,
			Description: e.Description,
			Multipliers: mult,
			DaysLeft:    e.DaysLeft,
		})
	}
	// Games saved before cost basis existed get an approximate one.
	domain.SeedCostBasis(&g)
	domain.SeedPriceLog(&g)
	return g
}

// scoreSQL ranks each player's best finished run: highest score first, an earlier
// finish winning a tie (and the row id settling the rest). DISTINCT ON keeps one run
// per user; ROW_NUMBER gives the rank, so a player far down the board still has one.
func scoreSQL(board Board) string {
	if board == BoardDay100 {
		// The day-100 board ranks the net worth on arriving at day 100; runs without one
		// (ended earlier, or stored before the board existed) are left off.
		return boardSQL(`SELECT DISTINCT ON (user_id) id, user_id, run_id, net_worth_day100 AS score,
    ` + strconv.Itoa(domain.BoardDay) + ` AS days, net_worth_day100 AS net_worth, created_at
  FROM runs
  WHERE net_worth_day100 IS NOT NULL
  ORDER BY user_id, net_worth_day100 DESC, created_at ASC, id ASC`)
	}
	return boardSQL(`SELECT DISTINCT ON (user_id) id, user_id, run_id, score, days, net_worth, created_at
  FROM runs
  ORDER BY user_id, score DESC, created_at ASC, id ASC`)
}

func boardSQL(best string) string {
	return `
WITH best AS (
  ` + best + `
), ranked AS (
  SELECT ROW_NUMBER() OVER (ORDER BY b.score DESC, b.created_at ASC, b.id ASC) AS rank,
         b.user_id, u.username, b.run_id, b.score, b.days, b.net_worth, b.created_at,
         (SELECT COUNT(*) FROM achievements a WHERE a.user_id = b.user_id) AS achievements
  FROM best b JOIN users u ON u.id = b.user_id
)
SELECT rank, user_id, username, run_id, score, days, net_worth, created_at, achievements FROM ranked`
}

type scoreScan struct {
	Rank      int
	UserID    uint
	Username  string
	RunID     string
	Score     int
	Days      int
	NetWorth  int
	CreatedAt time.Time
	// Achievements is how many achievements the player has unlocked.
	Achievements int
}

func (p *Postgres) TopScores(ctx context.Context, limit int) ([]ScoreRow, error) {
	return p.TopBoard(ctx, BoardAllTime, limit)
}

func (p *Postgres) TopBoard(ctx context.Context, board Board, limit int) ([]ScoreRow, error) {
	var scans []scoreScan
	if err := p.db.WithContext(ctx).Raw(scoreSQL(board)+" ORDER BY rank LIMIT ?", limit).Scan(&scans).Error; err != nil {
		return nil, err
	}
	out := make([]ScoreRow, 0, len(scans))
	for _, s := range scans {
		out = append(out, ScoreRow(s))
	}
	return out, nil
}

func (p *Postgres) BestScore(ctx context.Context, userID uint) (*ScoreRow, error) {
	return p.BestOnBoard(ctx, BoardAllTime, userID)
}

func (p *Postgres) BestOnBoard(ctx context.Context, board Board, userID uint) (*ScoreRow, error) {
	var scans []scoreScan
	if err := p.db.WithContext(ctx).Raw(scoreSQL(board)+" WHERE user_id = ?", userID).Scan(&scans).Error; err != nil {
		return nil, err
	}
	if len(scans) == 0 {
		return nil, nil
	}
	row := ScoreRow(scans[0])
	return &row, nil
}

func (p *Postgres) UserRuns(ctx context.Context, userID uint) ([]RunSummary, error) {
	var rows []runRow
	err := p.db.WithContext(ctx).Select("run_id, score, days, net_worth, capital, ended_by, created_at").
		Where("user_id = ?", userID).Order("created_at DESC, id DESC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]RunSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, RunSummary{
			RunID: r.RunID, Score: r.Score, Days: r.Days, NetWorth: r.NetWorth,
			Capital: r.Capital, EndedBy: r.EndedBy, CreatedAt: r.CreatedAt,
		})
	}
	return out, nil
}

func (p *Postgres) GetRun(ctx context.Context, userID uint, runID string) (RunDetail, error) {
	var row runRow
	err := p.db.WithContext(ctx).Where("user_id = ? AND run_id = ?", userID, runID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return RunDetail{}, ErrNotFound
	}
	if err != nil {
		return RunDetail{}, err
	}
	rec := domain.RunRecord{
		RunID: row.RunID, Days: row.Days, Score: row.Score, NetWorth: row.NetWorth,
		Capital: row.Capital, EndedBy: row.EndedBy, Stats: domain.Stats(row.Stats),
	}
	for _, p := range row.Timeline {
		rec.Timeline = append(rec.Timeline, domain.TimelinePoint{
			Day: p.Day, Kind: domain.PointKind(p.Kind), Resource: domain.Resource(p.Resource), Facility: domain.FacilityType(p.Facility),
			Qty: p.Qty, Amount: p.Amount, Produced: p.Produced, Capital: p.Capital, Stock: fromCounts(p.Stock),
		})
	}
	for _, pp := range row.PriceLog {
		rec.PriceLog = append(rec.PriceLog, domain.PricePoint{Day: pp.Day, Prices: fromCounts(pp.Prices), Events: pp.Events})
	}
	return RunDetail{RunRecord: rec, CreatedAt: row.CreatedAt}, nil
}
