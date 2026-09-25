package domain

import "math"

// Market depth. The market is not bottomless: buying a lot pushes the ask up and selling
// a lot pushes the bid down, so a player who moves a big volume every day pays for it.
// Each resource remembers how many cases the player recently bought and sold (separately,
// so no buy-then-sell round trip can profit), and forgets half of it overnight.

func snapUp(x float64) int   { return int(math.Ceil(snap(x))) }
func snapDown(x float64) int { return int(math.Floor(snap(x))) }

// impactMove is the fractional price move for the k-th case (1-based) traded now, given
// what was already traded: zero inside the free depth, then ImpactSlope per case, capped.
func impactMove(cfg Config, r Resource, pressure float64, k int) float64 {
	excess := pressure + float64(k) - float64(cfg.FreeDepth[r])
	if excess <= 0 {
		return 0
	}
	return math.Min(cfg.ImpactSlope*excess, cfg.ImpactCap)
}

// unitAsk is what the k-th case bought now costs: the plain ask, raised by the impact.
func unitAsk(cfg Config, r Resource, plain int, pressure float64, k int) int {
	m := impactMove(cfg, r, pressure, k)
	if m == 0 {
		return plain
	}
	return snapUp(float64(plain) * (1 + m))
}

// unitBid is what the k-th case sold now raises: the plain bid, lowered by the impact,
// never below $1.
func unitBid(cfg Config, r Resource, plain int, pressure float64, k int) int {
	m := impactMove(cfg, r, pressure, k)
	if m == 0 {
		return plain
	}
	if b := snapDown(float64(plain) * (1 - m)); b > 1 {
		return b
	}
	return 1
}

// MarginalAsk and MarginalBid are the price of the next single case, which is what the
// market row shows. They equal the plain quote until the free depth is used up.
func MarginalAsk(g Game, cfg Config, r Resource) int {
	return unitAsk(cfg, r, Quotes(g, cfg)[r].Ask, g.BuyPressure[r], 1)
}

func MarginalBid(g Game, cfg Config, r Resource) int {
	return unitBid(cfg, r, Quotes(g, cfg)[r].Bid, g.SellPressure[r], 1)
}

// FreeDepthLeft is how many more cases can be bought (or sold) at the plain price.
func FreeDepthLeft(cfg Config, r Resource, pressure float64) int {
	return max(0, int(math.Floor(float64(cfg.FreeDepth[r])-pressure)))
}

// TradeQuote is what a purchase or sale of Qty cases costs or raises, in whole dollars.
type TradeQuote struct {
	// Qty is how many cases the quote covers (fewer than asked when clamped).
	Qty int
	// Total is the cost (buy) or proceeds (sell) with price impact.
	Total int
	// PlainTotal is the same trade at the plain quote, for measuring slippage.
	PlainTotal int
}

// Average is the mean price per case with impact; 0 for an empty quote.
func (q TradeQuote) Average() float64 {
	if q.Qty == 0 {
		return 0
	}
	return float64(q.Total) / float64(q.Qty)
}

// Slippage is how much worse than the plain quote the trade is, as a fraction (0.04 is
// 4%): dearer for a buy, cheaper for a sale.
func (q TradeQuote) Slippage() float64 {
	if q.PlainTotal == 0 {
		return 0
	}
	return math.Abs(float64(q.Total-q.PlainTotal)) / float64(q.PlainTotal)
}

// QuoteBuy prices buying qty cases of r now. With clamp it buys as many as it can, up to
// qty, limited by cash and free warehouse space; without it the quote is for exactly qty
// and ignores both. It never changes the game.
func QuoteBuy(g Game, cfg Config, r Resource, qty int, clamp bool) TradeQuote {
	plain := Quotes(g, cfg)[r].Ask
	space := Capacity(g, cfg, r) - g.Inventory[r]
	var q TradeQuote
	for k := 1; k <= qty; k++ {
		price := unitAsk(cfg, r, plain, g.BuyPressure[r], k)
		if clamp && (k > space || q.Total+price > g.Capital) {
			break
		}
		q.Qty, q.Total, q.PlainTotal = k, q.Total+price, q.PlainTotal+plain
	}
	return q
}

// QuoteSell prices selling qty cases of r now. With clamp it sells as many as are held,
// up to qty; without it the quote is for exactly qty.
func QuoteSell(g Game, cfg Config, r Resource, qty int, clamp bool) TradeQuote {
	plain := Quotes(g, cfg)[r].Bid
	var q TradeQuote
	for k := 1; k <= qty; k++ {
		if clamp && k > g.Inventory[r] {
			break
		}
		q.Qty, q.Total, q.PlainTotal = k, q.Total+unitBid(cfg, r, plain, g.SellPressure[r], k), q.PlainTotal+plain
	}
	return q
}

func (g *Game) addPressure(buy bool, r Resource, qty int) {
	m := &g.SellPressure
	if buy {
		m = &g.BuyPressure
	}
	if *m == nil {
		*m = make(map[Resource]float64, len(Resources))
	}
	(*m)[r] += float64(qty)
}

// forgetPressure lets the market forget part of the player's recent volume overnight.
func (g *Game) forgetPressure(cfg Config) {
	for _, m := range []map[Resource]float64{g.BuyPressure, g.SellPressure} {
		for r, p := range m {
			if p *= 1 - cfg.Recovery; p < 0.01 {
				p = 0
			}
			m[r] = p
		}
	}
}

// buyCost is what qty cases cost now, stopping as soon as the total passes limit (so a
// huge qty is cheap to reject).
func buyCost(g Game, cfg Config, r Resource, qty, limit int) int {
	plain := Quotes(g, cfg)[r].Ask
	total := 0
	for k := 1; k <= qty; k++ {
		total += unitAsk(cfg, r, plain, g.BuyPressure[r], k)
		if total > limit {
			break
		}
	}
	return total
}
