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
	// Empire: the first day the bot was in era k (index k, 2 to 5), its net worth then,
	// and its net worth at the start of each day (index = day-1).
	eraDay [6]int
	eraNW  [6]int
	nw     []int
	// buyouts is how many rivals the bot bought.
	buyouts int
	// flatRun is the longest stretch of days its net worth did not rise.
	flatRun int
}

// The cash cushion a careful owner keeps before reinvesting: days of the new upkeep plus a
// share of a full batch. Above level 1 a bad stretch costs far more, so the cushion is
// longer (2 days at level 1, 6 above; late game phase 0 found 2 bankrupted one careful
// player in five once upgrades paid off).
func reserveFor(g Game, cfg Config) int {
	days := 2
	if g.ProductionLevel > 1 {
		days = 6
	}
	return days*TotalUpkeep(g, cfg) + unitCost(g, cfg)*ProductionCapacity(g, cfg)*3/10
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
	// upgrades: also buys upgrades whose payback at base prices is short (see upgraderRule).
	upgrades bool
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
		// A careful player stops where the market stops paying: no batch case that loses money.
		capacityBinds := true
		if best := profitableBatch(g, cfg, n, 0); best < n {
			n, capacityBinds = best, false
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
			need := reserveFor(gc, cfg)
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
		for capacityBinds { // only add capacity when capacity is what limits profit
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
		if p.upgrades {
			buyPayingUpgrades(&g, cfg)
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
		if _, err := simEndDay(&g, cfg); err != nil {
			break
		}
		if g.Status == StatusBankrupt {
			res.bankrupt = d
			break
		}
	}
	return res
}

// upgrader is the careful grower that also buys upgrades: whenever an upgrade would pay
// for itself within upgraderPayback days at base prices and the cash cushion allows it.
func upgrader(cfg Config, seed int64, days int) simResult {
	return owner(cfg, seed, days, player{upgrades: true})
}

// upgraderPayback is the longest payback, in days, at which the upgrader buys.
const upgraderPayback = 20

// upgradeDailyGain is what an upgrade would add per day at base prices, for the effects
// a steady producer can use: extra output from a yield bonus, and the price a deeper
// market saves on a full day's sales. Effects that only help a player who trades
// around them (freezer, forecast, quality of life) are worth nothing to this bot.
func upgradeDailyGain(g Game, cfg Config, u UpgradeDef) float64 {
	gain := 0.0
	capacity := ProductionCapacity(g, cfg)
	bid := float64(cfg.BasePrice[Lemonade]) * (1 - cfg.Spread)
	for _, e := range u.Effects {
		switch e.Kind {
		case "yield_bonus":
			gain += e.Value / 100 * float64(capacity) * bid
		case "depth_bonus":
			if e.Target != string(Lemonade) {
				continue
			}
			with := g.Clone()
			with.Upgrades[u.Key] = 1
			steady := func(x *Game) int {
				x.SellPressure[Lemonade] = float64(capacity) // what a full day of sales leaves at dawn
				return QuoteSell(*x, cfg, Lemonade, capacity, false).Total
			}
			base := g.Clone()
			gain += float64(steady(&with) - steady(&base))
		}
	}
	return gain
}

// buyPayingUpgrades buys every upgrade that pays back fast enough and leaves the same cash
// cushion as any other reinvestment. Locked and unaffordable ones are skipped.
func buyPayingUpgrades(g *Game, cfg Config) {
	// The jump to level 2 (a market three and a half times deeper) comes first: a level-1
	// business that spends its savings on gadgets stays stuck at level 1 and starves.
	if g.WarehouseLevel < 2 {
		return
	}
	for _, u := range cfg.Upgrades {
		if g.Owns(u.Key) || u.Cost > g.Capital || UpgradeLock(*g, cfg, u) != nil {
			continue
		}
		gain := upgradeDailyGain(*g, cfg, u) - float64(u.Upkeep)
		if gain <= 0 || float64(u.Cost)/gain > upgraderPayback {
			continue
		}
		after := g.Clone()
		after.Upgrades[u.Key] = 1
		// A long cushion (12 days of upkeep): an upgrade cannot be sold back.
		cushion := 12*TotalUpkeep(after, cfg) + unitCost(after, cfg)*ProductionCapacity(after, cfg)*3/10
		if g.Capital-u.Cost >= cushion {
			_ = BuyUpgrade(g, cfg, u.Key)
		}
	}
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
		if _, err := simEndDay(&g, cfg); err != nil {
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
		if _, err := simEndDay(&g, cfg); err != nil {
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
		for _, d := range []int{5, 10, 15, 20, 30, 45, 60, 90, 120, 150, 200} {
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
	for _, d := range []int{5, 10, 15, 20, 30, 45, 60, 90, 120, 150, 200} {
		if len(at[d]) > 0 {
			fmt.Printf(" d%d=$%d", d, median(at[d]))
		}
	}
	fmt.Println()
}

func TestBalanceReport(t *testing.T) {
	if os.Getenv("BALANCE_REPORT") == "" {
		t.Skip("set BALANCE_REPORT=1 to print the balance report")
	}
	cfg := DefaultConfig()
	report("grower", cfg, grower, 120, 200)
	report("upgrader", cfg, upgrader, 120, 200)
	upgraderSummary(cfg, 200)
	report("sloppy", cfg, sloppy, 45, 200)
	report("careless", cfg, careless, 60, 200)
	report("idle", cfg, idle, 100, 50)
	fmt.Println("--- exploit bots (balance handoff) ---")
	report("spammer", cfg, spammer, 120, 200)
	report("thresholder", cfg, thresholder, 120, 200)
	report("strict-thr", cfg, thresholderStrict, 120, 200)
	report("hoarder", cfg, hoarder, 120, 200)
	report("opportunist", cfg, opportunist, 120, 200)
	exploitSummary(cfg, 200)
	avg, neg, thin, p5, p50, p95 := marginStats(cfg, 300, 200)
	fmt.Printf("margin per lemonade (bid - input asks): avg $%.1f | median $%d | p5 $%d p95 $%d | unprofitable %.1f%% of days | under $10: %.1f%%\n", avg, p50, p5, p95, neg, thin)
}

// upgraderSummary compares the upgrader with the careful grower it would otherwise be:
// the median of the day-90 capital ratio, and how often it is ahead.
func upgraderSummary(cfg Config, seeds int) {
	var ratio []int
	ahead, bankrupt := 0, 0
	for s := int64(1); s <= int64(seeds); s++ {
		a, b := capAt(upgrader(cfg, s, 90), 90), capAt(grower(cfg, s, 90), 90)
		if b > 0 {
			ratio = append(ratio, 100*a/b)
		}
		if a > b {
			ahead++
		}
		if a == 0 {
			bankrupt++
		}
	}
	fmt.Printf("upgrader vs grower at day 90: median %d%% of the grower's capital | ahead in %d%% of seeds | upgrader bankrupt %d%%\n",
		median(ratio), 100*ahead/seeds, 100*bankrupt/seeds)
}

// exploitSummary prints the numbers the balance targets are stated in: how often the
// max-volume spammer fails by day 90, and how the skilled bots compare with it.
func exploitSummary(cfg Config, seeds int) {
	poor, spammerWins := 0, 0
	var ratio []int // thresholder / spammer day-60 capital, in percent
	for s := int64(1); s <= int64(seeds); s++ {
		sp, th := spammer(cfg, s, 90), thresholder(cfg, s, 90)
		if sp.bankrupt > 0 || capAt(sp, 90) < 1000 {
			poor++
		}
		a, b := capAt(sp, 60), capAt(th, 60)
		if a > b {
			spammerWins++
		}
		if a > 0 {
			ratio = append(ratio, 100*b/a)
		}
	}
	beat := 0
	for s := int64(1); s <= int64(seeds); s++ {
		if capAt(spammer(cfg, s, 90), 60) > capAt(opportunist(cfg, s, 90), 60) {
			beat++
		}
	}
	fmt.Printf("spammer beats opportunist at day 60: %d%% of seeds\n", 100*beat/seeds)
	fmt.Printf("spammer bankrupt or under $1000 at day 90: %d%% of seeds | spammer beats thresholder at day 60: %d%% | thresholder/spammer day-60 capital, median: %d%%\n",
		100*poor/seeds, 100*spammerWins/seeds, median(ratio))
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

// No single upgrade may carry the late game: knocking out any one of the upgrades the
// upgrader can buy today must not delay "everything maxed" by more than 10 days, and the
// upgrader must beat the plain grower at day 90 without going bankrupt more often.
func TestBalanceNoSingleUpgradeIsIndispensable(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	const seeds = 20
	medianMaxed := func(cfg Config) int {
		var days []int
		for s := int64(1); s <= seeds; s++ {
			if d := upgrader(cfg, s, 120).maxedDay; d > 0 {
				days = append(days, d)
			} else {
				days = append(days, 121)
			}
		}
		return median(days)
	}
	full := DefaultConfig()
	all := medianMaxed(full)
	for _, u := range full.Upgrades {
		if u.Requires.Era > Era(Game{}, full) {
			continue // locked until Empire lands: the bot cannot buy it
		}
		cfg := full
		cfg.Upgrades = nil
		for _, o := range full.Upgrades {
			if o.Key != u.Key {
				cfg.Upgrades = append(cfg.Upgrades, o)
			}
		}
		if without := medianMaxed(cfg); without-all > 10 {
			t.Errorf("without %s everything maxes on day %d instead of %d: one upgrade is worth more than 10 days", u.Key, without, all)
		}
	}
}

func TestBalanceUpgraderBeatsTheGrowerWithoutMoreBankruptcies(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	cfg := DefaultConfig()
	var ratio []int
	deadU, deadG := 0, 0
	const seeds = 60
	for s := int64(1); s <= seeds; s++ {
		u, g := upgrader(cfg, s, 90), grower(cfg, s, 90)
		if u.bankrupt > 0 {
			deadU++
		}
		if g.bankrupt > 0 {
			deadG++
		}
		if b := capAt(g, 90); b > 0 {
			ratio = append(ratio, 100*capAt(u, 90)/b)
		}
	}
	if m := median(ratio); m < 105 || m > 145 {
		t.Errorf("the upgrader has %d%% of the grower's day-90 capital, want about 110 to 130%%", m)
	}
	if deadU > deadG+seeds/20 {
		t.Errorf("upgrader bankruptcies %d, grower %d of %d", deadU, deadG, seeds)
	}
}
