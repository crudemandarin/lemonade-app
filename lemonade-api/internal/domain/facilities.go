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
	base := g.WarehouseQty[r] * warehouseTier(cfg, g.WarehouseLevel).Size
	if pct := storageBonusPct(g, cfg, r); pct > 0 {
		return int(float64(base) * (1 + pct/100))
	}
	return base
}

// ProductionCapacity returns lemonade produced per day at full throughput.
func ProductionCapacity(g Game, cfg Config) int {
	return g.ProductionQty * productionTier(cfg, g.ProductionLevel).Size
}

// warehouseBuildings is the total building count across all five warehouses.
func warehouseBuildings(g Game) int {
	total := 0
	for _, n := range g.WarehouseQty {
		total += n
	}
	return total
}

// WarehouseUpkeep is the total daily upkeep for all warehouses.
func WarehouseUpkeep(g Game, cfg Config) int {
	return facilityUpkeepAfterDiscount(g, cfg, "warehouse", warehouseBuildings(g)*warehouseTier(cfg, g.WarehouseLevel).Upkeep)
}

// ProductionUpkeep is the total daily upkeep for the production facility.
func ProductionUpkeep(g Game, cfg Config) int {
	return facilityUpkeepAfterDiscount(g, cfg, "production", g.ProductionQty*productionTier(cfg, g.ProductionLevel).Upkeep)
}

// TotalUpkeep is the daily upkeep across both facility types, the hubs and upgrades.
func TotalUpkeep(g Game, cfg Config) int {
	return WarehouseUpkeep(g, cfg) + ProductionUpkeep(g, cfg) + HubUpkeep(g, cfg) + UpgradeUpkeep(g, cfg)
}

// ResaleValue is what one building of the given tier sells for.
func ResaleValue(cfg Config, t Tier) int {
	return int(cfg.ResaleRate * float64(t.BuildCost))
}

// FacilityResaleValue is what all buildings would fetch if sold today (net worth).
func FacilityResaleValue(g Game, cfg Config) int {
	return warehouseBuildings(g)*ResaleValue(cfg, warehouseTier(cfg, g.WarehouseLevel)) +
		g.ProductionQty*ResaleValue(cfg, productionTier(cfg, g.ProductionLevel))
}

// CanSellFacility says whether one building could be sold now, and if not, why:
// ErrMinFacility, or a *StockExceedsCapacityError. resource selects the
// warehouse and is ignored for production.
func CanSellFacility(g Game, cfg Config, kind FacilityType, resource Resource) error {
	switch kind {
	case Warehouse:
		if !cfg.Valid(resource) {
			return ErrInvalidFacility
		}
		if g.WarehouseQty[resource] <= 1 {
			return ErrMinFacility
		}
		size := warehouseTier(cfg, g.WarehouseLevel).Size
		if excess := g.Inventory[resource] - (g.WarehouseQty[resource]-1)*size; excess > 0 {
			return &StockExceedsCapacityError{Excess: excess}
		}
	case Production:
		if g.ProductionQty <= 1 {
			return ErrMinFacility
		}
	default:
		return ErrInvalidFacility
	}
	return nil
}

// SellFacility sells one building back at ResaleRate of its build cost. Only
// quantity is sold: levels never change. It never sells stock to make room.
func SellFacility(g *Game, cfg Config, kind FacilityType, resource Resource) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	if err := CanSellFacility(*g, cfg, kind, resource); err != nil {
		return err
	}

	var proceeds int
	switch kind {
	case Warehouse:
		proceeds = ResaleValue(cfg, warehouseTier(cfg, g.WarehouseLevel))
		g.WarehouseQty[resource]--
	case Production:
		proceeds = ResaleValue(cfg, productionTier(cfg, g.ProductionLevel))
		g.ProductionQty--
		resource = ""
	}
	g.Capital += proceeds
	g.Stats.FacilitiesSold++
	g.Stats.FacilityProceeds += proceeds
	g.record(TimelinePoint{Day: g.Day, Kind: PointSellFacility, Facility: kind, Resource: resource, Qty: 1, Amount: proceeds})
	return nil
}
