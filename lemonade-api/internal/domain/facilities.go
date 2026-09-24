package domain

// warehouseTier returns the current warehouse tier (level is 1-based).
func warehouseTier(cfg Config, level int) Tier {
	return cfg.WarehouseTiers[level-1]
}

// productionTier returns the current production tier (level is 1-based).
func productionTier(cfg Config, level int) Tier {
	return cfg.ProductionTiers[level-1]
}

// Capacity returns the case capacity of one resource's warehouse.
func Capacity(g Game, cfg Config, r Resource) int {
	return g.WarehouseQty[r] * warehouseTier(cfg, g.WarehouseLevel).Size
}

// ProductionCapacity returns lemonade produced per day at full throughput.
func ProductionCapacity(g Game, cfg Config) int {
	return g.ProductionQty * productionTier(cfg, g.ProductionLevel).Size
}

// warehouseBuildings is the total building count across all five warehouses.
func warehouseBuildings(g Game) int {
	total := 0
	for _, r := range Resources {
		total += g.WarehouseQty[r]
	}
	return total
}

// WarehouseUpkeep is the total daily upkeep for all warehouses.
func WarehouseUpkeep(g Game, cfg Config) int {
	return warehouseBuildings(g) * warehouseTier(cfg, g.WarehouseLevel).Upkeep
}

// ProductionUpkeep is the total daily upkeep for the production facility.
func ProductionUpkeep(g Game, cfg Config) int {
	return g.ProductionQty * productionTier(cfg, g.ProductionLevel).Upkeep
}

// TotalUpkeep is the daily upkeep across both facility types.
func TotalUpkeep(g Game, cfg Config) int {
	return WarehouseUpkeep(g, cfg) + ProductionUpkeep(g, cfg)
}
