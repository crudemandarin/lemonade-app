// Package content holds the game's data tables: the commodities, the recipes and
// (as later tracks add them) upgrades, rivals and achievements. It is plain data
// with string keys and no imports from the domain; the domain reads these tables
// into Config, so content never depends on the rules that use it.
package content

// Storage classes. Warehouses are still per commodity (late game Products A); the
// class is recorded now so the catalog does not change shape when storage pools by
// class.
const (
	StorageDry      = "dry"
	StorageCold     = "cold"
	StorageFrozen   = "frozen"
	StorageFinished = "finished"
)

// StorageClassNames are the display names of the storage classes.
var StorageClassNames = map[string]string{
	StorageDry: "Dry store", StorageCold: "Cold room", StorageFrozen: "Freezer", StorageFinished: "Finished goods",
}

// Categories group commodities in the market panel.
const (
	CategoryIngredient = "ingredient"
	CategoryProduct    = "product"
)

// Shelf lives. A positive value is days before stock spoils.
const (
	Keeps        = 0  // never spoils
	MeltsNightly = -1 // gone at the end of every day (ice)
)

// CommodityDef is one tradable good.
type CommodityDef struct {
	Key          string
	Name         string
	Category     string
	StorageClass string
	// BasePrice is the long-run price the daily walk reverts to, in whole dollars.
	BasePrice int
	// ShelfLifeDays: Keeps, MeltsNightly, or days before stock spoils.
	ShelfLifeDays int
	// Input goods are consumed by recipes; Product goods are made by them.
	Input   bool
	Product bool
	// Order is the display order (ascending) in the market panel and charts.
	Order int
	// DepthPct is how deep this good's market is compared with lemonade's, in percent, and
	// is what sets its free depth (0 means the same as lemonade, for the original goods).
	DepthPct int
}

// Commodities is the catalog. The order of this slice is the display order, and
// the first five keep the order of the original fixed list (lemon, sugar, ice,
// cup, lemonade) so records stored as arrays still line up.
var Commodities = []CommodityDef{
	{Key: "lemon", Name: "Lemons", Category: CategoryIngredient, StorageClass: StorageCold, BasePrice: 20, ShelfLifeDays: Keeps, Input: true, Order: 10},
	{Key: "sugar", Name: "Sugar", Category: CategoryIngredient, StorageClass: StorageDry, BasePrice: 10, ShelfLifeDays: Keeps, Input: true, Order: 20},
	{Key: "ice", Name: "Ice", Category: CategoryIngredient, StorageClass: StorageFrozen, BasePrice: 10, ShelfLifeDays: MeltsNightly, Input: true, Order: 30},
	{Key: "cup", Name: "Cups", Category: CategoryIngredient, StorageClass: StorageDry, BasePrice: 10, ShelfLifeDays: Keeps, Input: true, Order: 40},
	{Key: "lemonade", Name: "Lemonade", Category: CategoryProduct, StorageClass: StorageFinished, BasePrice: 90, ShelfLifeDays: Keeps, Product: true, Order: 50},

	// Launch set (late game Products B): four ingredients, each unlocked by learning a recipe.
	// Fresh fruit and herbs spoil after a few days; lemons stay as they were. Depth is a
	// percentage of lemonade's.
	{Key: "lime", Name: "Limes", Category: CategoryIngredient, StorageClass: StorageCold, BasePrice: 18, ShelfLifeDays: 3, Input: true, Order: 60, DepthPct: 100},
	{Key: "mint", Name: "Mint", Category: CategoryIngredient, StorageClass: StorageCold, BasePrice: 12, ShelfLifeDays: 3, Input: true, Order: 70, DepthPct: 100},
	{Key: "honey", Name: "Honey", Category: CategoryIngredient, StorageClass: StorageDry, BasePrice: 24, ShelfLifeDays: Keeps, Input: true, Order: 80, DepthPct: 100},
	{Key: "strawberry", Name: "Strawberries", Category: CategoryIngredient, StorageClass: StorageCold, BasePrice: 30, ShelfLifeDays: 3, Input: true, Order: 90, DepthPct: 100},

	{Key: "limeade", Name: "Limeade", Category: CategoryProduct, StorageClass: StorageFinished, BasePrice: 88, ShelfLifeDays: Keeps, Product: true, Order: 100, DepthPct: 35},
	{Key: "mint_lemonade", Name: "Mint lemonade", Category: CategoryProduct, StorageClass: StorageFinished, BasePrice: 110, ShelfLifeDays: Keeps, Product: true, Order: 110, DepthPct: 30},
	{Key: "honey_lemonade", Name: "Honey lemonade", Category: CategoryProduct, StorageClass: StorageFinished, BasePrice: 115, ShelfLifeDays: Keeps, Product: true, Order: 120, DepthPct: 30},
	{Key: "strawberry_lemonade", Name: "Strawberry lemonade", Category: CategoryProduct, StorageClass: StorageFinished, BasePrice: 135, ShelfLifeDays: Keeps, Product: true, Order: 130, DepthPct: 45},
}

// LegacyOrder is the order of the original five commodities, used to read records
// stored as fixed arrays (timeline stock, price log) before the catalog existed.
var LegacyOrder = []string{"lemon", "sugar", "ice", "cup", "lemonade"}
