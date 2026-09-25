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
		candidates := eligibleEvents(cfg.Events, g.Events)
		if len(candidates) > 0 {
			def := candidates[rng.Intn(len(candidates))]
			active := ActiveEvent{
				Key:         def.Key,
				Name:        def.Name,
				Description: def.Description,
				Multipliers: def.Multipliers,
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
