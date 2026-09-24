package domain

import "math"

// effectivePrice rounds a resource's walked price, adjusted by the given active
// events' multipliers, to a whole dollar (SPEC rule 18), clamped to a $1 minimum.
func effectivePrice(walked float64, events []ActiveEvent, r Resource) int {
	price := walked
	for _, e := range events {
		if m, ok := e.Multipliers[r]; ok {
			price *= m
		}
	}
	rounded := int(math.Round(price))
	if rounded < 1 {
		rounded = 1
	}
	return rounded
}

// quote turns a whole-dollar effective price into bid/ask (SPEC rule 7).
func quote(price int, spread float64) Quote {
	bid := int(math.Floor(snap(float64(price) * (1 - spread))))
	if bid < 1 {
		bid = 1
	}
	ask := int(math.Ceil(snap(float64(price) * (1 + spread))))
	return Quote{Price: price, Bid: bid, Ask: ask}
}

// snap removes float noise (e.g. 100*1.1 = 110.00000000000001) so floor/ceil
// don't round an exact whole dollar the wrong way.
func snap(x float64) float64 {
	return math.Round(x*1e6) / 1e6
}

// Quotes returns each resource's effective price, bid, and ask.
func Quotes(g Game, cfg Config) map[Resource]Quote {
	out := make(map[Resource]Quote, len(Resources))
	for _, r := range Resources {
		price := effectivePrice(g.Market[r].Price, g.Events, r)
		out[r] = quote(price, cfg.Spread)
	}
	return out
}
