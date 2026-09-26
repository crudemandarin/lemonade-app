package domain

// expandStep is how the growth bots expand: one more production building and one more
// warehouse building for every commodity, in that commodity's storage class. Classes
// pool their commodities, so a class holding two of them (dry: sugar and cups) grows by
// two, and a class at its building cap simply refuses.
func expandStep(g *Game, cfg Config) {
	_ = Expand(g, cfg, Production, "")
	for _, r := range Resources {
		_ = Expand(g, cfg, Warehouse, r)
	}
}

// expandStepCost is what expandStep would spend right now, found by trying it.
func expandStepCost(g Game, cfg Config) int {
	t := g.Clone()
	t.Capital = 1 << 40
	expandStep(&t, cfg)
	return (1 << 40) - t.Capital
}

// batchRoom is how many cases of r a bot can buy in one go: the free space in r's storage
// pool, shared out between the inputs of the main recipe that live in the same class (a
// batch buys the same number of each, and sugar and cups both go into the dry store).
func batchRoom(g Game, cfg Config, r Resource) int {
	share := 0
	for _, in := range Inputs {
		if ClassOf(cfg, in) == ClassOf(cfg, r) {
			share++
		}
	}
	return FreeSpace(g, cfg, r) / max(share, 1)
}
