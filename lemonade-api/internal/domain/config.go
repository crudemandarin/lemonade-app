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
	// Era is the territory era that unlocks this tier for a new game. Tiers 1 to 4 are
	// era 1: phase 0 balanced them without territories, and saves already hold them.
	Era int
}

// Campaign is one level of marketing campaign: a temporary presence bonus in one
// territory. It costs CostPerDepth times the territory's lemonade depth.
type Campaign struct {
	Name         string
	Bonus        float64
	Days         int
	CostPerDepth float64
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

	// MaxLevel is the highest tier a game reaches without entering a territory (4).
	// Tiers 5 to 7 unlock by era; see levelCap. MaxQuantity is the building cap of the
	// Neighborhood; each territory entered adds its BuildingCap (see buildingCap).
	MaxLevel    int
	MaxQuantity int

	// Territories and Rivals are the empire catalog (internal/domain/content), in ladder
	// order. Shares are percent. See empire.go.
	Territories []content.TerritoryDef
	Rivals      []content.RivalDef
	// NeighborhoodStartShare is the player's share the phase 0 depth numbers assume.
	NeighborhoodStartShare float64
	// MaxShareShiftPerDay caps how far a territory's share moves in a day, in points.
	MaxShareShiftPerDay float64
	// ShiftPerEdge converts the presence edge (player presence minus rival strength) to
	// points of share a day, before the cap.
	ShiftPerEdge float64
	// ShareFloor is the lowest share rivals can push the player to, as a fraction of
	// the entry share (the Neighborhood never drops below its 40% start).
	ShareFloor float64
	// FillPenalty is the presence lost when the player sells none of a territory's
	// depth; it scales down to zero as the fill rate reaches 100%.
	FillPenalty float64
	// Rival strength: a rival's pull on the share, by personality.
	RivalStrength map[string]float64
	// FriendlyPremium and HostilePremium multiply a rival's valuation in a buyout.
	FriendlyPremium float64
	HostilePremium  float64
	// ValuationMultiple scales every catalog buyout price (a knob for balance).
	ValuationMultiple float64
	// RivalGrowth is a rival's daily valuation growth: buying earlier is cheaper.
	RivalGrowth float64
	// HubSynergy is the hub upkeep discount per territory beyond the first, up to
	// HubSynergyMax.
	HubSynergy    float64
	HubSynergyMax float64
	// Campaigns are the marketing campaign levels, cheapest first.
	Campaigns []Campaign
	// PriceWar and other rival event knobs; see rivalevents.go.
	PriceWarMultiplier float64
	PriceWarDays       int
	PriceWarCooldown   int
	CampaignRivalBonus float64
	CampaignRivalDays  int
	TelegraphDays      int
	FoldValuation      float64
	FoldStruggleDays   int
	MergerDiscount     float64
	MergerOfferDays    int
	MergerLossStreak   int

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
		DepthByLevel: []float64{1, 3.5, 6, 12, 12, 12, 12},
		ImpactShape:  0.24, // 0.3% per case at the level-1 depth of 80
		Recovery:     0.5,
		ImpactCap:    0.6,
		ResaleRate:   0.5,
		MaxLevel:     4,
		MaxQuantity:  10,
		Territories:  append([]content.TerritoryDef{}, content.Territories...),
		Rivals:       append([]content.RivalDef{}, content.Rivals...),

		NeighborhoodStartShare: 40,
		MaxShareShiftPerDay:    1,
		ShiftPerEdge:           10,
		ShareFloor:             0.5,
		FillPenalty:            0.05,
		RivalStrength: map[string]float64{
			content.Passive: 1.0, content.Aggressive: 1.08, content.Premium: 1.04,
			content.Opportunist: 1.0, content.Integrated: 1.0,
		},
		FriendlyPremium:   1.2,
		HostilePremium:    1.6,
		ValuationMultiple: 1,
		RivalGrowth:       0.004,
		HubSynergy:        0.05,
		HubSynergyMax:     0.20,
		Campaigns: []Campaign{
			{Name: "Flyers", Bonus: 0.05, Days: 5, CostPerDepth: 2.5},
			{Name: "Street team", Bonus: 0.10, Days: 7, CostPerDepth: 6},
			{Name: "Ad blitz", Bonus: 0.20, Days: 10, CostPerDepth: 15},
		},
		PriceWarMultiplier: 0.85,
		PriceWarDays:       3,
		PriceWarCooldown:   20,
		CampaignRivalBonus: 0.2,
		CampaignRivalDays:  5,
		TelegraphDays:      1,
		FoldValuation:      0.35,
		FoldStruggleDays:   3,
		MergerDiscount:     0.8,
		MergerOfferDays:    3,
		MergerLossStreak:   5,
		HistoryLength:      14,
		EventChance:        0.25,
		Upgrades:           append([]UpgradeDef(nil), content.Upgrades...),
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
			{Name: "Pantry", Size: 10, BuildCost: 100, UpgradeCost: 155, Upkeep: 2, Era: 1},
			{Name: "Garage", Size: 25, BuildCost: 220, UpgradeCost: 200, Upkeep: 4, Era: 1},
			{Name: "Barn", Size: 60, BuildCost: 450, UpgradeCost: 400, Upkeep: 8, Era: 1},
			{Name: "Industrial Warehouse", Size: 150, BuildCost: 900, UpgradeCost: 900, Upkeep: 15, Era: 1},
			{Name: "Distribution Center", Size: 400, BuildCost: 2000, UpgradeCost: 2000, Upkeep: 32, Era: 4},
			{Name: "Logistics Hub", Size: 1000, BuildCost: 4500, UpgradeCost: 4500, Upkeep: 65, Era: 5},
			{Name: "Automated Megastore", Size: 2500, BuildCost: 10000, UpgradeCost: 0, Upkeep: 130, Era: 5},
		},
		ProductionTiers: []Tier{
			{Name: "Kitchen", Size: 10, BuildCost: 500, UpgradeCost: 780, Upkeep: 20, Era: 1},
			{Name: "Food Truck", Size: 25, BuildCost: 1100, UpgradeCost: 1000, Upkeep: 40, Era: 1},
			{Name: "Bottling Plant", Size: 60, BuildCost: 2200, UpgradeCost: 2000, Upkeep: 75, Era: 1},
			{Name: "Lemonade Factory", Size: 150, BuildCost: 4500, UpgradeCost: 4500, Upkeep: 150, Era: 1},
			{Name: "Regional Plant", Size: 400, BuildCost: 10000, UpgradeCost: 9900, Upkeep: 320, Era: 4},
			{Name: "Mega Plant", Size: 1000, BuildCost: 22000, UpgradeCost: 21600, Upkeep: 650, Era: 5},
			{Name: "Global Network", Size: 2500, BuildCost: 48000, UpgradeCost: 0, Upkeep: 1300, Era: 5},
		},
	}
}
