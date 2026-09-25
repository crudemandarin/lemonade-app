package api

import (
	"errors"

	"lemonade-api/internal/domain"
)

// These types mirror lemonade-web/src/app/core/api.models.ts exactly.

type userDTO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type resourceViewDTO struct {
	Resource      domain.Resource `json:"resource"`
	Stock         int             `json:"stock"`
	Capacity      int             `json:"capacity"`
	Price         int             `json:"price"`
	PreviousPrice *int            `json:"previousPrice"`
	Bid           int             `json:"bid"`
	Ask           int             `json:"ask"`
	History       []int           `json:"history"`
}

type upgradeOptionDTO struct {
	TierName        string `json:"tierName"`
	CostPerBuilding int    `json:"costPerBuilding"`
	TotalCost       int    `json:"totalCost"`
	SizePerBuilding int    `json:"sizePerBuilding"`
	// UpkeepIncrease is the extra daily upkeep for the whole type after the upgrade.
	UpkeepIncrease int `json:"upkeepIncrease"`
}

// saleDTO says what selling one building would return and whether it is allowed now.
// Reason is empty when CanSell, else min_facility or stock_exceeds_capacity.
type saleDTO struct {
	SellValue int    `json:"sellValue"`
	CanSell   bool   `json:"canSell"`
	Reason    string `json:"sellBlockedReason"`
	// CasesToSell is how many cases must go first when Reason is stock_exceeds_capacity.
	CasesToSell int `json:"casesToSell"`
}

type warehouseResourceDTO struct {
	Resource domain.Resource `json:"resource"`
	Count    int             `json:"count"`
	Capacity int             `json:"capacity"`
	saleDTO
}

type warehouseViewDTO struct {
	TierName        string                 `json:"tierName"`
	Level           int                    `json:"level"`
	MaxLevel        int                    `json:"maxLevel"`
	Buildings       int                    `json:"buildings"`
	MaxCount        int                    `json:"maxCount"`
	SizePerBuilding int                    `json:"sizePerBuilding"`
	ExpandCost      int                    `json:"expandCost"`
	UpkeepPerDay    int                    `json:"upkeepPerDay"`
	Upgrade         *upgradeOptionDTO      `json:"upgrade"`
	Resources       []warehouseResourceDTO `json:"resources"`
}

type productionViewDTO struct {
	TierName        string            `json:"tierName"`
	Level           int               `json:"level"`
	MaxLevel        int               `json:"maxLevel"`
	Buildings       int               `json:"buildings"`
	MaxCount        int               `json:"maxCount"`
	SizePerBuilding int               `json:"sizePerBuilding"`
	ExpandCost      int               `json:"expandCost"`
	UpkeepPerDay    int               `json:"upkeepPerDay"`
	Upgrade         *upgradeOptionDTO `json:"upgrade"`
	RatePerDay      int               `json:"ratePerDay"`
	saleDTO
}

type facilitiesDTO struct {
	Warehouse  warehouseViewDTO  `json:"warehouse"`
	Production productionViewDTO `json:"production"`
}

type gameEventDTO struct {
	Key         string                      `json:"key"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Multipliers map[domain.Resource]float64 `json:"multipliers"`
	DaysLeft    int                         `json:"daysLeft"`
}

// timelinePointDTO is the state right after one action; stock is in the order
// lemon, sugar, ice, cup, lemonade.
type timelinePointDTO struct {
	Day      int    `json:"day"`
	Kind     string `json:"kind"`
	Resource string `json:"resource,omitempty"`
	Facility string `json:"facility,omitempty"`
	Qty      int    `json:"qty"`
	Amount   int    `json:"amount"`
	Produced int    `json:"produced"`
	Capital  int    `json:"capital"`
	Stock    []int  `json:"stock"`
}

type statsDTO struct {
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

type gameViewDTO struct {
	Day          int                `json:"day"`
	Capital      int                `json:"capital"`
	Status       domain.Status      `json:"status"`
	UpkeepPerDay int                `json:"upkeepPerDay"`
	Resources    []resourceViewDTO  `json:"resources"`
	Facilities   facilitiesDTO      `json:"facilities"`
	Events       []gameEventDTO     `json:"events"`
	Timeline     []timelinePointDTO `json:"timeline"`
	Stats        statsDTO           `json:"stats"`
	Projection   projectionDTO      `json:"projection"`
}

type priceChangeDTO struct {
	Resource domain.Resource `json:"resource"`
	Before   int             `json:"before"`
	After    int             `json:"after"`
}

type dayReportDTO struct {
	Day        int `json:"day"`
	Produced   int `json:"produced"`
	IceMelted  int `json:"iceMelted"`
	UpkeepPaid int `json:"upkeepPaid"`
	// Stock sold at bid because cash alone could not cover upkeep (0 when none).
	ForcedSaleCases    int              `json:"forcedSaleCases"`
	ForcedSaleProceeds int              `json:"forcedSaleProceeds"`
	CapitalBefore      int              `json:"capitalBefore"`
	CapitalAfter       int              `json:"capitalAfter"`
	PriceChanges       []priceChangeDTO `json:"priceChanges"`
	NewEvents          []gameEventDTO   `json:"newEvents"`
	ExpiredEvents      []gameEventDTO   `json:"expiredEvents"`
	Bankrupt           bool             `json:"bankrupt"`
}

type endDayResponseDTO struct {
	Report dayReportDTO `json:"report"`
	Game   gameViewDTO  `json:"game"`
}

type errorDTO struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// projectionDTO is what End day would do right now: see domain.PreviewEndDay.
type projectionDTO struct {
	LemonadeToProduce int    `json:"lemonadeToProduce"`
	IceToMelt         int    `json:"iceToMelt"`
	LimitedBy         string `json:"limitedBy"`
}

func toGameView(g domain.Game, cfg domain.Config) gameViewDTO {
	quotes := domain.Quotes(g, cfg)

	resources := make([]resourceViewDTO, 0, len(domain.Resources))
	for _, r := range domain.Resources {
		q := quotes[r]
		m := g.Market[r]
		resources = append(resources, resourceViewDTO{
			Resource:      r,
			Stock:         g.Inventory[r],
			Capacity:      domain.Capacity(g, cfg, r),
			Price:         q.Price,
			PreviousPrice: m.PreviousEffective,
			Bid:           q.Bid,
			Ask:           q.Ask,
			History:       append([]int{}, m.History...),
		})
	}

	return gameViewDTO{
		Day:          g.Day,
		Capital:      g.Capital,
		Status:       g.Status,
		UpkeepPerDay: domain.TotalUpkeep(g, cfg),
		Resources:    resources,
		Facilities: facilitiesDTO{
			Warehouse:  toWarehouseView(g, cfg),
			Production: toProductionView(g, cfg),
		},
		Events:     toEventDTOs(g.Events),
		Timeline:   toTimelineDTOs(g),
		Stats:      statsDTO(g.Stats),
		Projection: projectionDTO(domain.PreviewEndDay(g, cfg)),
	}
}

// toTimelineDTOs maps the timeline. A game saved before the timeline existed has none,
// so it gets a start point from its current state and the charts always have something.
func toTimelineDTOs(g domain.Game) []timelinePointDTO {
	if len(g.Timeline) == 0 {
		stock := make([]int, 0, len(domain.Resources))
		for _, r := range domain.Resources {
			stock = append(stock, g.Inventory[r])
		}
		return []timelinePointDTO{{Day: g.Day, Kind: string(domain.PointStart), Capital: g.Capital, Stock: stock}}
	}
	out := make([]timelinePointDTO, 0, len(g.Timeline))
	for _, p := range g.Timeline {
		out = append(out, timelinePointDTO{
			Day: p.Day, Kind: string(p.Kind), Resource: string(p.Resource), Facility: string(p.Facility),
			Qty: p.Qty, Amount: p.Amount, Produced: p.Produced, Capital: p.Capital, Stock: p.Stock[:],
		})
	}
	return out
}

func toSale(g domain.Game, cfg domain.Config, kind domain.FacilityType, r domain.Resource) saleDTO {
	value := domain.ResaleValue(cfg, cfg.ProductionTiers[g.ProductionLevel-1])
	if kind == domain.Warehouse {
		value = domain.ResaleValue(cfg, cfg.WarehouseTiers[g.WarehouseLevel-1])
	}
	s := saleDTO{SellValue: value, CanSell: true}
	var excess *domain.StockExceedsCapacityError
	switch err := domain.CanSellFacility(g, cfg, kind, r); {
	case err == nil:
	case errors.As(err, &excess):
		s.CanSell, s.Reason, s.CasesToSell = false, "stock_exceeds_capacity", excess.Excess
	default:
		s.CanSell, s.Reason = false, "min_facility"
	}
	return s
}

func toWarehouseView(g domain.Game, cfg domain.Config) warehouseViewDTO {
	tier := cfg.WarehouseTiers[g.WarehouseLevel-1]

	buildings := 0
	resources := make([]warehouseResourceDTO, 0, len(domain.Resources))
	for _, r := range domain.Resources {
		buildings += g.WarehouseQty[r]
		resources = append(resources, warehouseResourceDTO{
			Resource: r,
			Count:    g.WarehouseQty[r],
			Capacity: domain.Capacity(g, cfg, r),
			saleDTO:  toSale(g, cfg, domain.Warehouse, r),
		})
	}

	return warehouseViewDTO{
		TierName:        tier.Name,
		Level:           g.WarehouseLevel,
		MaxLevel:        cfg.MaxLevel,
		Buildings:       buildings,
		MaxCount:        cfg.MaxQuantity,
		SizePerBuilding: tier.Size,
		ExpandCost:      tier.BuildCost,
		UpkeepPerDay:    domain.WarehouseUpkeep(g, cfg),
		Upgrade:         upgradeOption(cfg.WarehouseTiers, g.WarehouseLevel, cfg.MaxLevel, buildings),
		Resources:       resources,
	}
}

func toProductionView(g domain.Game, cfg domain.Config) productionViewDTO {
	tier := cfg.ProductionTiers[g.ProductionLevel-1]
	return productionViewDTO{
		TierName:        tier.Name,
		Level:           g.ProductionLevel,
		MaxLevel:        cfg.MaxLevel,
		Buildings:       g.ProductionQty,
		MaxCount:        cfg.MaxQuantity,
		SizePerBuilding: tier.Size,
		ExpandCost:      tier.BuildCost,
		UpkeepPerDay:    domain.ProductionUpkeep(g, cfg),
		Upgrade:         upgradeOption(cfg.ProductionTiers, g.ProductionLevel, cfg.MaxLevel, g.ProductionQty),
		RatePerDay:      domain.ProductionCapacity(g, cfg),
		saleDTO:         toSale(g, cfg, domain.Production, ""),
	}
}

// upgradeOption describes the next level, or nil at max level.
func upgradeOption(tiers []domain.Tier, level, maxLevel, buildings int) *upgradeOptionDTO {
	if level >= maxLevel {
		return nil
	}
	current, next := tiers[level-1], tiers[level]
	return &upgradeOptionDTO{
		TierName:        next.Name,
		CostPerBuilding: current.UpgradeCost,
		TotalCost:       current.UpgradeCost * buildings,
		SizePerBuilding: next.Size,
		UpkeepIncrease:  (next.Upkeep - current.Upkeep) * buildings,
	}
}

func toEventDTOs(events []domain.ActiveEvent) []gameEventDTO {
	out := make([]gameEventDTO, 0, len(events))
	for _, e := range events {
		out = append(out, gameEventDTO{
			Key:         e.Key,
			Name:        e.Name,
			Description: e.Description,
			Multipliers: e.Multipliers,
			DaysLeft:    e.DaysLeft,
		})
	}
	return out
}

func toDayReport(r domain.DayReport) dayReportDTO {
	changes := make([]priceChangeDTO, 0, len(r.PriceChanges))
	for _, c := range r.PriceChanges {
		changes = append(changes, priceChangeDTO{Resource: c.Resource, Before: c.Before, After: c.After})
	}
	return dayReportDTO{
		Day:                r.Day,
		Produced:           r.Produced,
		IceMelted:          r.IceMelted,
		UpkeepPaid:         r.UpkeepPaid,
		ForcedSaleCases:    r.ForcedSaleCases,
		ForcedSaleProceeds: r.ForcedSaleProceeds,
		CapitalBefore:      r.CapitalBefore,
		CapitalAfter:       r.CapitalAfter,
		PriceChanges:       changes,
		NewEvents:          toEventDTOs(r.NewEvents),
		ExpiredEvents:      toEventDTOs(r.ExpiredEvents),
		Bankrupt:           r.Bankrupt,
	}
}
