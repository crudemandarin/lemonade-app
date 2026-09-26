package domain

import "math/rand"

// tickEvents advances every active event's countdown by one day, removing any
// that expire, then may spawn one new event (SPEC rule 20). An event already
// active is never re-spawned, so multipliers never stack with themselves.
func tickEvents(g *Game, rng *rand.Rand, cfg Config) (expired, spawned []ActiveEvent) {
	remaining := make([]ActiveEvent, 0, len(g.Events))
	for _, e := range g.Events {
		e.DaysLeft--
		if e.DaysLeft <= 0 {
			expired = append(expired, e)
		} else {
			remaining = append(remaining, e)
		}
	}
	g.Events = remaining

	if rng.Float64() < cfg.EventChance {
		candidates := eventsInEra(eligibleEvents(cfg.Events, g.Events), Era(*g, cfg))
		if len(candidates) > 0 {
			def := candidates[rng.Intn(len(candidates))]
			active := ActiveEvent{
				Key:         def.Key,
				Name:        def.Name,
				Description: def.Description,
				Multipliers: spreadToDrinks(*g, cfg, def),
				DaysLeft:    def.Duration,
			}
			g.Events = append(g.Events, active)
			spawned = append(spawned, active)
		}
	}
	return expired, spawned
}

// eligibleEvents returns the event definitions that may start now: not already
// active, and not in conflict (in either direction, see conflicts and
// EventDef.Excludes) with an active event.
func eligibleEvents(all []EventDef, active []ActiveEvent) []EventDef {
	activeKeys := make(map[string]bool, len(active))
	for _, e := range active {
		activeKeys[e.Key] = true
	}
	out := make([]EventDef, 0, len(all))
	for _, d := range all {
		if !activeKeys[d.Key] && !conflictsWithActive(d, all, activeKeys) {
			out = append(out, d)
		}
	}
	return out
}

// conflicts reports whether two events pull the same resource's price in
// opposite directions (one multiplier above 1, the other below). Such events
// never run together. It is derived from the multipliers, so a new row in the
// event table is checked automatically; EventDef.Excludes adds explicit pairs.
func conflicts(a, b EventDef) bool {
	for r, ma := range a.Multipliers {
		mb, ok := b.Multipliers[r]
		if ok && (ma > 1 && mb < 1 || ma < 1 && mb > 1) {
			return true
		}
	}
	return false
}

// conflictsWithActive reports whether d conflicts with an active event: derived
// from multipliers, or listed in either definition's Excludes.
func conflictsWithActive(d EventDef, all []EventDef, activeKeys map[string]bool) bool {
	for _, key := range d.Excludes {
		if activeKeys[key] {
			return true
		}
	}
	for _, other := range all {
		if !activeKeys[other.Key] {
			continue
		}
		if conflicts(d, other) {
			return true
		}
		for _, key := range other.Excludes {
			if key == d.Key {
				return true
			}
		}
	}
	return false
}

// eventsInEra keeps the events that may start in the given era.
func eventsInEra(defs []EventDef, era int) []EventDef {
	out := make([]EventDef, 0, len(defs))
	for _, d := range defs {
		if d.Era <= era {
			out = append(out, d)
		}
	}
	return out
}

// spreadToDrinks is an event's multipliers, with the lemonade multiplier also applied to
// every other cold drink (a product made with ice) the player has unlocked, when the event
// is a Drinks one. The event keeps its own map: the definition is never changed.
func spreadToDrinks(g Game, cfg Config, def EventDef) map[Resource]float64 {
	m, ok := def.Multipliers[Lemonade]
	if !def.Drinks || !ok {
		return def.Multipliers
	}
	out := make(map[Resource]float64, len(def.Multipliers)+4)
	for r, v := range def.Multipliers {
		out[r] = v
	}
	for _, rec := range KnownRecipes(g, cfg) {
		if rec.Output != Lemonade && recipeUses(rec, Ice) > 0 {
			out[rec.Output] = m
		}
	}
	return out
}
