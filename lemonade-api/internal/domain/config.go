package domain

import "lemonade-api/internal/domain/content"

// Tier holds the per-building numbers for one facility level.
type Tier struct {
	Name string
	// Size is cases held (warehouse) or lemonade produced per day (production), per building.
	Size int
	// BuildCost is the cost to add one building at this level (Expand).
	BuildCost int
	// UpgradeCost is the per-building cost to move to the next level (0 at max level).
	UpgradeCost int
	// Upkeep is the per-building daily upkeep at this level.
	Upkeep int
}

// EventDef is one row of the event table (see DESIGN.md §6).
type EventDef struct {
	Key         string
	Name        string
	Description string
	Duration    int
	Multipliers map[Resource]float64
	// Excludes lists event keys that cannot be active at the same time as this
	// one. It only needs to be declared on one side of a pair.
	Excludes []string
	// Kind groups events for forecasts: EventWeather, EventMarket or EventSupply.
	Kind string
}

// Event kinds. A weather radio forecasts only EventWeather events.
const (
	EventWeather = "weather"
	EventMarket  = "market"
	EventSupply  = "supply"
)

// Config holds every tunable "physics" knob. See DESIGN.md §6.
type Config struct {
	StartingCapital int

	// Commodities and Recipes are the content catalog (internal/domain/content), in
	// display order. BasePrice starts as the catalog's base prices and stays a knob
	// here so tuning and tests can change it.
	Commodities []Commodity
	Recipes     []Recipe
	BasePrice   map[Resource]int
	Spread      float64

	// Walk: p' = p + RevertRate*(base-p) + p*Sigma*N(0,1), clamped to [ClampMin, ClampMax]*base.
	RevertRate float64
	Sigma      float64
	ClampMin   float64
	ClampMax   float64

	// Market depth: the player's own trades move prices against them. The first
	// FreeDepth[r] cases bought (or sold) recently trade at the plain quote at warehouse
	// level 1; DepthByLevel multiplies it for higher levels (index level-1). Beyond the depth
	// each case moves the price by ImpactShape times its share of the depth, up to
	// ImpactCap. Bought and sold cases are tracked separately, so buying only raises the ask
	// and selling only lowers the bid. Each night the remembered volume falls by Recovery.
	FreeDepth    map[Resource]int
	DepthByLevel []float64
	ImpactShape  float64
	Recovery     float64
	ImpactCap    float64

	// ResaleRate is the share of a building's build cost returned when it is sold.
	ResaleRate float64

	MaxLevel    int
	MaxQuantity int

	HistoryLength int

	EventChance float64
	Events      []EventDef

	// Upgrades is the upgrade table (content/upgrades.go), in display order.
	Upgrades []UpgradeDef

	// WarehouseTiers and ProductionTiers are indexed by level-1 (level 1..MaxLevel).
	WarehouseTiers  []Tier
	ProductionTiers []Tier
}

// DefaultConfig returns the balance numbers from DESIGN.md §6.
func DefaultConfig() Config {
	commodities, recipes, base := catalog()
	return Config{
		StartingCapital: 1000,
		Commodities:     commodities,
		Recipes:         recipes,
		BasePrice:       base,
		Spread:          0.10,
		RevertRate:      0.15,
		Sigma:           0.12,
		ClampMin:        0.25,
		ClampMax:        4.0,
		FreeDepth: map[Resource]int{
			Lemon: 80, Sugar: 80, Ice: 80, Cup: 80, Lemonade: 80,
		},
		DepthByLevel:  []float64{1, 3.5, 6, 12},
		ImpactShape:   0.24, // 0.3% per case at the level-1 depth of 80
		Recovery:      0.5,
		ImpactCap:     0.6,
		ResaleRate:    0.5,
		MaxLevel:      4,
		MaxQuantity:   10,
		HistoryLength: 14,
		EventChance:   0.25,
		Upgrades:      append([]UpgradeDef(nil), content.Upgrades...),
		Events: []EventDef{
			{
				Key:         "heat_wave",
				Kind:        EventWeather,
				Name:        "Heat Wave",
				Description: "Scorching days drive lemonade demand and ice consumption.",
				Duration:    2,
				Multipliers: map[Resource]float64{Lemonade: 1.4, Ice: 1.3},
				Excludes:    []string{"rainy_week"}, // no heat wave in the rain
			},
			{
				Key:         "rainy_week",
				Kind:        EventWeather,
				Name:        "Rainy Week",
				Description: "Nobody wants lemonade in the rain.",
				Duration:    3,
				Multipliers: map[Resource]float64{Lemonade: 0.75},
			},
			{
				Key:         "lemon_blight",
				Kind:        EventSupply,
				Name:        "Lemon Blight",
				Description: "A crop blight drives up lemon prices.",
				Duration:    3,
				Multipliers: map[Resource]float64{Lemon: 1.7},
			},
			{
				Key:         "sugar_glut",
				Kind:        EventSupply,
				Name:        "Sugar Glut",
				Description: "A bumper harvest floods the sugar market.",
				Duration:    2,
				Multipliers: map[Resource]float64{Sugar: 0.7},
			},
			{
				Key:         "holiday",
				Kind:        EventMarket,
				Name:        "Holiday",
				Description: "A local holiday brings out thirsty crowds.",
				Duration:    1,
				Multipliers: map[Resource]float64{Lemonade: 1.35},
			},
			{
				Key:         "cup_shortage",
				Kind:        EventSupply,
				Name:        "Cup Shortage",
				Description: "A supplier shortage drives up cup prices.",
				Duration:    2,
				Multipliers: map[Resource]float64{Cup: 1.5},
			},
		},
		WarehouseTiers: []Tier{
			{Name: "Pantry", Size: 10, BuildCost: 100, UpgradeCost: 155, Upkeep: 2},
			{Name: "Garage", Size: 25, BuildCost: 220, UpgradeCost: 200, Upkeep: 4},
			{Name: "Barn", Size: 60, BuildCost: 450, UpgradeCost: 400, Upkeep: 8},
			{Name: "Industrial Warehouse", Size: 150, BuildCost: 900, UpgradeCost: 0, Upkeep: 15},
		},
		ProductionTiers: []Tier{
			{Name: "Kitchen", Size: 10, BuildCost: 500, UpgradeCost: 780, Upkeep: 20},
			{Name: "Food Truck", Size: 25, BuildCost: 1100, UpgradeCost: 1000, Upkeep: 40},
			{Name: "Bottling Plant", Size: 60, BuildCost: 2200, UpgradeCost: 2000, Upkeep: 75},
			{Name: "Lemonade Factory", Size: 150, BuildCost: 4500, UpgradeCost: 0, Upkeep: 150},
		},
	}
}
