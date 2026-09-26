package domain

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
)

// The diligent bot is the design's benchmark (late game design section 7): a player who
// uses the empire sensibly. It enters a territory when the depth it opens pays the entry
// cost back within paybackDays, buys the rival that is cheapest per point of share when
// cash allows with a cushion, runs a campaign only to defend a share that is slipping
// or under threat, buys the brand upgrades of the territories it is in, and grows its
// capacity as reach grows. The careful grower is the control that ignores all of this.

const (
	paybackDays     = 25  // enter a territory when its depth pays the cost back this fast
	buyoutPayback   = 60  // and a rival when the share it brings pays back this fast
	cushionDays     = 6   // days of upkeep kept in cash after any empire spend
	capacityToReach = 1.1 // keep production capacity about equal to reach
	sampleMargin    = 20  // dollars a case of lemonade earns at base prices, after impact, for planning
)

// growDiligent is growFast with the empire's caps: the building cap and tier cap now depend
// on the territories held.
func growDiligent(g *Game, cfg Config, reserve func(Game) int, res *simResult, day int) {
	for guard := 0; ; guard++ {
		if guard > 500 {
			panic(fmt.Sprintf("growDiligent loops: day %d cap %d pl %d/%d wl %d", day, g.Capital, g.ProductionLevel, g.ProductionQty, g.WarehouseLevel))
		}
		// Capacity about equal to reach: more than the market absorbs earns nothing, and
		// the cash is better saved for the next territory or rival.
		if float64(ProductionCapacity(*g, cfg)) >= capacityToReach*float64(freeDepth(*g, cfg, Lemonade)) {
			return
		}
		pl, wl := g.ProductionLevel, g.WarehouseLevel
		maxQ, maxL := buildingCap(*g, cfg), levelCap(*g, cfg)
		canExpand := g.ProductionQty < maxQ
		expandCost, expandGain := 1<<30, 0
		if canExpand && pl == wl {
			expandCost = expandStepCost(*g, cfg)
			expandGain = productionTier(cfg, pl).Size
		}
		upCost, upGain := 1<<30, 0
		if pl == wl && pl < maxL {
			upCost = productionTier(cfg, pl).UpgradeCost*g.ProductionQty + warehouseTier(cfg, wl).UpgradeCost*warehouseBuildings(*g)
			upGain = g.ProductionQty * (productionTier(cfg, pl+1).Size - productionTier(cfg, pl).Size)
		}
		preferExpand := expandGain > 0 && (upGain == 0 || float64(expandCost)/float64(expandGain) <= float64(upCost)/float64(upGain))
		safe := func(cost int, apply func(*Game)) bool {
			gc := g.Clone()
			apply(&gc)
			return g.Capital-cost >= reserve(gc)
		}
		doExpand := func(x *Game) {
			expandStep(x, cfg)
		}
		doUpgrade := func(x *Game) {
			_ = Upgrade(x, cfg, Production)
			_ = Upgrade(x, cfg, Warehouse)
		}
		switch {
		case preferExpand && safe(expandCost, doExpand):
			doExpand(g)
			continue
		case !preferExpand && upGain > 0 && safe(upCost, doUpgrade):
			doUpgrade(g)
			if res.firstUp == 0 {
				res.firstUp = day
			}
			continue
		}
		return
	}
}

// affordableBatch is the largest batch up to n whose four inputs, priced with the market's
// reaction to the buying, cost no more than the cash above two days of upkeep. A big
// business that buys a batch it cannot pay for ends up with three inputs and no fourth,
// makes nothing, and still owes the upkeep.
func affordableBatch(g Game, cfg Config, n int) int {
	budget := g.Capital - 2*TotalUpkeep(g, cfg)
	cost := func(k int) int {
		total := 0
		for _, in := range Inputs {
			total += QuoteBuy(g, cfg, in, k, false).Total
		}
		return total
	}
	lo, hi := 0, n
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if cost(mid) <= budget {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// cashAfter says whether spending cost still leaves the cushion.
func cashAfter(g Game, cfg Config, cost int) bool {
	return g.Capital-cost >= cushionDays*TotalUpkeep(g, cfg)+unitCost(g, cfg)*ProductionCapacity(g, cfg)*3/10
}

// diligentEmpire is one day of empire decisions, after the day's inputs are bought.
func diligentEmpire(g *Game, cfg Config, res *simResult, peak map[string]float64) {
	// Brand upgrades of the territories it is in, when they cost under a third of its cash.
	for _, key := range []string{"roadside_sign", "billboard", "radio_spot", "tv_campaign", "global_brand", "delivery_fleet"} {
		if g.Owns(key) {
			continue
		}
		for _, u := range cfg.Upgrades {
			if u.Key == key && u.Requires.Era <= Era(*g, cfg) && g.Capital >= 3*u.Cost && cashAfter(*g, cfg, u.Cost) {
				_ = BuyUpgrade(g, cfg, key)
			}
		}
	}

	// The next territory, if its depth pays back fast enough.
	for i, d := range cfg.Territories {
		if g.Territories[d.Key].Entered {
			continue
		}
		gain := float64(d.Depth) * d.EntryShare / 100 * sampleMargin
		hub := float64(d.HubUpkeep)
		if net := gain - hub; net > 0 && float64(d.EntryCost)/net <= paybackDays && cashAfter(*g, cfg, d.EntryCost) && i > 0 {
			_ = EnterTerritory(g, cfg, d.Key)
		}
		break
	}

	// The rival cheapest per share point, if the depth it adds pays it back.
	for {
		best, bestPer := "", 0.0
		bestHostile := false
		for _, rd := range cfg.Rivals {
			r, ok := g.Rivals[rd.Key]
			if !ok || r.Status != RivalActive || r.Share <= 0 {
				continue
			}
			t := g.Territories[rd.Territory]
			hostile := rd.RefuseBelowShare > 0 && t.Share < rd.RefuseBelowShare
			price, _ := BuyoutPrice(*g, cfg, rd.Key, hostile)
			depth := ShareDepth(*g, cfg, rd.Territory, r.Share)
			if float64(price)/(depth*sampleMargin) > buyoutPayback || !cashAfter(*g, cfg, price) {
				continue
			}
			if per := float64(price) / r.Share; best == "" || per < bestPer {
				best, bestPer, bestHostile = rd.Key, per, hostile
			}
		}
		if best == "" {
			break
		}
		if err := BuyOut(g, cfg, best, bestHostile); err != nil {
			break
		}
		res.buyouts++
	}

	// Campaigns only to defend: a share below its peak, or a rival move announced.
	for _, d := range cfg.Territories {
		t := g.Territories[d.Key]
		if !t.Entered {
			continue
		}
		if t.Share > peak[d.Key] {
			peak[d.Key] = t.Share
		}
		threatened := t.Share < peak[d.Key]-0.5
		for _, rd := range cfg.Rivals {
			if r, ok := g.Rivals[rd.Key]; ok && rd.Territory == d.Key && r.TelegraphKey != "" && r.Status == RivalActive {
				threatened = true
			}
		}
		if threatened && t.CampaignDaysLeft == 0 {
			if cost := CampaignCost(cfg, d.Key, 1); cashAfter(*g, cfg, 4*cost) {
				_ = RunCampaign(g, cfg, d.Key, 1)
			}
		}
	}
}

// noteEmpire records era arrival, net worth and the flat-run length.
func noteEmpire(res *simResult, g Game, cfg Config) {
	nw := NetWorth(g, cfg)
	if era := Era(g, cfg); res.eraDay[era] == 0 {
		res.eraDay[era], res.eraNW[era] = g.Day, nw
	}
	res.nw = append(res.nw, nw)
}

// longestFlat is the longest run of days, from a start day on, in which net worth did not
// rise above its best so far.
func longestFlat(nw []int, from int) int {
	best, run, peak := 0, 0, 0
	for i, v := range nw {
		if i+1 < from {
			peak = max(peak, v)
			continue
		}
		if v > peak {
			peak, run = v, 0
			continue
		}
		run++
		best = max(best, run)
	}
	return best
}

// diligent plays the careful grower's business with the empire on top.
func diligent(cfg Config, seed int64, days int) simResult {
	g := NewGame(cfg, seed)
	res := simResult{}
	peak := map[string]float64{}
	for d := 1; d <= days; d++ {
		res.capital = append(res.capital, g.Capital)
		noteEmpire(&res, g, cfg)
		sellAll(&g, cfg, Lemonade)

		n := ProductionCapacity(g, cfg)
		for _, in := range Inputs {
			if free := batchRoom(g, cfg, in); free < n {
				n = free
			}
		}
		if free := batchRoom(g, cfg, Lemonade); free < n {
			n = free
		}
		if uc := unitCost(g, cfg); uc > 0 && g.Capital/uc < n {
			n = g.Capital / uc
		}
		capacityBinds := true
		if best := profitableBatch(g, cfg, n, 0); best < n {
			n, capacityBinds = best, false
		}
		n = affordableBatch(g, cfg, n)
		for _, in := range Inputs {
			_ = Buy(&g, cfg, in, n)
		}

		diligentEmpire(&g, cfg, &res, peak)
		if capacityBinds {
			growDiligent(&g, cfg, func(x Game) int { return reserveFor(x, cfg) }, &res, d)
		}
		if !endDay(&g, cfg, &res, d) {
			break
		}
	}
	return res
}

// runSeeds plays seeds 1..seeds in parallel and returns the results in seed order.
func runSeeds(cfg Config, bot func(Config, int64, int) simResult, days, seeds int) []simResult {
	out := make([]simResult, seeds)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i := 0; i < seeds; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			out[i] = bot(cfg, int64(i+1), days)
		}(i)
	}
	wg.Wait()
	return out
}

func eraReport(name string, cfg Config, bot func(Config, int64, int) simResult, days, seeds int) {
	var eraDays [6][]int
	var eraNW [6][]int
	var bankrupt []int
	var buyouts, flat []int
	at := map[int][]int{}
	for _, r := range runSeeds(cfg, bot, days, seeds) {
		for e := 2; e <= 5; e++ {
			if r.eraDay[e] > 0 {
				eraDays[e] = append(eraDays[e], r.eraDay[e])
				eraNW[e] = append(eraNW[e], r.eraNW[e])
			}
		}
		if r.bankrupt > 0 {
			bankrupt = append(bankrupt, r.bankrupt)
		}
		buyouts = append(buyouts, r.buyouts)
		flat = append(flat, longestFlat(r.nw, 1))
		for _, d := range []int{30, 60, 90, 120, 150, 200} {
			if d <= len(r.nw) {
				at[d] = append(at[d], r.nw[d-1])
			}
		}
	}
	fmt.Printf("%-9s bankrupt %d%% | buyouts median %d | longest flat median %d days |", name, 100*len(bankrupt)/seeds, median(buyouts), median(flat))
	for _, d := range []int{30, 60, 90, 120, 150, 200} {
		if len(at[d]) > 0 {
			fmt.Printf(" nw d%d=%s", d, compactMoney(median(at[d])))
		}
	}
	fmt.Println()
	for e := 2; e <= 5; e++ {
		if len(eraDays[e]) == 0 {
			fmt.Printf("  era %d: reached by 0%%\n", e)
			continue
		}
		sort.Ints(eraDays[e])
		fmt.Printf("  era %d: reached by %d%%, median day %d (p25 %d, p75 %d), net worth then %s\n", e,
			100*len(eraDays[e])/seeds, median(eraDays[e]), eraDays[e][len(eraDays[e])/4], eraDays[e][len(eraDays[e])*3/4], compactMoney(median(eraNW[e])))
	}
}

func compactMoney(n int) string {
	switch {
	case n >= 1_000_000_000 || n <= -1_000_000_000:
		return fmt.Sprintf("$%.1fB", float64(n)/1e9)
	case n >= 1_000_000 || n <= -1_000_000:
		return fmt.Sprintf("$%.1fM", float64(n)/1e6)
	case n >= 10_000 || n <= -10_000:
		return fmt.Sprintf("$%.0fk", float64(n)/1e3)
	}
	return fmt.Sprintf("$%d", n)
}

// TestDiligentReport prints the era table for the diligent bot and the careful grower:
//
//	DILIGENT_REPORT=1 DILIGENT_SEEDS=40 go test ./internal/domain -run TestDiligentReport -v -count=1
func TestDiligentReport(t *testing.T) {
	if os.Getenv("DILIGENT_REPORT") == "" {
		t.Skip("set DILIGENT_REPORT=1 to print the diligent bot report")
	}
	seeds := 40
	if v := os.Getenv("DILIGENT_SEEDS"); v != "" {
		fmt.Sscan(v, &seeds)
	}
	cfg := DefaultConfig()
	eraReport("diligent", cfg, diligent, 200, seeds)
}
