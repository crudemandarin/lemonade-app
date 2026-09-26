package domain

import (
	"slices"

	"lemonade-api/internal/domain/content"
)

// warehouseTier returns the current warehouse tier (level is 1-based).
func warehouseTier(cfg Config, level int) Tier {
	return cfg.WarehouseTiers[level-1]
}

// productionTier returns the current production tier (level is 1-based).
func productionTier(cfg Config, level int) Tier {
	return cfg.ProductionTiers[level-1]
}

// The storage classes, as the domain and its callers name them.
const (
	StorageDry      = content.StorageDry
	StorageCold     = content.StorageCold
	StorageFrozen   = content.StorageFrozen
	StorageFinished = content.StorageFinished
)

// ClassOf is the storage class a commodity is kept in ("" for an unknown one).
func ClassOf(cfg Config, r Resource) string {
	if c, ok := cfg.Commodity(r); ok {
		return c.StorageClass
	}
	return ""
}

// StorageClasses lists the storage classes in the order they first appear in the catalog.
func StorageClasses(cfg Config) []string {
	var out []string
	for _, c := range cfg.Commodities {
		if !slices.Contains(out, c.StorageClass) {
			out = append(out, c.StorageClass)
		}
	}
	return out
}

// ClassMembers are the commodities kept in a storage class, in catalog order.
func ClassMembers(cfg Config, class string) []Resource {
	var out []Resource
	for _, r := range cfg.Resources() {
		if ClassOf(cfg, r) == class {
			out = append(out, r)
		}
	}
	return out
}

// ResolveClass is classFor for callers outside the package: the storage class a key names
// (a class, or a commodity meaning its class).
func ResolveClass(cfg Config, key string) (string, bool) { return classFor(cfg, key) }

// classFor resolves the key a warehouse action names: a storage class, or any commodity
// (meaning its class).
func classFor(cfg Config, key string) (string, bool) {
	if slices.Contains(StorageClasses(cfg), key) {
		return key, true
	}
	if class := ClassOf(cfg, Resource(key)); class != "" {
		return class, true
	}
	return "", false
}

// ClassStock is the cases held of every commodity in a storage class.
func ClassStock(g Game, cfg Config, class string) int {
	n := 0
	for _, r := range ClassMembers(cfg, class) {
		n += g.Inventory[r]
	}
	return n
}

// ClassCapacity is the pooled case capacity of a storage class.
func ClassCapacity(g Game, cfg Config, class string) int {
	base := g.WarehouseQty[class] * warehouseTier(cfg, g.WarehouseLevel).Size
	if pct := sumEffects(g, cfg, content.EffStoragePct, class); len(g.Upgrades) > 0 && pct > 0 {
		return int(float64(base) * (1 + pct/100))
	}
	return base
}

// Capacity is the case capacity of the pool r is stored in: shared with every other
// commodity of its storage class.
func Capacity(g Game, cfg Config, r Resource) int {
	return ClassCapacity(g, cfg, ClassOf(cfg, r))
}

// FreeSpace is how many more cases of r fit: the room left in its class's pool.
func FreeSpace(g Game, cfg Config, r Resource) int {
	class := ClassOf(cfg, r)
	return ClassCapacity(g, cfg, class) - ClassStock(g, cfg, class)
}

// ProductionCapacity returns lemonade produced per day at full throughput.
func ProductionCapacity(g Game, cfg Config) int {
	return g.ProductionQty * productionTier(cfg, g.ProductionLevel).Size
}

// warehouseBuildings is the total building count across every storage class.
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
// warehouse (a commodity means its storage class, or name the class itself) and is
// ignored for production.
func CanSellFacility(g Game, cfg Config, kind FacilityType, resource Resource) error {
	switch kind {
	case Warehouse:
		class, ok := classFor(cfg, string(resource))
		if !ok {
			return ErrInvalidFacility
		}
		if g.WarehouseQty[class] <= 1 {
			return ErrMinFacility
		}
		size := warehouseTier(cfg, g.WarehouseLevel).Size
		if excess := ClassStock(g, cfg, class) - (g.WarehouseQty[class]-1)*size; excess > 0 {
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
		class, _ := classFor(cfg, string(resource))
		g.WarehouseQty[class]--
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
