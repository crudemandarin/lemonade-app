package domain

// Perishables (late game Products B). Fresh goods have a shelf life in days. Their stock
// is tracked by age (Game.Aged: index 0 is bought today); production and sales use the
// oldest first, and each night what has reached the end of its shelf life spoils. The
// cost basis falls with it. This is not an event and not random, so it does not break
// the rule that no random event destroys inventory. Lemons keep (grandfathered), ice
// melts nightly (step 4, as before), and finished goods never spoil.

// shelfDays is how many days stock of r can be held: its shelf life plus what the storage
// upgrades add (a cold room). 0 means it does not spoil.
func shelfDays(g Game, cfg Config, r Resource) int {
	c, ok := cfg.Commodity(r)
	if !ok || c.ShelfLifeDays <= 0 {
		return 0
	}
	return c.ShelfLifeDays + ExtraShelfLife(g, cfg, c.StorageClass)
}

// addAged records freshly bought cases of a perishable as age 0.
func (g *Game) addAged(cfg Config, r Resource, n int) {
	if n <= 0 || shelfDays(*g, cfg, r) == 0 {
		return
	}
	if g.Aged == nil {
		g.Aged = map[Resource][]int{}
	}
	ages := g.Aged[r]
	if len(ages) == 0 {
		ages = []int{0}
	}
	ages[0] += n
	g.Aged[r] = ages
}

// takeOldest removes n cases from a perishable's age buckets, oldest first. Stock that
// no bucket accounts for (it was put in some other way) counts as fresh and goes last.
func (g *Game) takeOldest(r Resource, n int) {
	ages := g.Aged[r]
	if len(ages) == 0 || n <= 0 {
		return
	}
	for i := len(ages) - 1; i >= 0 && n > 0; i-- {
		take := min(ages[i], n)
		ages[i] -= take
		n -= take
	}
	g.Aged[r] = ages
}

// balanceAged makes the buckets add up to the stock: extra stock is fresh, and buckets
// never hold more than the inventory (the oldest give way).
func (g *Game) balanceAged(r Resource) {
	if g.Aged == nil {
		g.Aged = map[Resource][]int{}
	}
	ages := g.Aged[r]
	sum := 0
	for _, a := range ages {
		sum += a
	}
	switch held := g.Inventory[r]; {
	case held > sum:
		if len(ages) == 0 {
			ages = []int{0}
		}
		ages[0] += held - sum
	case held < sum:
		g.Aged[r] = ages
		g.takeOldest(r, sum-held)
		return
	}
	g.Aged[r] = ages
}

// spoilPerishables is the spoilage half of step 4: the cases that reach the end of their
// shelf life tonight go off, then the rest age a day.
func spoilPerishables(g *Game, cfg Config, report *DayReport) {
	for _, c := range cfg.Commodities {
		days := shelfDays(*g, cfg, c.Key)
		if days == 0 {
			continue
		}
		if g.Inventory[c.Key] == 0 && len(g.Aged[c.Key]) == 0 {
			continue
		}
		g.balanceAged(c.Key)
		ages := g.Aged[c.Key]
		gone := 0
		for i := days - 1; i < len(ages); i++ {
			gone += ages[i]
		}
		if gone > 0 {
			g.removeStock(c.Key, gone) // oldest first, and the basis falls with it
			if report.Spoiled == nil {
				report.Spoiled = map[Resource]int{}
			}
			report.Spoiled[c.Key] = gone
		}
		ages = g.Aged[c.Key]
		next := make([]int, 0, days)
		next = append(next, 0)
		for i := 0; i < len(ages) && i < days-1; i++ {
			next = append(next, ages[i])
		}
		g.Aged[c.Key] = next
	}
}

// SeedMarkets gives a game a market for every commodity in the catalog: a game saved
// before a commodity existed has none for it yet, so it starts at the base price.
func SeedMarkets(g *Game, cfg Config) {
	if g.Market == nil {
		g.Market = map[Resource]*ResourceMarket{}
	}
	for _, r := range cfg.Resources() {
		if g.Market[r] == nil {
			base := cfg.BasePrice[r]
			g.Market[r] = &ResourceMarket{Price: float64(base), History: []int{base}}
		}
	}
}
