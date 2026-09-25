package domain

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
}

// Config holds every tunable "physics" knob. See DESIGN.md §6.
type Config struct {
	StartingCapital int
	BasePrice       map[Resource]int
	Spread          float64

	// Walk: p' = p + RevertRate*(base-p) + p*Sigma*N(0,1), clamped to [ClampMin, ClampMax]*base.
	RevertRate float64
	Sigma      float64
	ClampMin   float64
	ClampMax   float64

	// Market depth: the player's own trades move prices against them. The first
	// FreeDepth[r] cases bought (or sold) recently trade at the plain quote; beyond that each
	// case moves the price by ImpactSlope, up to ImpactCap. Bought and sold cases are tracked
	// separately, so buying only raises the ask and selling only lowers the bid. Each night
	// the remembered volume falls by Recovery.
	FreeDepth   map[Resource]int
	ImpactSlope float64
	Recovery    float64
	ImpactCap   float64

	// ResaleRate is the share of a building's build cost returned when it is sold.
	ResaleRate float64

	MaxLevel    int
	MaxQuantity int

	HistoryLength int

	EventChance float64
	Events      []EventDef

	// WarehouseTiers and ProductionTiers are indexed by level-1 (level 1..MaxLevel).
	WarehouseTiers  []Tier
	ProductionTiers []Tier
}

// DefaultConfig returns the balance numbers from DESIGN.md §6.
func DefaultConfig() Config {
	return Config{
		StartingCapital: 1000,
		BasePrice: map[Resource]int{
			Lemon:    20,
			Sugar:    10,
			Ice:      10,
			Cup:      10,
			Lemonade: 90,
		},
		Spread:     0.10,
		RevertRate: 0.15,
		Sigma:      0.12,
		ClampMin:   0.25,
		ClampMax:   4.0,
		FreeDepth: map[Resource]int{
			Lemon: 80, Sugar: 80, Ice: 80, Cup: 80, Lemonade: 80,
		},
		ImpactSlope:   0.003,
		Recovery:      0.5,
		ImpactCap:     0.6,
		ResaleRate:    0.5,
		MaxLevel:      4,
		MaxQuantity:   10,
		HistoryLength: 14,
		EventChance:   0.25,
		Events: []EventDef{
			{
				Key:         "heat_wave",
				Name:        "Heat Wave",
				Description: "Scorching days drive lemonade demand and ice consumption.",
				Duration:    2,
				Multipliers: map[Resource]float64{Lemonade: 1.4, Ice: 1.3},
				Excludes:    []string{"rainy_week"}, // no heat wave in the rain
			},
			{
				Key:         "rainy_week",
				Name:        "Rainy Week",
				Description: "Nobody wants lemonade in the rain.",
				Duration:    3,
				Multipliers: map[Resource]float64{Lemonade: 0.75},
			},
			{
				Key:         "lemon_blight",
				Name:        "Lemon Blight",
				Description: "A crop blight drives up lemon prices.",
				Duration:    3,
				Multipliers: map[Resource]float64{Lemon: 1.7},
			},
			{
				Key:         "sugar_glut",
				Name:        "Sugar Glut",
				Description: "A bumper harvest floods the sugar market.",
				Duration:    2,
				Multipliers: map[Resource]float64{Sugar: 0.7},
			},
			{
				Key:         "holiday",
				Name:        "Holiday",
				Description: "A local holiday brings out thirsty crowds.",
				Duration:    1,
				Multipliers: map[Resource]float64{Lemonade: 1.35},
			},
			{
				Key:         "cup_shortage",
				Name:        "Cup Shortage",
				Description: "A supplier shortage drives up cup prices.",
				Duration:    2,
				Multipliers: map[Resource]float64{Cup: 1.5},
			},
		},
		WarehouseTiers: []Tier{
			{Name: "Pantry", Size: 10, BuildCost: 100, UpgradeCost: 100, Upkeep: 2},
			{Name: "Garage", Size: 20, BuildCost: 300, UpgradeCost: 250, Upkeep: 6},
			{Name: "Barn", Size: 40, BuildCost: 800, UpgradeCost: 600, Upkeep: 16},
			{Name: "Industrial Warehouse", Size: 80, BuildCost: 2000, UpgradeCost: 0, Upkeep: 40},
		},
		ProductionTiers: []Tier{
			{Name: "Kitchen", Size: 10, BuildCost: 500, UpgradeCost: 1000, Upkeep: 20},
			{Name: "Food Truck", Size: 20, BuildCost: 1500, UpgradeCost: 2500, Upkeep: 50},
			{Name: "Bottling Plant", Size: 40, BuildCost: 4000, UpgradeCost: 6000, Upkeep: 120},
			{Name: "Lemonade Factory", Size: 80, BuildCost: 10000, UpgradeCost: 0, Upkeep: 280},
		},
	}
}
