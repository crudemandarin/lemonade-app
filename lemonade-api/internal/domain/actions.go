package domain

// Buy purchases qty cases of r at the current ask price (SPEC rule 5).
func Buy(g *Game, cfg Config, r Resource, qty int) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	if qty <= 0 {
		return ErrInvalidQuantity
	}

	ask := Quotes(*g, cfg)[r].Ask
	cost := ask * qty
	if cost > g.Capital {
		return ErrInsufficientFunds
	}
	if g.Inventory[r]+qty > Capacity(*g, cfg, r) {
		return ErrCapacityExceeded
	}

	g.Capital -= cost
	g.Inventory[r] += qty
	g.Stats.CasesBought += qty
	g.Stats.Spent += cost
	g.record(TimelinePoint{Day: g.Day, Kind: PointBuy, Resource: r, Qty: qty, Amount: cost})
	return nil
}

// Sell sells qty cases of r at the current bid price (SPEC rule 6).
func Sell(g *Game, cfg Config, r Resource, qty int) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	if qty <= 0 {
		return ErrInvalidQuantity
	}
	if g.Inventory[r] < qty {
		return ErrInsufficientStock
	}

	bid := Quotes(*g, cfg)[r].Bid
	g.Capital += bid * qty
	g.Inventory[r] -= qty
	g.Stats.CasesSold += qty
	g.Stats.Earned += bid * qty
	g.record(TimelinePoint{Day: g.Day, Kind: PointSell, Resource: r, Qty: qty, Amount: bid * qty})
	return nil
}

// Expand adds one building at the facility's current level (SPEC rule 9).
// resource selects which warehouse to expand; it is ignored for production.
func Expand(g *Game, cfg Config, kind FacilityType, resource Resource) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}

	switch kind {
	case Warehouse:
		if g.WarehouseQty[resource] >= cfg.MaxQuantity {
			return ErrMaxQuantity
		}
		cost := warehouseTier(cfg, g.WarehouseLevel).BuildCost
		if cost > g.Capital {
			return ErrInsufficientFunds
		}
		g.Capital -= cost
		g.WarehouseQty[resource]++
		g.recordFacility(PointExpand, kind, resource, 1, cost)
	case Production:
		if g.ProductionQty >= cfg.MaxQuantity {
			return ErrMaxQuantity
		}
		cost := productionTier(cfg, g.ProductionLevel).BuildCost
		if cost > g.Capital {
			return ErrInsufficientFunds
		}
		g.Capital -= cost
		g.ProductionQty++
		g.recordFacility(PointExpand, kind, "", 1, cost)
	default:
		return ErrInvalidFacility
	}
	return nil
}

// Upgrade raises every building of kind to the next level (SPEC rule 10).
func Upgrade(g *Game, cfg Config, kind FacilityType) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}

	switch kind {
	case Warehouse:
		if g.WarehouseLevel >= cfg.MaxLevel {
			return ErrMaxLevel
		}
		total := warehouseTier(cfg, g.WarehouseLevel).UpgradeCost * warehouseBuildings(*g)
		if total > g.Capital {
			return ErrInsufficientFunds
		}
		g.Capital -= total
		g.WarehouseLevel++
		g.recordFacility(PointUpgrade, kind, "", 0, total)
	case Production:
		if g.ProductionLevel >= cfg.MaxLevel {
			return ErrMaxLevel
		}
		total := productionTier(cfg, g.ProductionLevel).UpgradeCost * g.ProductionQty
		if total > g.Capital {
			return ErrInsufficientFunds
		}
		g.Capital -= total
		g.ProductionLevel++
		g.recordFacility(PointUpgrade, kind, "", 0, total)
	default:
		return ErrInvalidFacility
	}
	return nil
}
