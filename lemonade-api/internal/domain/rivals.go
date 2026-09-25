package domain

import (
	"math"

	"lemonade-api/internal/domain/content"
)

// The rivals tick (EndDay step 10) and the rival events it starts. Rivals are stat
// blocks: each day per territory the player's presence is set against the rivals'
// strength and the share moves toward the stronger side by at most MaxShareShiftPerDay
// points. Rivals fight back only when the player is taking share, so a player who
// ignores them keeps their share (and never goes bankrupt on their account).

// Rival event keys.
const (
	EventPriceWar       = "price_war"
	EventRivalCampaign  = "rival_campaign"
	EventMergerOffer    = "merger_offer"
	EventRivalFolds     = "rival_folds"
	EventRivalStruggles = "rival_struggling"
)

const (
	priceWarChance      = 0.10 // per day per eligible aggressive rival
	holidayWarFactor    = 3.0  // Sour Sam loves a holiday
	rivalCampaignChance = 0.06
	hostileBidChance    = 0.05
	hostileBidBonus     = 0.3
	hostileBidDays      = 3
	foldDays            = 5
	momentumPull        = 0.9
	momentumNoise       = 0.002
)

// presence is the player's pull in one territory: a base of 1, brand upgrades, a
// running campaign, less a penalty when the player sells none of their depth.
func presence(g Game, cfg Config, territory string) float64 {
	p := 1.0 + presenceBonus(g, cfg, territory)
	if t := g.Territories[territory]; t.CampaignDaysLeft > 0 {
		p += t.CampaignBonus
	}
	if depth := freeDepth(g, cfg, Lemonade); depth > 0 {
		// Selling pressure after the night's forgetting is half of yesterday's sales.
		fill := math.Min(1, 2*g.SellPressure[Lemonade]/float64(depth))
		p -= cfg.FillPenalty * (1 - fill)
	}
	return p
}

// presenceBonus is the brand presence from upgrades (signs, billboards and the like):
// the presence_bonus effects that name the territory or "all", as a fraction.
func presenceBonus(g Game, cfg Config, territory string) float64 {
	return (sumEffects(g, cfg, content.EffPresenceBonus, territory) + sumEffects(g, cfg, content.EffPresenceBonus, "all")) / 100
}

// rivalStrength is one rival's pull, by personality, raised while it campaigns.
func rivalStrength(cfg Config, def content.RivalDef, r RivalState) float64 {
	s := cfg.RivalStrength[def.Personality]
	if s == 0 {
		s = 1
	}
	if r.CampaignDaysLeft > 0 {
		s += cfg.CampaignRivalBonus
	}
	return s
}

// activeRivals lists a territory's active rivals in catalog order.
func activeRivals(g Game, cfg Config, territory string) []content.RivalDef {
	var out []content.RivalDef
	for _, d := range cfg.Rivals {
		if r, ok := g.Rivals[d.Key]; ok && d.Territory == territory && r.Status == RivalActive && r.Share > 0 {
			out = append(out, d)
		}
	}
	return out
}

// avgStrength is the share-weighted strength of a territory's active rivals.
func avgStrength(g Game, cfg Config, territory string) float64 {
	sum, weight := 0.0, 0.0
	for _, d := range activeRivals(g, cfg, territory) {
		r := g.Rivals[d.Key]
		sum += r.Share * rivalStrength(cfg, d, r)
		weight += r.Share
	}
	if weight == 0 {
		return 0
	}
	return sum / weight
}

// shareFloor is the lowest the player's share can be pushed to in a territory.
func shareFloor(cfg Config, i int) float64 {
	if i == 0 {
		return cfg.NeighborhoodStartShare
	}
	return cfg.Territories[i].EntryShare * cfg.ShareFloor
}

// contest moves one territory's share for the day and returns the points the player
// gained (negative when rivals took some).
func contest(g *Game, cfg Config, i int) float64 {
	def := cfg.Territories[i]
	t := g.Territories[def.Key]
	rivals := activeRivals(*g, cfg, def.Key)
	if len(rivals) == 0 {
		return 0
	}
	edge := presence(*g, cfg, def.Key) - avgStrength(*g, cfg, def.Key)
	shift := math.Max(-cfg.MaxShareShiftPerDay, math.Min(cfg.MaxShareShiftPerDay, edge*cfg.ShiftPerEdge))

	switch {
	case shift > 0:
		pool := 0.0
		for _, d := range rivals {
			pool += g.Rivals[d.Key].Share
		}
		shift = math.Min(shift, pool)
		for _, d := range rivals {
			r := g.Rivals[d.Key]
			r.Share -= shift * r.Share / pool
			g.Rivals[d.Key] = r
		}
	case shift < 0:
		shift = -math.Min(-shift, math.Max(0, t.Share-shareFloor(cfg, i)))
		weight := 0.0
		for _, d := range rivals {
			weight += g.Rivals[d.Key].Share * rivalStrength(cfg, d, g.Rivals[d.Key])
		}
		for _, d := range rivals {
			r := g.Rivals[d.Key]
			r.Share += -shift * r.Share * rivalStrength(cfg, d, r) / weight
			g.Rivals[d.Key] = r
		}
	}
	t.Share += shift
	g.Territories[def.Key] = t
	return shift
}

// distributeFolded hands a folded rival's freed share to whoever has more presence.
func distributeFolded(g *Game, cfg Config, d content.RivalDef, i int) {
	r := g.Rivals[d.Key]
	chunk := r.Share
	if r.FoldDaysLeft > 1 {
		chunk = r.Share / float64(r.FoldDaysLeft)
	}
	r.FoldDaysLeft = max(r.FoldDaysLeft-1, 0)
	r.Share -= chunk
	if r.FoldDaysLeft == 0 {
		chunk, r.Share = chunk+r.Share, 0
	}
	g.Rivals[d.Key] = r

	rivals := activeRivals(*g, cfg, d.Territory)
	if len(rivals) == 0 || presence(*g, cfg, d.Territory) >= avgStrength(*g, cfg, d.Territory) {
		t := g.Territories[d.Territory]
		t.Share += chunk
		g.Territories[d.Territory] = t
		return
	}
	pool := 0.0
	for _, o := range rivals {
		pool += g.Rivals[o.Key].Share
	}
	for _, o := range rivals {
		s := g.Rivals[o.Key]
		s.Share += chunk * s.Share / pool
		g.Rivals[o.Key] = s
	}
}

func telegraphDays(cfg Config, d content.RivalDef) int {
	if d.TelegraphDays > 0 {
		return d.TelegraphDays
	}
	return cfg.TelegraphDays
}

func holiday(g Game) bool {
	for _, e := range g.Events {
		if e.Key == "holiday" {
			return true
		}
	}
	return false
}

func priceWarActive(g Game) bool {
	for _, e := range g.Events {
		if e.Key == EventPriceWar {
			return true
		}
	}
	return false
}

// stepRivals is EndDay step 10. It draws two numbers per rival in the game, in catalog
// order, from the rivals' own random streams, so the market's stream never moves.
func stepRivals(g *Game, cfg Config, report *DayReport) {
	SeedEmpire(g, cfg)
	valueRNG := dayRNG(g.Seed, g.Day, SaltRivals)
	eventRNG := dayRNG(g.Seed, g.Day, SaltRivalEvents)

	// 1. The share contest, territory by territory.
	gained := make([]float64, len(cfg.Territories))
	for i, def := range cfg.Territories {
		if g.Territories[def.Key].Entered {
			before := map[string]float64{}
			for _, k := range cfg.Rivals {
				if k.Territory == def.Key {
					before[k.Key] = g.Rivals[k.Key].Share
				}
			}
			gained[i] = contest(g, cfg, i)
			// Valuation follows share: a rival that shrinks is worth less.
			for _, k := range cfg.Rivals {
				if r, ok := g.Rivals[k.Key]; ok && k.Territory == def.Key && r.Status == RivalActive && before[k.Key] > 0 {
					r.Valuation *= r.Share / before[k.Key]
					g.Rivals[k.Key] = r
				}
			}
		}
	}

	// 2. Per rival: drift, streaks, events, folds. Rolls are drawn whatever happens.
	for _, d := range cfg.Rivals {
		r, ok := g.Rivals[d.Key]
		if !ok {
			continue
		}
		noise, roll := valueRNG.NormFloat64(), eventRNG.Float64()
		_, ti, _ := territoryDef(cfg, d.Territory)
		switch r.Status {
		case RivalFolded:
			if r.Share > 0 {
				distributeFolded(g, cfg, d, ti)
			}
			continue
		case RivalAcquired:
			continue
		}
		r.Momentum = r.Momentum*momentumPull + noise*momentumNoise
		r.Valuation *= 1 + cfg.RivalGrowth + r.Momentum
		if r.CampaignDaysLeft > 0 {
			r.CampaignDaysLeft--
		}
		if r.OfferDaysLeft > 0 {
			r.OfferDaysLeft--
		}
		if gained[ti] > 0 {
			r.LossStreak++
		} else {
			r.LossStreak = 0
		}

		t := g.Territories[d.Territory]
		threatened := t.Share > cfg.Territories[ti].EntryShare+1e-9 // the player is taking share
		if r.TelegraphKey != "" && r.TelegraphDay < g.Day {
			r.TelegraphKey = "" // stale: it was due yesterday and did not fire (rival gone)
		}
		if r.TelegraphKey == "" {
			r = telegraphEvent(*g, cfg, d, r, roll, threatened)
		}

		// A rival at a third of its value or less is struggling, then folds.
		if r.Valuation < cfg.FoldValuation*float64(d.Buyout)*cfg.ValuationMultiple {
			r.StruggleDays++
			r.Mood = MoodStruggling
			if r.StruggleDays >= cfg.FoldStruggleDays {
				r.Status, r.FoldDaysLeft, r.Mood = RivalFolded, foldDays, MoodCalm
				r.TelegraphKey, r.OfferDaysLeft, r.CampaignDaysLeft = "", 0, 0
			}
		} else {
			r.StruggleDays = 0
			switch {
			case r.CampaignDaysLeft > 0 || r.TelegraphKey != "":
				r.Mood = MoodHostile
			default:
				r.Mood = MoodCalm
			}
		}

		// An opportunist that keeps losing share offers to sell at a discount.
		if d.Personality == content.Opportunist && r.Status == RivalActive && r.OfferDaysLeft == 0 && r.LossStreak >= cfg.MergerLossStreak {
			r.OfferDaysLeft, r.LossStreak = cfg.MergerOfferDays, 0
		}
		g.Rivals[d.Key] = r
	}

	// 3. Campaigns run their course.
	for _, def := range cfg.Territories {
		if t := g.Territories[def.Key]; t.Entered && t.CampaignDaysLeft > 0 {
			t.CampaignDaysLeft--
			if t.CampaignDaysLeft == 0 {
				t.CampaignBonus = 0
			}
			g.Territories[def.Key] = t
		}
	}
}

// telegraphEvent decides whether a rival announces a move today, and which. It fires
// TelegraphDays later (fireRivalEvents), no matter what happens in between.
func telegraphEvent(g Game, cfg Config, d content.RivalDef, r RivalState, roll float64, threatened bool) RivalState {
	if r.Status != RivalActive || !threatened {
		return r
	}
	t := g.Territories[d.Territory]
	announce := func(key string) RivalState {
		r.TelegraphKey, r.TelegraphDay = key, g.Day+telegraphDays(cfg, d)
		return r
	}
	switch d.Personality {
	case content.Aggressive:
		chance := priceWarChance
		if holiday(g) {
			chance *= holidayWarFactor
		}
		if roll < chance && g.Day-r.LastWarDay >= cfg.PriceWarCooldown && !priceWarActive(g) {
			return announce(EventPriceWar)
		}
		if roll >= chance && roll < chance+rivalCampaignChance && r.CampaignDaysLeft == 0 {
			return announce(EventRivalCampaign)
		}
	case content.Opportunist, content.Premium, content.Integrated:
		// They only lure share back from a leader holding half or more of the territory.
		if t.Share >= 50 && roll < hostileBidChance && r.CampaignDaysLeft == 0 {
			return announce(EventRivalCampaign)
		}
	}
	return r
}

// fireRivalEvents runs the telegraphed rival events that are due today. It is called
// from the events step, after events tick and before the market moves, and draws no
// random numbers: a rival does exactly what it announced.
func fireRivalEvents(g *Game, cfg Config) {
	for _, d := range cfg.Rivals {
		r, ok := g.Rivals[d.Key]
		if !ok || r.TelegraphKey == "" || r.TelegraphDay != g.Day {
			continue
		}
		key := r.TelegraphKey
		r.TelegraphKey = ""
		if r.Status == RivalActive {
			switch key {
			case EventPriceWar:
				r.LastWarDay = g.Day
				if d.Personality != content.Premium && !priceWarActive(*g) {
					g.Events = append(g.Events, ActiveEvent{
						Key: EventPriceWar, Name: "Price war: " + d.Name,
						Description: d.Name + " slashes prices to win your customers.",
						Multipliers: map[Resource]float64{Lemonade: cfg.PriceWarMultiplier},
						DaysLeft:    priceWarDays(*g, cfg),
					})
				}
			case EventRivalCampaign:
				r.CampaignDaysLeft = cfg.CampaignRivalDays
			}
		}
		g.Rivals[d.Key] = r
	}
}

// priceWarDays is how long a price war lasts: half as long with the PR team.
func priceWarDays(g Game, cfg Config) int {
	if HasUnlock(g, cfg, "pr_team") {
		return max(1, cfg.PriceWarDays/2)
	}
	return cfg.PriceWarDays
}

// supplyFactor scales the depth of an input for the integrated rivals that control
// it: less while they are active, restored (or better) once the player buys them.
func supplyFactor(g Game, cfg Config, r Resource) float64 {
	f := 1.0
	for _, d := range cfg.Rivals {
		st, ok := g.Rivals[d.Key]
		if !ok || len(d.SupplyResources) == 0 {
			continue
		}
		for _, k := range d.SupplyResources {
			if Resource(k) != r {
				continue
			}
			switch {
			case st.Status == RivalActive:
				f *= d.SupplyFactor
			case st.Status == RivalAcquired && d.AcquiredFactor > 0:
				f *= d.AcquiredFactor
			}
		}
	}
	return f
}
