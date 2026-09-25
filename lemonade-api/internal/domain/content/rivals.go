package content

// Personalities. They are data read by the rival tick, not code branches.
const (
	Passive     = "passive"
	Aggressive  = "aggressive"
	Premium     = "premium"
	Opportunist = "opportunist"
	Integrated  = "integrated"
)

// RivalDef is one rival business (late game content, table C). Rivals are stat
// blocks: a share, a valuation and a mood, never a simulated game.
type RivalDef struct {
	Key         string
	Name        string
	Territory   string
	Personality string
	// Share is the starting share in percent. Per territory the rivals add up to
	// 100 minus the player's entry share.
	Share float64
	// Buyout is the starting valuation in whole dollars.
	Buyout int
	// FriendlyPremium multiplies the valuation for a friendly buyout (1.2 unless set).
	FriendlyPremium float64
	// RefuseBelowShare: refuses a friendly buyout until the player holds this share
	// of the territory, in percent (0 never refuses).
	RefuseBelowShare float64
	// TelegraphDays is how many days ahead this rival announces its moves (0 uses the
	// default of one day).
	TelegraphDays int
	// SupplyResources are the input commodities an integrated rival controls: while it
	// is active their depth for the player is multiplied by SupplyFactor, and once the
	// player acquires it by AcquiredFactor (0 means no change).
	SupplyResources []string
	SupplyFactor    float64
	AcquiredFactor  float64
	// Flavor is one line of story.
	Flavor string
}

// Rivals is the roster. Keys are stable snake_case.
var Rivals = []RivalDef{
	{Key: "lil_lucy", Name: "Lil' Lucy's Lemonade", Territory: "neighborhood", Personality: Passive, Share: 15, Buyout: 2500, FriendlyPremium: 1.0, Flavor: "A kid's stand. She sells to you gladly."},
	{Key: "sour_sam", Name: "Sour Sam's Stand", Territory: "neighborhood", Personality: Aggressive, Share: 25, Buyout: 6000, RefuseBelowShare: 60, Flavor: "Starts price wars on holidays."},
	{Key: "squeeze_box", Name: "The Squeeze Box", Territory: "neighborhood", Personality: Opportunist, Share: 20, Buyout: 4500, Flavor: "Offers a merger when it keeps losing share."},

	{Key: "main_st_squeeze", Name: "Main Street Squeeze", Territory: "city", Personality: Passive, Share: 30, Buyout: 60000, Flavor: "The established local chain; slow and steady."},
	{Key: "cafe_citron", Name: "Cafe Citron", Territory: "city", Personality: Premium, Share: 15, Buyout: 55000, Flavor: "Premium cafe; immune to price wars."},
	{Key: "zest_express", Name: "Zest Express", Territory: "city", Personality: Aggressive, Share: 25, Buyout: 50000, Flavor: "A fleet of food trucks that campaigns against the leader."},
	{Key: "pucker_up", Name: "Pucker Up Co.", Territory: "city", Personality: Opportunist, Share: 20, Buyout: 40000, Flavor: "Social-media darling; share swings with the mood."},

	{Key: "sunbelt_bev", Name: "Sunbelt Beverages", Territory: "region", Personality: Passive, Share: 30, Buyout: 600000, Flavor: "Regional bottler."},
	{Key: "golden_grove", Name: "Golden Grove Co-op", Territory: "region", Personality: Integrated, Share: 20, Buyout: 450000, SupplyResources: []string{"lemon"}, SupplyFactor: 0.8, AcquiredFactor: 1.2, Flavor: "Owns the orchards."},
	{Key: "valley_fresh", Name: "Valley Fresh", Territory: "region", Personality: Premium, Share: 17, Buyout: 400000, Flavor: "Organic brand; holds share stubbornly."},
	{Key: "tart_and_co", Name: "Tart & Co.", Territory: "region", Personality: Aggressive, Share: 15, Buyout: 300000, Flavor: "Aggressive discounter."},
	{Key: "big_pour", Name: "Big Pour Regional", Territory: "region", Personality: Opportunist, Share: 10, Buyout: 220000, Flavor: "Overextended; likely to fold."},

	{Key: "national_lemon_works", Name: "National Lemon Works", Territory: "nation", Personality: Passive, Share: 30, Buyout: 5000000, Flavor: "The old giant."},
	{Key: "fizz_nation", Name: "FizzNation", Territory: "nation", Personality: Aggressive, Share: 20, Buyout: 3500000, Flavor: "Bottled-sparkling specialist."},
	{Key: "yellowbrand", Name: "Yellowbrand", Territory: "nation", Personality: Premium, Share: 20, Buyout: 4000000, Flavor: "A household name."},
	{Key: "squeezo", Name: "Squeezo", Territory: "nation", Personality: Opportunist, Share: 15, Buyout: 2500000, Flavor: "Tech-y delivery app; makes merger offers."},
	{Key: "harvest_holdings", Name: "Harvest Holdings", Territory: "nation", Personality: Integrated, Share: 10, Buyout: 2000000, SupplyResources: []string{"sugar"}, SupplyFactor: 0.8, Flavor: "Controls the sugar contracts."},

	{Key: "global_citrus", Name: "Global Citrus Holdings", Territory: "world", Personality: Passive, Share: 35, Buyout: 60000000, Flavor: "The final boss."},
	{Key: "megapour", Name: "MegaPour International", Territory: "world", Personality: Aggressive, Share: 25, Buyout: 40000000, TelegraphDays: 2, Flavor: "Global price wars, telegraphed early."},
	{Key: "lemonopoly", Name: "Lemonopoly Group", Territory: "world", Personality: Premium, Share: 22, Buyout: 45000000, RefuseBelowShare: 25, Flavor: "Refuses a buyout until you own a quarter of the World."},
	{Key: "zestcorp", Name: "Zestcorp", Territory: "world", Personality: Integrated, Share: 15, Buyout: 25000000, SupplyResources: []string{"lemon", "lime"}, SupplyFactor: 0.85, Flavor: "Controls the citrus supply."},
}
