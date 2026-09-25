package store

import (
	"context"
	"errors"
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
	Stock    [5]int `json:"s"`
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
	Prices [5]int   `json:"prices"`
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

	Inventory map[string]int `gorm:"type:jsonb;serializer:json"`
	// CostBasis is NULL on rows saved before it existed; fromRow seeds those.
	CostBasis    map[string]int       `gorm:"type:jsonb;serializer:json"`
	WarehouseQty map[string]int       `gorm:"type:jsonb;serializer:json"`
	Market       map[string]marketRow `gorm:"type:jsonb;serializer:json"`
	Events       []eventRow           `gorm:"type:jsonb;serializer:json"`
	Timeline     []pointRow           `gorm:"type:jsonb;serializer:json"`
	Stats        statsRow             `gorm:"type:jsonb;serializer:json"`
	// PriceLog is NULL on rows saved before it existed; fromRow seeds those.
	PriceLog []priceRow `gorm:"type:jsonb;serializer:json"`

	UpdatedAt time.Time
}

func (gameRow) TableName() string { return "games" }

// Postgres implements Repository on GORM.
type Postgres struct {
	db *gorm.DB
}

func NewPostgres(db *gorm.DB) *Postgres {
	return &Postgres{db: db}
}

// Migrate creates or updates the users and games tables.
func (p *Postgres) Migrate() error {
	return p.db.AutoMigrate(&userRow{}, &gameRow{})
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

func (p *Postgres) Mutate(ctx context.Context, userID uint, fn func(g *domain.Game) error) (domain.Game, error) {
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
		if err := fn(&g); err != nil {
			return err
		}

		updated := toRow(g)
		updated.ID = row.ID
		updated.UserID = row.UserID
		if err := tx.Save(&updated).Error; err != nil {
			return err
		}
		out = g
		return nil
	})
	return out, err
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
		Inventory:       make(map[string]int, len(g.Inventory)),
		WarehouseQty:    make(map[string]int, len(g.WarehouseQty)),
		CostBasis:       make(map[string]int, len(g.CostBasis)),
		Market:          make(map[string]marketRow, len(g.Market)),
		Events:          make([]eventRow, 0, len(g.Events)),
		Timeline:        make([]pointRow, 0, len(g.Timeline)),
		PriceLog:        make([]priceRow, 0, len(g.PriceLog)),
		Stats:           statsRow(g.Stats),
	}
	for _, p := range g.Timeline {
		row.Timeline = append(row.Timeline, pointRow{
			Day: p.Day, Kind: string(p.Kind), Resource: string(p.Resource), Facility: string(p.Facility),
			Qty: p.Qty, Amount: p.Amount, Produced: p.Produced, Capital: p.Capital, Stock: p.Stock,
		})
	}
	for _, pp := range g.PriceLog {
		row.PriceLog = append(row.PriceLog, priceRow{Day: pp.Day, Prices: pp.Prices, Events: pp.Events})
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
		Inventory:       make(map[domain.Resource]int, len(row.Inventory)),
		WarehouseQty:    make(map[domain.Resource]int, len(row.WarehouseQty)),
		Market:          make(map[domain.Resource]*domain.ResourceMarket, len(row.Market)),
		Stats:           domain.Stats(row.Stats),
	}
	for _, p := range row.Timeline {
		g.Timeline = append(g.Timeline, domain.TimelinePoint{
			Day: p.Day, Kind: domain.PointKind(p.Kind), Resource: domain.Resource(p.Resource), Facility: domain.FacilityType(p.Facility),
			Qty: p.Qty, Amount: p.Amount, Produced: p.Produced, Capital: p.Capital, Stock: p.Stock,
		})
	}
	for _, pp := range row.PriceLog {
		g.PriceLog = append(g.PriceLog, domain.PricePoint{Day: pp.Day, Prices: pp.Prices, Events: pp.Events})
	}
	for r, n := range row.Inventory {
		g.Inventory[domain.Resource(r)] = n
	}
	for r, n := range row.WarehouseQty {
		g.WarehouseQty[domain.Resource(r)] = n
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
