package domain

// Buy purchases qty cases of r at the current ask price, plus any price impact from the
// player's own recent buying (SPEC rule 5; see impact.go). It is all or nothing.
func Buy(g *Game, cfg Config, r Resource, qty int) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	if qty <= 0 {
		return ErrInvalidQuantity
	}

	cost := buyCost(*g, cfg, r, qty, g.Capital)
	if cost > g.Capital {
		return ErrInsufficientFunds
	}
	if g.Inventory[r]+qty > Capacity(*g, cfg, r) {
		return ErrCapacityExceeded
	}

	plain := Quotes(*g, cfg)[r].Ask * qty
	g.Capital -= cost
	g.Inventory[r] += qty
	g.addBasis(r, cost)
	g.addPressure(true, r, qty)
	g.Stats.CasesBought += qty
	g.Stats.Spent += cost
	g.noteBuy(cfg, r, qty, cost, plain)
	g.record(TimelinePoint{Day: g.Day, Kind: PointBuy, Resource: r, Qty: qty, Amount: cost})
	return nil
}

// Sell sells qty cases of r at the current bid price, less any price impact from the
// player's own recent selling (SPEC rule 6; see impact.go).
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

	q := QuoteSell(*g, cfg, r, qty, false)
	proceeds := q.Total
	g.Capital += proceeds
	basis := g.removeStock(r, qty)
	g.addPressure(false, r, qty)
	g.Stats.CasesSold += qty
	g.Stats.Earned += proceeds
	g.noteSell(cfg, r, qty, proceeds, q.PlainTotal, basis)
	g.record(TimelinePoint{Day: g.Day, Kind: PointSell, Resource: r, Qty: qty, Amount: proceeds})
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

// BuyClamped buys as many cases as it can, up to qty: limited by cash (counting price
// impact) and by free warehouse space. If none can be bought it returns the error for the
// binding limit (cash first), like Buy. It exists for "buy max".
func BuyClamped(g *Game, cfg Config, r Resource, qty int) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	if qty <= 0 {
		return ErrInvalidQuantity
	}
	n := QuoteBuy(*g, cfg, r, qty, true).Qty
	if n == 0 {
		if MarginalAsk(*g, cfg, r) > g.Capital {
			return ErrInsufficientFunds
		}
		return ErrCapacityExceeded
	}
	return Buy(g, cfg, r, n)
}

// SellClamped sells up to qty cases, limited by stock; ErrInsufficientStock
// when there is none to sell.
func SellClamped(g *Game, cfg Config, r Resource, qty int) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	if qty <= 0 {
		return ErrInvalidQuantity
	}
	if stock := g.Inventory[r]; stock < qty {
		if stock == 0 {
			return ErrInsufficientStock
		}
		qty = stock
	}
	return Sell(g, cfg, r, qty)
}

// GiveUp ends the run on the player's say-so. Nothing is sold or charged: the
// final net worth is what the player had when they stopped.
func GiveUp(g *Game) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	g.Status = StatusGaveUp
	return nil
}
