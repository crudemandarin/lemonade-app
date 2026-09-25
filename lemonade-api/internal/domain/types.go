package domain

// ResourceMarket is one resource's market state.
type ResourceMarket struct {
	// Price is the walked price: a float that mean-reverts toward the base price.
	Price float64
	// PreviousEffective is yesterday's whole-dollar effective price; nil on day 1.
	PreviousEffective *int
	// History is effective whole-dollar prices, oldest first, capped at Config.HistoryLength.
	History []int
}

// ActiveEvent is a spawned event still affecting the market.
type ActiveEvent struct {
	Key         string
	Name        string
	Description string
	Multipliers map[Resource]float64
	DaysLeft    int
}

// Game is the full mutable state of one player's playthrough. It is a plain value;
// callers own persistence. All mutating domain functions take *Game.
type Game struct {
	// RunID identifies this playthrough across saves and its finished record. Empty on
	// games saved before runs existed; the API assigns one on their first mutation.
	RunID   string
	Seed    int64
	Day     int
	Capital int
	Status  Status

	Inventory map[Resource]int
	// BuyPressure and SellPressure are the cases the player recently bought and sold, per
	// resource, which the market remembers and forgets overnight; see impact.go.
	BuyPressure  map[Resource]float64
	SellPressure map[Resource]float64
	// CostBasis is the total dollars paid for the stock of each resource (for
	// lemonade, the cost of the inputs it was made from). Average cost is basis / stock.
	CostBasis map[Resource]int

	WarehouseLevel int
	// WarehouseQty is the building count per resource's warehouse.
	WarehouseQty map[Resource]int

	ProductionLevel int
	ProductionQty   int

	Market map[Resource]*ResourceMarket
	Events []ActiveEvent

	// PriceLog is one point per day, for the price chart; see pricelog.go.
	PriceLog []PricePoint

	// Timeline and Stats record how the game went; see timeline.go.
	Timeline []TimelinePoint
	Stats    Stats

	// Upgrades maps an owned upgrade's key to its level (1: upgrades have one level
	// today). UpgradeSpend is what they cost in total. See upgrades.go.
	Upgrades     map[string]int
	UpgradeSpend int
	// IceOld is how many cases of the ice in stock are one night old (kept by a
	// freezer). Production and sales use the oldest ice first; see endday.go.
	IceOld int
	// Carry holds fractions of a case left over from yield bonuses and use discounts,
	// so output stays whole cases; see effects.go.
	Carry map[string]float64

	// Goals are run-scoped facts only achievements read; see goals.go.
	Goals GoalStats

	// NetWorthDay100 is the net worth on first reaching BoardDay (nil before, and for
	// runs that end earlier); it feeds the day-100 board. See milestones.go.
	NetWorthDay100 *int
}

// User identifies a player. Login is username-only (SPEC rule 22).
type User struct {
	ID       uint
	Username string
}

// Quote is one resource's tradable prices, all whole dollars.
type Quote struct {
	Price int
	Bid   int
	Ask   int
}

// PriceChange is one resource's effective price before/after an EndDay tick.
type PriceChange struct {
	Resource Resource
	Before   int
	After    int
}

// DayReport summarizes one EndDay transition. Day is the day that just ended.
type DayReport struct {
	Day       int
	Produced  int
	IceMelted int
	// IceKept is ice a freezer moved to tomorrow instead of letting it melt.
	IceKept    int
	UpkeepPaid int
	// UpgradeUpkeep is the part of UpkeepPaid that is due for upgrades (owed, not necessarily paid).
	UpgradeUpkeep int
	// IceMade is ice made overnight by an ice machine, and its cost.
	IceMade, IceMadeCost int
	// Pnl is the bookkeeper's profit and loss for the day; nil without the upgrade.
	Pnl *PnlLine
	// ForcedSale* describe stock sold at bid because cash alone could not cover upkeep.
	ForcedSaleCases    int
	ForcedSaleProceeds int
	CapitalBefore      int
	CapitalAfter       int
	PriceChanges       []PriceChange
	NewEvents          []ActiveEvent
	ExpiredEvents      []ActiveEvent
	Bankrupt           bool
}

// Clone returns a deep copy, so stores and tests never alias a live game's maps.
func (g Game) Clone() Game {
	c := g

	c.Inventory = make(map[Resource]int, len(g.Inventory))
	for k, v := range g.Inventory {
		c.Inventory[k] = v
	}

	c.BuyPressure = make(map[Resource]float64, len(g.BuyPressure))
	for k, v := range g.BuyPressure {
		c.BuyPressure[k] = v
	}
	c.SellPressure = make(map[Resource]float64, len(g.SellPressure))
	for k, v := range g.SellPressure {
		c.SellPressure[k] = v
	}

	c.CostBasis = make(map[Resource]int, len(g.CostBasis))
	for k, v := range g.CostBasis {
		c.CostBasis[k] = v
	}

	c.WarehouseQty = make(map[Resource]int, len(g.WarehouseQty))
	for k, v := range g.WarehouseQty {
		c.WarehouseQty[k] = v
	}

	c.Upgrades = make(map[string]int, len(g.Upgrades))
	for k, v := range g.Upgrades {
		c.Upgrades[k] = v
	}
	c.Carry = make(map[string]float64, len(g.Carry))
	for k, v := range g.Carry {
		c.Carry[k] = v
	}

	c.Market = make(map[Resource]*ResourceMarket, len(g.Market))
	for k, m := range g.Market {
		mc := *m
		mc.History = append([]int(nil), m.History...)
		if m.PreviousEffective != nil {
			p := *m.PreviousEffective
			mc.PreviousEffective = &p
		}
		c.Market[k] = &mc
	}

	c.Timeline = append([]TimelinePoint(nil), g.Timeline...)
	c.PriceLog = make([]PricePoint, len(g.PriceLog))
	for i, p := range g.PriceLog {
		p.Events = append([]string(nil), p.Events...)
		c.PriceLog[i] = p
	}

	c.Events = nil
	for _, e := range g.Events {
		ec := e
		ec.Multipliers = make(map[Resource]float64, len(e.Multipliers))
		for k, v := range e.Multipliers {
			ec.Multipliers[k] = v
		}
		c.Events = append(c.Events, ec)
	}
	c.Goals = g.Goals.clone()
	if g.NetWorthDay100 != nil {
		v := *g.NetWorthDay100
		c.NetWorthDay100 = &v
	}
	return c
}
