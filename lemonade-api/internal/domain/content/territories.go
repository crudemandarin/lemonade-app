package content

// TerritoryDef is one rung of the territory ladder (late game content, table A).
// Era is the rung number: entering a territory puts the player in that era.
type TerritoryDef struct {
	Key  string
	Name string
	Era  int
	// Depth is the territory's lemonade demand in cases a day; the player's reach in
	// it is Depth times their share.
	Depth int
	// EntryShare is the player's share, in percent, on entering.
	EntryShare float64
	// EntryCost is the price of entering, in whole dollars (0 for the start).
	EntryCost int
	// HubUpkeep is the daily upkeep of the distribution hub.
	HubUpkeep int
	// BuildingCap is extra buildings per type this territory allows.
	BuildingCap int
}

// Territories is the ladder, in order. Each needs the one before it.
var Territories = []TerritoryDef{
	{Key: "neighborhood", Name: "The Neighborhood", Era: 1, Depth: 200, EntryShare: 40, EntryCost: 0, HubUpkeep: 0, BuildingCap: 10},
	{Key: "city", Name: "Citrus City", Era: 2, Depth: 800, EntryShare: 10, EntryCost: 15000, HubUpkeep: 150, BuildingCap: 10},
	{Key: "region", Name: "Sunbelt Region", Era: 3, Depth: 3000, EntryShare: 8, EntryCost: 120000, HubUpkeep: 900, BuildingCap: 10},
	{Key: "nation", Name: "The Nation", Era: 4, Depth: 12000, EntryShare: 5, EntryCost: 1000000, HubUpkeep: 5000, BuildingCap: 10},
	{Key: "world", Name: "The World", Era: 5, Depth: 50000, EntryShare: 3, EntryCost: 8000000, HubUpkeep: 30000, BuildingCap: 20},
}
