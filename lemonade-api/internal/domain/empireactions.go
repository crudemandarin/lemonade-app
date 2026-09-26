package domain

import "math"

// The empire's player actions: enter a territory, buy out a rival, run a campaign.

// EnterTerritory pays the entry cost and takes the entry share. The previous territory
// on the ladder must be entered first. Rivals of the new territory appear at their
// catalog shares, which with the entry share add up to 100.
func EnterTerritory(g *Game, cfg Config, key string) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	SeedEmpire(g, cfg)
	def, i, ok := territoryDef(cfg, key)
	if !ok {
		return ErrUnknownTerritory
	}
	if g.Territories[key].Entered {
		return ErrAlreadyEntered
	}
	if i > 0 && !g.Territories[cfg.Territories[i-1].Key].Entered {
		return ErrTerritoryLocked
	}
	if def.EntryCost > g.Capital {
		return ErrInsufficientFunds
	}
	g.Capital -= def.EntryCost
	g.Territories[key] = TerritoryState{Entered: true, Share: def.EntryShare}
	seedRivals(g, cfg, key)
	g.record(TimelinePoint{Day: g.Day, Kind: PointEnterTerritory, Resource: Resource(key), Amount: def.EntryCost})
	return nil
}

// BuyoutPrice is what buying rival key out costs now, in whole dollars, and whether it
// is a discounted merger offer. hostile pays the hostile premium and ignores refusals.
func BuyoutPrice(g Game, cfg Config, key string, hostile bool) (price int, offer bool) {
	r, ok := g.Rivals[key]
	if !ok {
		return 0, false
	}
	def, _ := rivalDef(cfg, key)
	value := math.Max(r.Valuation, MinBuyoutValue(g, cfg, key)) * cycleValuationFactor(g)
	premium := cfg.FriendlyPremium
	if def.FriendlyPremium > 0 {
		premium = def.FriendlyPremium
	}
	if hostile {
		premium = cfg.HostilePremium
	} else if r.OfferDaysLeft > 0 {
		premium *= cfg.MergerDiscount
		offer = true
	}
	return int(math.Round(value * premium)), offer
}

// BuyOut pays valuation times the premium for a rival and takes its whole share. A
// friendly buyout can be refused (*RivalRefusesError) until the player holds enough of
// the territory; a hostile one (1.6 times) never is. The price counts in net worth at
// ResaleRate; the rival's buildings are not modelled.
func BuyOut(g *Game, cfg Config, key string, hostile bool) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	SeedEmpire(g, cfg)
	r, ok := g.Rivals[key]
	def, defOK := rivalDef(cfg, key)
	if !ok || !defOK {
		return ErrUnknownRival
	}
	if r.Status != RivalActive {
		return ErrRivalGone
	}
	t := g.Territories[def.Territory]
	if !t.Entered {
		return ErrNotEntered
	}
	if !hostile && def.RefuseBelowShare > 0 && t.Share < def.RefuseBelowShare {
		return &RivalRefusesError{Key: key, NeedShare: def.RefuseBelowShare}
	}
	price, _ := BuyoutPrice(*g, cfg, key, hostile)
	if price > g.Capital {
		return ErrInsufficientFunds
	}
	g.Capital -= price
	t.Share = math.Min(100, t.Share+r.Share)
	g.Territories[def.Territory] = t
	r.Share, r.Status, r.PricePaid, r.Hostile = 0, RivalAcquired, price, hostile
	r.TelegraphKey, r.OfferDaysLeft, r.CampaignDaysLeft, r.Mood = "", 0, 0, MoodCalm
	g.Rivals[key] = r
	g.record(TimelinePoint{Day: g.Day, Kind: PointBuyout, Resource: Resource(key), Amount: price})
	return nil
}

// CampaignCost is what a campaign level costs in a territory, in whole dollars.
func CampaignCost(cfg Config, territory string, level int) int {
	d, _, ok := territoryDef(cfg, territory)
	if !ok || level < 1 || level > len(cfg.Campaigns) {
		return 0
	}
	return int(math.Round(cfg.Campaigns[level-1].CostPerDepth * float64(d.Depth)))
}

// RunCampaign spends cash for a presence bonus in one territory for a few days. Levels
// are 1 to len(Campaigns). Only one campaign runs in a territory at a time.
func RunCampaign(g *Game, cfg Config, territory string, level int) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	SeedEmpire(g, cfg)
	if _, _, ok := territoryDef(cfg, territory); !ok {
		return ErrUnknownTerritory
	}
	t := g.Territories[territory]
	if !t.Entered {
		return ErrNotEntered
	}
	if level < 1 || level > len(cfg.Campaigns) {
		return ErrInvalidCampaign
	}
	if t.CampaignDaysLeft > 0 {
		return ErrCampaignRunning
	}
	cost := CampaignCost(cfg, territory, level)
	if cost > g.Capital {
		return ErrInsufficientFunds
	}
	g.Capital -= cost
	c := cfg.Campaigns[level-1]
	t.CampaignDaysLeft, t.CampaignBonus = c.Days, c.Bonus
	g.Territories[territory] = t
	g.record(TimelinePoint{Day: g.Day, Kind: PointCampaign, Resource: Resource(territory), Amount: cost})
	return nil
}

// ShareDepth is the lemonade depth a rival's whole share would add to the player's reach
// if the player held it: the territory's depth per share point times the points.
func ShareDepth(g Game, cfg Config, territory string, points float64) float64 {
	d, i, ok := territoryDef(cfg, territory)
	if !ok {
		return 0
	}
	if i == 0 {
		return float64(cfg.FreeDepth[Lemonade]) * levelMultiplier(g, cfg) * points / cfg.NeighborhoodStartShare
	}
	return float64(d.Depth) * points / 100
}

// MinBuyoutValue is the least a rival is worth: MinPaybackDays of the profit its share
// would add at base prices. It keeps every buyout from repaying itself in a few days,
// whatever the rival's own valuation and however deep the player's warehouses make the
// Neighborhood.
func MinBuyoutValue(g Game, cfg Config, key string) float64 {
	def, ok := rivalDef(cfg, key)
	r, have := g.Rivals[key]
	if !ok || !have {
		return 0
	}
	return cfg.MinPaybackDays * ShareDepth(g, cfg, def.Territory, r.Share) * float64(cfg.BuyoutMargin)
}
