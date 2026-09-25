package domain

import "math"

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
	out := make(map[Resource]Quote, len(cfg.Commodities))
	for _, r := range cfg.Resources() {
		price := effectivePriceFor(g, cfg, g.Market[r].Price, r)
		out[r] = quote(price, cfg.Spread)
		if d := inputDiscountPct(g, cfg, r); d > 0 {
			q := out[r]
			q.Ask = int(math.Ceil(snap(float64(price) * (1 + cfg.Spread) * (1 - d/100))))
			if q.Ask < q.Bid {
				q.Ask = q.Bid
			}
			out[r] = q
		}
	}
	return out
}
