package domain

import (
	"math"

	"lemonade-api/internal/domain/content"
)

// The effect hooks. Owned upgrades carry typed effects (content/upgrades.go); the rest
// of the domain reads them only through the helpers here and never looks at an
// upgrade key. Each effect kind has exactly one hook:
//
//	ice_keep_cases       IceKeep            (freezer rotation, endday.go)
//	event_damp           eventMultiplier    (every price, quotes.go)
//	event_floor          eventMultiplier
//	forecast_days        Forecast
//	depth_bonus(_pct)    freeDepth          (impact.go)
//	input_depth_pct      freeDepth
//	input_discount_pct   Quotes             (asks of inputs)
//	upkeep_discount_pct  WarehouseUpkeep, ProductionUpkeep
//	yield_bonus          produce            (whole cases, fractions carry over)
//	use_discount         produce            (likewise)
//	storage_bonus_pct    Capacity
//	shelf_life_days      ExtraShelfLife     (read by perishables, Products B)
//	make                 makeInputs         (start of production)
//	presence_bonus       presenceBonus      (share contest, rivals.go)
//	hub_upkeep_discount  HubUpkeep          (empire.go)
//	unlock, qol          HasUnlock, HasFeature

// effectsOf returns the effects of one kind carried by owned upgrades, in table order.
func effectsOf(g Game, cfg Config, kind string) []Effect {
	if len(g.Upgrades) == 0 {
		return nil
	}
	var out []Effect
	for _, u := range cfg.Upgrades {
		if !g.Owns(u.Key) {
			continue
		}
		for _, e := range u.Effects {
			if e.Kind == kind {
				out = append(out, e)
			}
		}
	}
	return out
}

// sumEffects adds up the values of a kind, for effects whose target matches (an empty
// target argument matches all).
func sumEffects(g Game, cfg Config, kind, target string) float64 {
	total := 0.0
	for _, e := range effectsOf(g, cfg, kind) {
		if target == "" || e.Target == target {
			total += e.Value
		}
	}
	return total
}

// IceKeep is how many cases of ice the freezers keep one extra night (the largest owned).
func IceKeep(g Game, cfg Config) int {
	best := 0.0
	for _, e := range effectsOf(g, cfg, content.EffIceKeep) {
		best = math.Max(best, e.Value)
	}
	return int(best)
}

// eventMultiplier is the strength of an active event on commodity r for this player:
// the event's own multiplier, scaled by any damping upgrade and floored by insurance.
// It changes what the player sees and trades at, never the event walk itself.
func eventMultiplier(g Game, cfg Config, e ActiveEvent, r Resource) (float64, bool) {
	m, ok := e.Multipliers[r]
	if !ok {
		return 1, false
	}
	if len(g.Upgrades) == 0 {
		return m, true
	}
	for _, d := range effectsOf(g, cfg, content.EffEventDamp) {
		if d.Target == e.Key {
			m = 1 + (m-1)*d.Value
		}
	}
	if m < 1 {
		if c, found := cfg.Commodity(r); found && c.Product {
			for _, f := range effectsOf(g, cfg, content.EffEventFloor) {
				m = math.Max(m, f.Value)
			}
		}
	}
	return m, true
}

// effectivePriceFor is the player's whole-dollar price for r: the walked price with every
// active event applied (see eventMultiplier), clamped to a $1 minimum.
func effectivePriceFor(g Game, cfg Config, walked float64, r Resource) int {
	price := walked
	for _, e := range g.Events {
		if m, ok := eventMultiplier(g, cfg, e, r); ok {
			price *= m
		}
	}
	price *= priceDrift(g)
	rounded := int(math.Round(price))
	if rounded < 1 {
		rounded = 1
	}
	return rounded
}

// inputDiscountPct is the percent off every input's ask.
func inputDiscountPct(g Game, cfg Config, r Resource) float64 {
	if len(g.Upgrades) == 0 {
		return 0
	}
	if c, ok := cfg.Commodity(r); !ok || !c.Input {
		return 0
	}
	return sumEffects(g, cfg, content.EffInputDiscountPct, "")
}

// depthWithUpgrades is the free depth of r after depth upgrades, given the level base.
func depthWithUpgrades(g Game, cfg Config, r Resource, base int) int {
	if len(g.Upgrades) == 0 {
		return base
	}
	pct := sumEffects(g, cfg, content.EffDepthBonusPct, string(r))
	if c, ok := cfg.Commodity(r); ok && c.Input {
		pct += sumEffects(g, cfg, content.EffInputDepthPct, "")
	}
	cases := sumEffects(g, cfg, content.EffDepthBonus, string(r))
	if pct == 0 && cases == 0 {
		return base
	}
	return int(math.Round(float64(base)*(1+pct/100))) + int(cases)
}

// facilityUpkeepAfterDiscount applies the upkeep discounts for one facility type
// ("production" or "warehouse") to a raw upkeep.
func facilityUpkeepAfterDiscount(g Game, cfg Config, kind string, raw int) int {
	if len(g.Upgrades) == 0 {
		return raw
	}
	pct := sumEffects(g, cfg, content.EffUpkeepDiscountPct, "all") + sumEffects(g, cfg, content.EffUpkeepDiscountPct, kind)
	if pct <= 0 {
		return raw
	}
	return int(math.Round(float64(raw) * (100 - math.Min(pct, 100)) / 100))
}

// storageBonusPct is the percent more capacity for a storage class.
func storageBonusPct(g Game, cfg Config, r Resource) float64 {
	if len(g.Upgrades) == 0 {
		return 0
	}
	c, ok := cfg.Commodity(r)
	if !ok {
		return 0
	}
	return sumEffects(g, cfg, content.EffStoragePct, c.StorageClass)
}

// ExtraShelfLife is how many extra days perishables of a storage class keep.
func ExtraShelfLife(g Game, cfg Config, class string) int {
	return int(sumEffects(g, cfg, content.EffShelfLife, class))
}

// HasUnlock says whether an owned upgrade unlocks a feature or recipe (target).
func HasUnlock(g Game, cfg Config, target string) bool {
	for _, e := range effectsOf(g, cfg, content.EffUnlock) {
		if e.Target == target {
			return true
		}
	}
	return false
}

// HasFeature says whether an owned upgrade turns on a convenience feature.
func HasFeature(g Game, cfg Config, feature string) bool {
	for _, e := range effectsOf(g, cfg, content.EffQoL) {
		if e.Target == feature {
			return true
		}
	}
	return false
}

// Features lists every convenience feature turned on, in table order.
func Features(g Game, cfg Config) []string {
	var out []string
	for _, e := range effectsOf(g, cfg, content.EffQoL) {
		out = append(out, e.Target)
	}
	return out
}

// takeSavings is how many whole cases of r a batch of `use` cases saves, given the
// use-discount upgrades. The fractional remainder carries to the next batch, so the
// saving is whole cases and deterministic.
func (g *Game) takeSavings(cfg Config, r Resource, use int) int {
	pct := sumEffects(*g, cfg, content.EffUseDiscount, string(r))
	if pct <= 0 || use <= 0 {
		return 0
	}
	whole := g.carry("save:"+string(r), float64(use)*pct/100, use)
	return whole
}

// yieldExtra is the bonus cases a run of `out` cases of recipe output makes, limited by
// `room` (free space); the remainder carries over like takeSavings.
func (g *Game) yieldExtra(cfg Config, recipeKey string, out, room int) int {
	pct := sumEffects(*g, cfg, content.EffYield, recipeKey)
	if pct <= 0 || out <= 0 {
		return 0
	}
	return g.carry("yield:"+recipeKey, float64(out)*pct/100, max(room, 0))
}

// carry adds amount to the fractional remainder stored under key and takes out the whole
// cases, at most limit. The fraction left stays; anything over the limit is dropped.
func (g *Game) carry(key string, amount float64, limit int) int {
	if g.Carry == nil {
		g.Carry = make(map[string]float64)
	}
	x := g.Carry[key] + amount
	whole := math.Floor(x + 1e-9)
	g.Carry[key] = math.Max(x-whole, 0)
	if g.Carry[key] < 1e-9 {
		delete(g.Carry, key)
	}
	return min(int(whole), limit)
}

// ForecastEntry is one upcoming event the player can see coming.
type ForecastEntry struct {
	DaysAhead int
	Key       string
	Name      string
	Duration  int
}

// Forecast lists the events that will start on each of the next days the player's
// forecast upgrades reach. It is exact: it draws from the same seeded stream EndDay
// will use (SaltMarket, seed and day), on a copy of the events, and touches nothing
// else, so the real walk is never disturbed. A player with no forecast sees nothing.
func Forecast(g Game, cfg Config) []ForecastEntry {
	if len(g.Upgrades) == 0 {
		return nil
	}
	weatherDays, allDays := 0, 0
	for _, e := range effectsOf(g, cfg, content.EffForecast) {
		if e.Target == "all" {
			allDays = max(allDays, int(e.Value))
		} else {
			weatherDays = max(weatherDays, int(e.Value))
		}
	}
	horizon := max(weatherDays, allDays)
	if horizon == 0 {
		return nil
	}
	kinds := make(map[string]string, len(cfg.Events))
	for _, d := range cfg.Events {
		kinds[d.Key] = d.Kind
	}
	sim := Game{Events: append([]ActiveEvent(nil), g.Events...)}
	var out []ForecastEntry
	for d := 1; d <= horizon; d++ {
		rng := dayRNG(g.Seed, g.Day+d, SaltMarket)
		_, spawned := tickEvents(&sim, rng, cfg)
		for _, e := range spawned {
			visible := allDays >= d || (weatherDays >= d && kinds[e.Key] == EventWeather)
			if visible {
				out = append(out, ForecastEntry{DaysAhead: d, Key: e.Key, Name: e.Name, Duration: e.DaysLeft})
			}
		}
	}
	return out
}
