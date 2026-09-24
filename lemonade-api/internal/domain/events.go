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

// eligibleEvents returns the event definitions not already active.
func eligibleEvents(all []EventDef, active []ActiveEvent) []EventDef {
	activeKeys := make(map[string]bool, len(active))
	for _, e := range active {
		activeKeys[e.Key] = true
	}
	out := make([]EventDef, 0, len(all))
	for _, d := range all {
		if !activeKeys[d.Key] {
			out = append(out, d)
		}
	}
	return out
}
