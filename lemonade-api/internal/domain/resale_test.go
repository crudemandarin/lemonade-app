package domain

import (
	"errors"
	"testing"
)

func TestSellWarehouseBuilding(t *testing.T) {
	g, cfg := newTestGame()
	g.WarehouseQty[Lemon] = 3
	before := TotalUpkeep(g, cfg)

	if err := SellFacility(&g, cfg, Warehouse, Lemon); err != nil {
		t.Fatal(err)
	}
	// Pantry build cost $100, resale rate 0.5.
	if g.WarehouseQty[Lemon] != 2 || g.Capital != 1000+50 {
		t.Fatalf("qty=%d capital=%d", g.WarehouseQty[Lemon], g.Capital)
	}
	if got := TotalUpkeep(g, cfg); got != before-cfg.WarehouseTiers[0].Upkeep {
		t.Fatalf("upkeep %d -> %d, want it to drop by one Pantry", before, got)
	}
	if g.Stats.FacilitiesSold != 1 || g.Stats.FacilityProceeds != 50 {
		t.Fatalf("stats %+v", g.Stats)
	}
	last := g.Timeline[len(g.Timeline)-1]
	if last.Kind != PointSellFacility || last.Facility != Warehouse || last.Resource != Lemon || last.Amount != 50 || last.Capital != g.Capital {
		t.Fatalf("timeline point %+v", last)
	}
}

func TestSellProductionBuildingUsesItsLevelsBuildCost(t *testing.T) {
	g, cfg := newTestGame()
	g.ProductionQty, g.ProductionLevel = 2, 2 // Food Truck: build $1,500
	if err := SellFacility(&g, cfg, Production, ""); err != nil {
		t.Fatal(err)
	}
	if g.ProductionQty != 1 || g.Capital != 1000+750 || g.ProductionLevel != 2 {
		t.Fatalf("qty=%d level=%d capital=%d", g.ProductionQty, g.ProductionLevel, g.Capital)
	}
}

func TestResaleRoundsDown(t *testing.T) {
	g, cfg := newTestGame()
	cfg.ResaleRate = 0.5
	cfg.WarehouseTiers = append([]Tier(nil), cfg.WarehouseTiers...)
	cfg.WarehouseTiers[0].BuildCost = 101
	g.WarehouseQty[Ice] = 2
	if err := SellFacility(&g, cfg, Warehouse, Ice); err != nil || g.Capital != 1050 {
		t.Fatalf("err=%v capital=%d, want floor(50.5)=50", err, g.Capital)
	}
}

func TestSellFacilityGuards(t *testing.T) {
	g, cfg := newTestGame()

	if err := SellFacility(&g, cfg, Production, ""); !errors.Is(err, ErrMinFacility) {
		t.Errorf("last production building: %v, want ErrMinFacility", err)
	}
	if err := SellFacility(&g, cfg, Warehouse, Sugar); !errors.Is(err, ErrMinFacility) {
		t.Errorf("last warehouse: %v, want ErrMinFacility", err)
	}

	g.WarehouseQty[Sugar] = 2
	g.Inventory[Sugar] = 14 // one Pantry holds 10
	err := SellFacility(&g, cfg, Warehouse, Sugar)
	var excess *StockExceedsCapacityError
	if !errors.Is(err, ErrStockExceedsCapacity) || !errors.As(err, &excess) || excess.Excess != 4 {
		t.Fatalf("got %v, want stock exceeds capacity by 4", err)
	}
	if g.WarehouseQty[Sugar] != 2 || g.Capital != 1000 {
		t.Fatal("a refused sale changed the game")
	}

	g.Inventory[Sugar] = 10
	if err := SellFacility(&g, cfg, Warehouse, Sugar); err != nil {
		t.Fatalf("stock that exactly fits should be sellable: %v", err)
	}

	if err := SellFacility(&g, cfg, Warehouse, Resource("bogus")); err == nil {
		t.Error("unknown resource should be rejected")
	}
	if err := SellFacility(&g, cfg, FacilityType("garage"), ""); !errors.Is(err, ErrInvalidFacility) {
		t.Errorf("unknown type: %v", err)
	}
	g.Status = StatusBankrupt
	if err := SellFacility(&g, cfg, Production, ""); !errors.Is(err, ErrGameOver) {
		t.Errorf("after game over: %v", err)
	}
}

func TestFacilityResaleValue(t *testing.T) {
	g, cfg := newTestGame()
	// 5 Pantries ($50 each) and one Kitchen ($250).
	if got := FacilityResaleValue(g, cfg); got != 5*50+250 {
		t.Fatalf("got %d, want 500", got)
	}
	g.WarehouseLevel, g.ProductionQty = 2, 3 // 5 Garages ($150), 3 Kitchens... level 1 kitchens
	if got := FacilityResaleValue(g, cfg); got != 5*150+3*250 {
		t.Fatalf("got %d", got)
	}
}

// Buying a building and selling it back must always lose money, however it was
// reached: by building at the level, or building at level 1 and upgrading.
func TestResaleIsAlwaysBelowWhatABuildingCost(t *testing.T) {
	cfg := DefaultConfig()
	for name, tiers := range map[string][]Tier{"warehouse": cfg.WarehouseTiers, "production": cfg.ProductionTiers} {
		viaUpgrades := tiers[0].BuildCost
		for i, tier := range tiers {
			cheapest := tier.BuildCost
			if viaUpgrades < cheapest {
				cheapest = viaUpgrades
			}
			if resale := ResaleValue(cfg, tier); resale >= cheapest {
				t.Errorf("%s level %d: resale $%d >= cheapest way to own one, $%d", name, i+1, resale, cheapest)
			}
			viaUpgrades += tier.UpgradeCost
		}
	}
}
