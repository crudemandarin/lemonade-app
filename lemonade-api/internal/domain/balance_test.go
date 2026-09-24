package domain

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"
)

// Balance checks. Simulated players (a growing business, a sloppy one, a careless
// one, and an idle one) play many seeded games against DefaultConfig, so tuning
// changes are judged by how the game plays, not by eyeballing numbers.
//
//   - The TestBalance* checks fail if a change breaks the intended feel of the game.
//   - To see the numbers behind them (and to re-tune), run:
//     BALANCE_REPORT=1 go test ./internal/domain -run TestBalanceReport -v

type simResult struct {
	capital   []int  // capital at the start of each day (index = day-1)
	bankrupt  int    // day of bankruptcy, 0 if none
	fullL1Day int    // first day every facility is at max count on level 1
	firstUp   int    // first day any upgrade was bought
	lvlDay    [5]int // first day both facility types reached level k (index k)
	maxedDay  int    // first day everything is at max level and max count
}

func unitCost(g Game, cfg Config) int {
	q := Quotes(g, cfg)
	return q[Lemon].Ask + q[Sugar].Ask + q[Ice].Ask + q[Cup].Ask
}

func sellAll(g *Game, cfg Config, r Resource) {
	if n := g.Inventory[r]; n > 0 {
		_ = Sell(g, cfg, r, n)
	}
}

// player sets how often the business owner slips up, per day.
type player struct {
	forgetIce float64 // buys the other inputs but forgets the ice, so no lemonade is made
	overspend float64 // invests without keeping a cash cushion
}

// grower plays carefully. sloppy makes the occasional mistake, as a human might.
func grower(cfg Config, seed int64, days int) simResult { return owner(cfg, seed, days, player{}) }
func sloppy(cfg Config, seed int64, days int) simResult {
	return owner(cfg, seed, days, player{forgetIce: 0.08, overspend: 0.06})
}

// owner sells everything each morning, buys a full batch, then reinvests in the
// cheapest way to add throughput (a balanced expansion or a balanced upgrade), but
// only if cash still covers upkeep after the purchase.
func owner(cfg Config, seed int64, days int, p player) simResult {
	g := NewGame(cfg, seed)
	mrng := rand.New(rand.NewSource(seed * 104729))
	res := simResult{}
	for d := 1; d <= days; d++ {
		res.capital = append(res.capital, g.Capital)
		sellAll(&g, cfg, Lemonade)

		// Today's batch.
		n := ProductionCapacity(g, cfg)
		for _, in := range Inputs {
			if free := Capacity(g, cfg, in) - g.Inventory[in]; free < n {
				n = free
			}
		}
		if free := Capacity(g, cfg, Lemonade) - g.Inventory[Lemonade]; free < n {
			n = free
		}
		if uc := unitCost(g, cfg); uc > 0 && g.Capital/uc < n {
			n = g.Capital / uc
		}
		forgetIce := mrng.Float64() < p.forgetIce
		for _, in := range Inputs {
			if in == Ice && forgetIce {
				continue
			}
			_ = Buy(&g, cfg, in, n)
		}

		// Reinvest only if, after the purchase, cash still covers two days of the NEW
		// upkeep plus working capital for a share of a full batch at the new rate.
		overspend := mrng.Float64() < p.overspend
		safe := func(cost int, apply func(*Game)) bool {
			if overspend {
				return g.Capital >= cost
			}
			gc := g.Clone()
			apply(&gc)
			need := 2*TotalUpkeep(gc, cfg) + unitCost(gc, cfg)*ProductionCapacity(gc, cfg)*3/10
			return g.Capital-cost >= need
		}
		doExpand := func(x *Game) {
			_ = Expand(x, cfg, Production, "")
			for _, r := range Resources {
				_ = Expand(x, cfg, Warehouse, r)
			}
		}
		doUpgrade := func(x *Game) {
			_ = Upgrade(x, cfg, Production)
			_ = Upgrade(x, cfg, Warehouse)
		}
		for {
			pl, wl := g.ProductionLevel, g.WarehouseLevel
			canExpand := g.ProductionQty < cfg.MaxQuantity
			for _, r := range Resources {
				if g.WarehouseQty[r] >= cfg.MaxQuantity {
					canExpand = false
				}
			}
			expandCost, expandGain := 1<<30, 0
			if canExpand && pl == wl {
				expandCost = productionTier(cfg, pl).BuildCost + 5*warehouseTier(cfg, wl).BuildCost
				expandGain = productionTier(cfg, pl).Size
			}
			upCost, upGain := 1<<30, 0
			if pl == wl && pl < cfg.MaxLevel {
				upCost = productionTier(cfg, pl).UpgradeCost*g.ProductionQty + warehouseTier(cfg, wl).UpgradeCost*warehouseBuildings(g)
				upGain = g.ProductionQty * (productionTier(cfg, pl+1).Size - productionTier(cfg, pl).Size)
			}
			preferExpand := expandGain > 0 && (upGain == 0 || float64(expandCost)/float64(expandGain) <= float64(upCost)/float64(upGain))
			switch {
			case preferExpand && safe(expandCost, doExpand):
				doExpand(&g)
				continue
			case !preferExpand && upGain > 0 && safe(upCost, doUpgrade):
				doUpgrade(&g)
				if res.firstUp == 0 {
					res.firstUp = d
				}
				continue
			}
			break
		}
		if res.fullL1Day == 0 && g.ProductionQty == cfg.MaxQuantity && g.ProductionLevel == 1 {
			res.fullL1Day = d
		}
		for k := 2; k <= cfg.MaxLevel; k++ {
			if res.lvlDay[k] == 0 && g.ProductionLevel >= k && g.WarehouseLevel >= k {
				res.lvlDay[k] = d
			}
		}
		if res.maxedDay == 0 && g.ProductionLevel == cfg.MaxLevel && g.WarehouseLevel == cfg.MaxLevel && g.ProductionQty == cfg.MaxQuantity {
			all := true
			for _, r := range Resources {
				if g.WarehouseQty[r] < cfg.MaxQuantity {
					all = false
				}
			}
			if all {
				res.maxedDay = d
			}
		}
		if _, err := EndDay(&g, cfg); err != nil {
			break
		}
		if g.Status == StatusBankrupt {
			res.bankrupt = d
			break
		}
	}
	return res
}

// careless: random buys, sells, and facility spending with no plan.
func careless(cfg Config, seed int64, days int) simResult {
	g := NewGame(cfg, seed)
	rng := rand.New(rand.NewSource(seed * 7919))
	res := simResult{}
	for d := 1; d <= days; d++ {
		res.capital = append(res.capital, g.Capital)
		for a := 0; a < 6; a++ {
			r := Resources[rng.Intn(len(Resources))]
			switch rng.Intn(5) {
			case 0, 1:
				_ = Buy(&g, cfg, r, 1+rng.Intn(10))
			case 2:
				_ = Sell(&g, cfg, r, 1+rng.Intn(10))
			case 3:
				_ = Expand(&g, cfg, Warehouse, r)
			case 4:
				if rng.Intn(3) == 0 {
					_ = Expand(&g, cfg, Production, "")
				}
			}
		}
		if _, err := EndDay(&g, cfg); err != nil {
			break
		}
		if g.Status == StatusBankrupt {
			res.bankrupt = d
			break
		}
	}
	return res
}

// idle: never acts, just ends the day.
func idle(cfg Config, seed int64, days int) simResult {
	g := NewGame(cfg, seed)
	res := simResult{}
	for d := 1; d <= days; d++ {
		res.capital = append(res.capital, g.Capital)
		if _, err := EndDay(&g, cfg); err != nil {
			break
		}
		if g.Status == StatusBankrupt {
			res.bankrupt = d
			break
		}
	}
	return res
}

func capAt(r simResult, day int) int {
	if day <= len(r.capital) {
		return r.capital[day-1]
	}
	return 0 // bankrupt before this day
}

func median(xs []int) int {
	if len(xs) == 0 {
		return -1
	}
	s := append([]int{}, xs...)
	sort.Ints(s)
	return s[len(s)/2]
}

func report(name string, cfg Config, bot func(Config, int64, int) simResult, days, seeds int) {
	at := map[int][]int{}
	var bankrupt, full, up, l2, l3, l4, maxed []int
	for s := int64(1); s <= int64(seeds); s++ {
		r := bot(cfg, s, days)
		for _, d := range []int{5, 10, 15, 20, 30, 45, 60} {
			if d <= len(r.capital) {
				at[d] = append(at[d], r.capital[d-1])
			}
		}
		if r.bankrupt > 0 {
			bankrupt = append(bankrupt, r.bankrupt)
		}
		if r.fullL1Day > 0 {
			full = append(full, r.fullL1Day)
		}
		if r.firstUp > 0 {
			up = append(up, r.firstUp)
		}
		if r.lvlDay[2] > 0 {
			l2 = append(l2, r.lvlDay[2])
		}
		if r.lvlDay[3] > 0 {
			l3 = append(l3, r.lvlDay[3])
		}
		if r.lvlDay[4] > 0 {
			l4 = append(l4, r.lvlDay[4])
		}
		if r.maxedDay > 0 {
			maxed = append(maxed, r.maxedDay)
		}
	}
	if len(maxed) > 0 || name == "grower" {
		fmt.Printf("  milestones (median day): L1 full %d | L2 %d | L3 %d | L4 %d | everything maxed %d  (%d%% maxed within %d days)\n",
			median(full), median(l2), median(l3), median(l4), median(maxed), 100*len(maxed)/seeds, days)
	}
	fmt.Printf("%-9s bankrupt %3d%% (median day %d) | maxed L1 by day %d (%d%% did) | 1st upgrade day %d | capital median:", name,
		100*len(bankrupt)/seeds, median(bankrupt), median(full), 100*len(full)/seeds, median(up))
	for _, d := range []int{5, 10, 15, 20, 30, 45, 60} {
		fmt.Printf(" d%d=$%d", d, median(at[d]))
	}
	fmt.Println()
}

func TestBalanceReport(t *testing.T) {
	if os.Getenv("BALANCE_REPORT") == "" {
		t.Skip("set BALANCE_REPORT=1 to print the balance report")
	}
	cfg := DefaultConfig()
	report("grower", cfg, grower, 90, 200)
	report("sloppy", cfg, sloppy, 45, 200)
	report("careless", cfg, careless, 60, 200)
	report("idle", cfg, idle, 100, 50)
	avg, neg, thin, p5, p50, p95 := marginStats(cfg, 300, 200)
	fmt.Printf("margin per lemonade (bid - input asks): avg $%.1f | median $%d | p5 $%d p95 $%d | unprofitable %.1f%% of days | under $10: %.1f%%\n", avg, p50, p5, p95, neg, thin)
}

// marginStats samples the market alone (idle player) and reports how often one
// lemonade is profitable: bid for lemonade minus the ask cost of its four inputs.
func marginStats(cfg Config, seeds, days int) (avg float64, pctNeg, pctThin float64, p5, p50, p95 int) {
	var ms []int
	for s := int64(1); s <= int64(seeds); s++ {
		g := NewGame(cfg, s)
		g.Capital = 1 << 40 // never bankrupt; we only want prices
		for d := 0; d < days; d++ {
			q := Quotes(g, cfg)
			ms = append(ms, q[Lemonade].Bid-(q[Lemon].Ask+q[Sugar].Ask+q[Ice].Ask+q[Cup].Ask))
			if _, err := EndDay(&g, cfg); err != nil {
				break
			}
		}
	}
	sort.Ints(ms)
	sum, neg, thin := 0, 0, 0
	for _, m := range ms {
		sum += m
		if m < 0 {
			neg++
		}
		if m < 10 {
			thin++
		}
	}
	n := len(ms)
	return float64(sum) / float64(n), 100 * float64(neg) / float64(n), 100 * float64(thin) / float64(n), ms[n/20], ms[n/2], ms[n*19/20]
}

// The checks below encode how the game is meant to feel. They use bands, not exact
// numbers, so small tuning changes pass and real regressions (too easy, too harsh,
// a stalled economy) fail. See the README's "Game physics" section.

func shareDead(bot func(Config, int64, int) simResult, cfg Config, seeds, days int) float64 {
	dead := 0
	for s := int64(1); s <= int64(seeds); s++ {
		if bot(cfg, s, days).bankrupt > 0 {
			dead++
		}
	}
	return float64(dead) / float64(seeds)
}

func TestBalanceCarefulPlayersMostlySurvive(t *testing.T) {
	if got := shareDead(grower, DefaultConfig(), 80, 60); got > 0.20 {
		t.Errorf("%.0f%% of careful players went bankrupt within 60 days, want at most 20%%: the game is too harsh", 100*got)
	}
}

func TestBalanceMistakesAreExpensive(t *testing.T) {
	got := shareDead(sloppy, DefaultConfig(), 80, 45)
	if got < 0.25 {
		t.Errorf("only %.0f%% of sloppy players went bankrupt within 45 days, want at least 25%%: losing is too hard", 100*got)
	}
	if got > 0.90 {
		t.Errorf("%.0f%% of sloppy players went bankrupt within 45 days, want at most 90%%: small mistakes are too deadly", 100*got)
	}
}

func TestBalanceCarelessPlayersLose(t *testing.T) {
	if got := shareDead(careless, DefaultConfig(), 80, 60); got < 0.80 {
		t.Errorf("only %.0f%% of careless players went bankrupt within 60 days, want at least 80%%", 100*got)
	}
}

func TestBalanceDoingNothingLoses(t *testing.T) {
	// No free lunch: fixed costs eat an idle player's savings in a few weeks, but not instantly.
	day := idle(DefaultConfig(), 1, 200).bankrupt
	if day < 20 || day > 60 {
		t.Errorf("an idle player went bankrupt on day %d, want between day 20 and 60", day)
	}
}

func TestBalanceMarketIsARealRisk(t *testing.T) {
	avg, unprofitable, _, _, _, _ := marginStats(DefaultConfig(), 100, 150)
	if avg < 12 || avg > 35 {
		t.Errorf("average margin per lemonade is $%.1f, want between $12 and $35", avg)
	}
	if unprofitable < 3 || unprofitable > 25 {
		t.Errorf("producing loses money on %.1f%% of days, want between 3%% and 25%%: the market is either riskless or a coin flip", unprofitable)
	}
}

func TestBalancePacing(t *testing.T) {
	cfg := DefaultConfig()
	var l1, maxed []int
	for s := int64(1); s <= 40; s++ {
		r := grower(cfg, s, 90)
		if r.fullL1Day > 0 {
			l1 = append(l1, r.fullL1Day)
		}
		if r.maxedDay > 0 {
			maxed = append(maxed, r.maxedDay)
		}
	}
	if d := median(l1); d < 15 || d > 40 {
		t.Errorf("a careful player fills level 1 by day %d (median), want between day 15 and 40", d)
	}
	if len(maxed) > 0 {
		if d := median(maxed); d < 45 {
			t.Errorf("a careful player has bought everything by day %d (median), want day 45 or later: there is too little left to work toward", d)
		}
	}
}
