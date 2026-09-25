package domain

import (
	"math"

	"lemonade-api/internal/domain/content"
)

// The empire: territories the player has entered, the share they hold in each, and the
// rivals who hold the rest. Shares are percent (0 to 100). Per territory the player's
// share plus the active rivals' shares is always 100. Prices stay global; territories
// add depth (reach), see freeDepth in impact.go.

// Rival statuses.
const (
	RivalActive   = "active"
	RivalAcquired = "acquired"
	RivalFolded   = "folded"
)

// Rival moods.
const (
	MoodCalm       = "calm"
	MoodStruggling = "struggling"
	MoodHostile    = "hostile"
)

// TerritoryState is the player's position in one territory.
type TerritoryState struct {
	Entered bool
	// Share is the player's share in percent.
	Share float64
	// CampaignDaysLeft and CampaignBonus are the running marketing campaign.
	CampaignDaysLeft int
	CampaignBonus    float64
}

// RivalState is a rival's stat block. It is never a simulated game.
type RivalState struct {
	Share     float64
	Valuation float64
	// Momentum nudges the daily valuation growth (a small signed number).
	Momentum float64
	Status   string
	Mood     string
	// TelegraphKey is the rival event announced for TelegraphDay, empty if none.
	TelegraphKey string
	TelegraphDay int
	// LossStreak is how many days in a row the rival lost share.
	LossStreak int
	// StruggleDays counts days below the fold threshold.
	StruggleDays int
	// LastWarDay is the last day a price war started (cooldown).
	LastWarDay int
	// CampaignDaysLeft is a running rival campaign (presence bonus).
	CampaignDaysLeft int
	// FoldDaysLeft counts down while a folded rival's freed share goes to whoever has
	// more presence (Share is what is left to hand out).
	FoldDaysLeft int
	// OfferDaysLeft is a running merger offer (buyout at a discount).
	OfferDaysLeft int
	// PricePaid is what the player paid to acquire it; acquisitions count in net worth.
	PricePaid int
	// Hostile says the acquisition was a hostile takeover.
	Hostile bool
}

// Empire errors, mapped to 409s by the API layer.
var (
	ErrTerritoryLocked  = errorsNew("previous territory not entered")
	ErrAlreadyEntered   = errorsNew("territory already entered")
	ErrUnknownTerritory = errorsNew("unknown territory")
	ErrUnknownRival     = errorsNew("unknown rival")
	ErrRivalGone        = errorsNew("rival is not active")
	ErrNotEntered       = errorsNew("territory not entered")
	ErrInvalidCampaign  = errorsNew("unknown campaign level")
	ErrCampaignRunning  = errorsNew("a campaign is already running there")
)

// RivalRefusesError says a rival will not sell in a friendly buyout, and why.
type RivalRefusesError struct {
	Key string
	// NeedShare is the share of the territory the player must hold, in percent.
	NeedShare float64
}

func (e *RivalRefusesError) Error() string { return "rival refuses a friendly buyout" }

// SeedEmpire gives a game the empire it should have: a game saved before territories
// existed (or a new one) enters the Neighborhood at its start share with its catalog
// rivals. It is idempotent and safe to call on every load.
func SeedEmpire(g *Game, cfg Config) {
	if g.Territories == nil {
		g.Territories = map[string]TerritoryState{}
	}
	if g.Rivals == nil {
		g.Rivals = map[string]RivalState{}
	}
	if len(cfg.Territories) == 0 {
		return
	}
	first := cfg.Territories[0]
	if _, ok := g.Territories[first.Key]; !ok {
		g.Territories[first.Key] = TerritoryState{Entered: true, Share: first.EntryShare}
		seedRivals(g, cfg, first.Key)
	}
}

// seedRivals adds the catalog rivals of a territory at their starting shares.
func seedRivals(g *Game, cfg Config, territory string) {
	for _, d := range cfg.Rivals {
		if d.Territory != territory {
			continue
		}
		if _, ok := g.Rivals[d.Key]; ok {
			continue
		}
		g.Rivals[d.Key] = RivalState{
			Share: d.Share, Valuation: float64(d.Buyout) * cfg.ValuationMultiple,
			Status: RivalActive, Mood: MoodCalm,
		}
	}
}

func territoryDef(cfg Config, key string) (content.TerritoryDef, int, bool) {
	for i, d := range cfg.Territories {
		if d.Key == key {
			return d, i, true
		}
	}
	return content.TerritoryDef{}, 0, false
}

func rivalDef(cfg Config, key string) (content.RivalDef, bool) {
	for _, d := range cfg.Rivals {
		if d.Key == key {
			return d, true
		}
	}
	return content.RivalDef{}, false
}

// Era is the highest territory the player has entered (1 to 5).
func Era(g Game, cfg Config) int {
	era := 1
	for _, d := range cfg.Territories {
		if g.Territories[d.Key].Entered && d.Era > era {
			era = d.Era
		}
	}
	return era
}

// levelCap is the highest facility tier the game may reach now: the tiers of era 1 are
// open to everyone, later tiers unlock with the era. A game already above it (an old
// save) keeps its level.
func levelCap(g Game, cfg Config) int {
	era := Era(g, cfg)
	cap := 0
	for i, t := range cfg.ProductionTiers {
		if t.Era <= era {
			cap = i + 1
		}
	}
	for i, t := range cfg.WarehouseTiers {
		if t.Era > era && i+1 <= cap {
			cap = i
		}
	}
	return max(cap, g.ProductionLevel, g.WarehouseLevel)
}

// buildingCap is how many buildings of each type the player may own: the Neighborhood's
// cap, plus each territory entered. A game already above it keeps its buildings.
func buildingCap(g Game, cfg Config) int {
	total := cfg.MaxQuantity
	for _, d := range cfg.Territories[1:] {
		if g.Territories[d.Key].Entered {
			total += d.BuildingCap
		}
	}
	return total
}

// reach is the free depth of r from every territory the player holds. The Neighborhood
// is the phase 0 depth (warehouse level) scaled by share over the start share, so a
// game that never moves its 40% plays as it always did. Every other territory adds
// its lemonade depth times the player's share, scaled to the commodity.
func reach(g Game, cfg Config, r Resource) int {
	start := cfg.NeighborhoodStartShare
	share := g.Territories[cfg.Territories[0].Key].Share
	if _, ok := g.Territories[cfg.Territories[0].Key]; !ok {
		share = start
	}
	mult := 1.0
	if i := g.WarehouseLevel - 1; i >= 0 && i < len(cfg.DepthByLevel) {
		mult = cfg.DepthByLevel[i]
	}
	total := float64(cfg.FreeDepth[r]) * mult * (share / start)
	ratio := float64(cfg.FreeDepth[r]) / float64(cfg.FreeDepth[Lemonade])
	for _, d := range cfg.Territories[1:] {
		if t := g.Territories[d.Key]; t.Entered {
			total += float64(d.Depth) * t.Share / 100 * ratio
		}
	}
	return int(math.Round(total * supplyFactor(g, cfg, r)))
}

// ReachIn is one territory's part of the reach of r, for the breakdown.
func ReachIn(g Game, cfg Config, territory string, r Resource) int {
	d, i, ok := territoryDef(cfg, territory)
	if !ok || !g.Territories[territory].Entered {
		return 0
	}
	if i == 0 {
		mult := 1.0
		if l := g.WarehouseLevel - 1; l >= 0 && l < len(cfg.DepthByLevel) {
			mult = cfg.DepthByLevel[l]
		}
		return int(math.Round(float64(cfg.FreeDepth[r]) * mult * g.Territories[d.Key].Share / cfg.NeighborhoodStartShare))
	}
	ratio := float64(cfg.FreeDepth[r]) / float64(cfg.FreeDepth[Lemonade])
	return int(math.Round(float64(d.Depth) * g.Territories[d.Key].Share / 100 * ratio))
}

// HubUpkeep is the daily upkeep of the distribution hubs, after the synergy discount
// (each territory beyond the first cuts it by HubSynergy, up to HubSynergyMax).
func HubUpkeep(g Game, cfg Config) int {
	total, entered := 0, 0
	for _, d := range cfg.Territories {
		if g.Territories[d.Key].Entered {
			total += d.HubUpkeep
			entered++
		}
	}
	discount := math.Min(cfg.HubSynergy*float64(max(entered-1, 0)), cfg.HubSynergyMax)
	upgrades := math.Min(sumEffects(g, cfg, content.EffHubUpkeepDiscount, ""), 100) / 100
	return int(math.Round(float64(total) * (1 - discount) * (1 - upgrades)))
}

// AcquisitionValue is what acquired rivals count for in net worth: ResaleRate times
// the price paid, like buildings. Entry costs and campaigns are not counted.
func AcquisitionValue(g Game, cfg Config) int {
	paid := 0
	for _, r := range g.Rivals {
		if r.Status == RivalAcquired {
			paid += r.PricePaid
		}
	}
	return int(cfg.ResaleRate * float64(paid))
}

// Reach is the free depth of r from every territory held: what the market panel and the
// empire strip call the player's reach.
func Reach(g Game, cfg Config, r Resource) int { return reach(g, cfg, r) }

// ReachAtLevel is Reach if the warehouses were at the given level, to preview an upgrade.
func ReachAtLevel(g Game, cfg Config, r Resource, level int) int {
	g.WarehouseLevel = level
	return reach(g, cfg, r)
}

// LevelCap is the highest facility tier the player may reach now (see levelCap).
func LevelCap(g Game, cfg Config) int { return levelCap(g, cfg) }

// BuildingCap is how many buildings of each type the player may own (see buildingCap).
func BuildingCap(g Game, cfg Config) int { return buildingCap(g, cfg) }

// Goal is the next thing to work toward, for the game page's empire strip.
type Goal struct {
	// Kind is "enter" (a territory), "buyout" (the cheapest rival left), or "" (done).
	Kind string
	Key  string
	Name string
	Cost int
}

// NextGoal is the next territory to enter, or once all are entered, the cheapest rival
// still standing.
func NextGoal(g Game, cfg Config) Goal {
	for _, d := range cfg.Territories {
		if !g.Territories[d.Key].Entered {
			return Goal{Kind: "enter", Key: d.Key, Name: d.Name, Cost: d.EntryCost}
		}
	}
	best := Goal{}
	for _, d := range cfg.Rivals {
		r, ok := g.Rivals[d.Key]
		if !ok || r.Status != RivalActive {
			continue
		}
		if price, _ := BuyoutPrice(g, cfg, d.Key, false); best.Kind == "" || price < best.Cost {
			best = Goal{Kind: "buyout", Key: d.Key, Name: d.Name, Cost: price}
		}
	}
	return best
}

// EraName is the name of the highest territory entered.
func EraName(g Game, cfg Config) string {
	name := cfg.Territories[0].Name
	for _, d := range cfg.Territories {
		if g.Territories[d.Key].Entered {
			name = d.Name
		}
	}
	return name
}

// TelegraphVisibleDays is how far ahead the player sees a rival's announced move: one
// day, more with the rival intel upgrade.
func TelegraphVisibleDays(g Game, cfg Config) int {
	if HasUnlock(g, cfg, "rival_intel") {
		return 2
	}
	return 1
}
