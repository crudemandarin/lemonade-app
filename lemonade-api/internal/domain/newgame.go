package domain

// NewGame starts a fresh game at Day 1: starting capital, empty inventory, every
// facility at level 1 quantity 1, and initial market prices (SPEC rules 1-3).
func NewGame(cfg Config, seed int64) Game {
	inventory := make(map[Resource]int, len(Resources))
	warehouseQty := make(map[Resource]int, len(Resources))
	market := make(map[Resource]*ResourceMarket, len(Resources))

	for _, r := range Resources {
		inventory[r] = 0
		warehouseQty[r] = 1
		base := cfg.BasePrice[r]
		market[r] = &ResourceMarket{
			Price:             float64(base),
			PreviousEffective: nil,
			History:           []int{base},
		}
	}

	return Game{
		Seed:            seed,
		Day:             1,
		Capital:         cfg.StartingCapital,
		Status:          StatusActive,
		Inventory:       inventory,
		WarehouseLevel:  1,
		WarehouseQty:    warehouseQty,
		ProductionLevel: 1,
		ProductionQty:   1,
		Market:          market,
		Events:          nil,
	}
}
