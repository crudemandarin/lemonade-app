package domain

// NewGame starts a fresh game at Day 1: starting capital, empty inventory, every
// facility at level 1 quantity 1, and initial market prices (SPEC rules 1-3).
func NewGame(cfg Config, seed int64) Game {
	inventory := make(map[Resource]int, len(cfg.Commodities))
	warehouseQty := make(map[string]int, 4)
	market := make(map[Resource]*ResourceMarket, len(cfg.Commodities))

	for _, r := range cfg.Resources() {
		inventory[r] = 0
		// One building per commodity to start, pooled by class, so a class holding two
		// commodities (dry: sugar and cups) starts with two.
		warehouseQty[ClassOf(cfg, r)]++
		base := cfg.BasePrice[r]
		market[r] = &ResourceMarket{
			Price:             float64(base),
			PreviousEffective: nil,
			History:           []int{base},
		}
	}

	g := Game{
		Seed:            seed,
		Day:             1,
		Capital:         cfg.StartingCapital,
		Status:          StatusActive,
		Inventory:       inventory,
		CostBasis:       make(map[Resource]int, len(cfg.Commodities)),
		BuyPressure:     make(map[Resource]float64, len(cfg.Commodities)),
		SellPressure:    make(map[Resource]float64, len(cfg.Commodities)),
		WarehouseLevel:  1,
		WarehouseQty:    warehouseQty,
		ProductionLevel: 1,
		ProductionQty:   1,
		Market:          market,
		Upgrades:        make(map[string]int),
		Carry:           make(map[string]float64),
		Events:          nil,
	}
	SeedEmpire(&g, cfg)
	g.record(TimelinePoint{Day: 1, Kind: PointStart})
	g.logPrices(cfg)
	return g
}
