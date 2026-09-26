package domain

import (
	"errors"
	"math/rand"
	"reflect"
	"testing"
)

// stocked gives a game plenty of cash, big warehouses and production for the tests below.
func stocked(t *testing.T) (Game, Config) {
	t.Helper()
	cfg := DefaultConfig()
	g := NewGame(cfg, 42)
	g.Capital = 1_000_000
	for _, class := range StorageClasses(cfg) {
		g.WarehouseQty[class] = 10
	}
	g.ProductionQty = 10 // 100 a night at level 1
	return g, cfg
}

func TestLearningARecipeCostsOnceAndUnlocksItsGoods(t *testing.T) {
	g, cfg := stocked(t)
	if CommodityUnlocked(g, cfg, "lime") || CommodityUnlocked(g, cfg, "limeade") {
		t.Fatal("limes and limeade are hidden until limeade is learned")
	}
	if err := Buy(&g, cfg, "lime", 1); !errors.Is(err, ErrCommodityLocked) {
		t.Fatalf("buying a locked commodity: %v", err)
	}
	if err := LearnRecipe(&g, cfg, "limeade"); err != nil {
		t.Fatalf("limeade is an era 1 recipe: %v", err)
	}
	if g.Capital != 1_000_000-800 {
		t.Fatalf("capital %d, want the $800 learn cost taken", g.Capital)
	}
	if !CommodityUnlocked(g, cfg, "lime") || !CommodityUnlocked(g, cfg, "limeade") {
		t.Fatal("learning limeade unlocks limes and limeade")
	}
	if err := LearnRecipe(&g, cfg, "limeade"); !errors.Is(err, ErrRecipeKnown) {
		t.Fatalf("learning twice: %v", err)
	}
	if err := Buy(&g, cfg, "lime", 1); err != nil {
		t.Fatalf("buying limes: %v", err)
	}
}

func TestRecipeLocks(t *testing.T) {
	g, cfg := stocked(t)
	var locked *RecipeLockedError
	if err := LearnRecipe(&g, cfg, "mint_lemonade"); !errors.As(err, &locked) || locked.Code != LockEra {
		t.Fatalf("an era 2 recipe in era 1: %v", err)
	}
	// Lemon bars need the oven as well as the era.
	g.Territories["city"] = TerritoryState{Entered: true, Share: 10}
	if err := LearnRecipe(&g, cfg, "lemon_bars"); !errors.As(err, &locked) || locked.Code != LockUnlock {
		t.Fatalf("lemon bars without an oven: %v", err)
	}
	g.Upgrades["oven"] = 1
	if err := LearnRecipe(&g, cfg, "lemon_bars"); err != nil {
		t.Fatalf("lemon bars with an oven: %v", err)
	}
	if err := LearnRecipe(&g, cfg, "nonsense"); !errors.Is(err, ErrUnknownRecipe) {
		t.Fatalf("unknown recipe: %v", err)
	}
	g.Capital = 10
	if err := LearnRecipe(&g, cfg, "mint_lemonade"); !errors.Is(err, ErrInsufficientFunds) {
		t.Fatalf("too poor: %v", err)
	}
}

// learnAll teaches every recipe that has no lock beyond era, for plan tests.
func learnAll(g *Game, cfg Config) {
	g.Territories["city"] = TerritoryState{Entered: true, Share: 10}
	g.Upgrades["oven"], g.Upgrades["zester"] = 1, 1
	for _, r := range cfg.Recipes {
		if r.Era > 0 {
			g.Recipes[r.Key] = true
		}
	}
}

func TestAPlanSharesCapacityInOrderAndHonorsTargets(t *testing.T) {
	g, cfg := stocked(t)
	learnAll(&g, cfg)
	for _, in := range []Resource{"lime", "lemon", "sugar", "ice", "cup"} {
		g.Inventory[in] = 60
	}
	if err := SetProductionPlan(&g, cfg, []PlanRow{{Recipe: "limeade", Target: 30}, {Recipe: "lemonade"}}); err != nil {
		t.Fatal(err)
	}
	var report DayReport
	res := produce(&g, cfg)
	report.Produced = producedTotal(res)
	// Limeade stops at its target of 30, and lemonade takes the rest of the capacity but
	// only 30 of the sugar, ice and cups remain (60 minus the limeade's 30).
	if res[0].Output != 30 || res[0].LimitedBy != LimitTarget {
		t.Fatalf("limeade: %+v", res[0])
	}
	if res[1].Output != 30 {
		t.Fatalf("lemonade should get what the limeade left: %+v", res[1])
	}
	if g.Inventory["limeade"] != 30 || g.Inventory[Lemonade] != 30 || g.Inventory[Sugar] != 0 {
		t.Fatalf("inventory: %v", g.Inventory)
	}
}

func TestEmptyPlanIsTheBaseGame(t *testing.T) {
	g, cfg := stocked(t)
	for _, in := range Inputs {
		g.Inventory[in] = 50
	}
	if p := EffectivePlan(g, cfg); len(p) != 1 || p[0].Recipe != "lemonade" || p[0].Target != 0 {
		t.Fatalf("default plan: %+v", p)
	}
	if err := SetProductionPlan(&g, cfg, []PlanRow{{Recipe: "limeade"}}); !errors.Is(err, ErrPlanRecipe) {
		t.Fatalf("an unlearned recipe in the plan: %v", err)
	}
	learnAll(&g, cfg)
	for _, bad := range [][]PlanRow{
		{{Recipe: "lemonade"}, {Recipe: "lemonade"}},
		{{Recipe: "lemonade", Target: -1}},
	} {
		if err := SetProductionPlan(&g, cfg, bad); err == nil {
			t.Fatalf("plan %+v should be rejected", bad)
		}
	}
}

func TestPreviewMatchesEndDayWithManyRecipes(t *testing.T) {
	cfg := DefaultConfig()
	rng := rand.New(rand.NewSource(3))
	for i := 0; i < 60; i++ {
		g, _ := stocked(t)
		learnAll(&g, cfg)
		g.ProductionQty = 1 + rng.Intn(10)
		for _, r := range cfg.Resources() {
			if c, _ := cfg.Commodity(r); c.Input {
				g.Inventory[r] = rng.Intn(40)
			}
		}
		g.CostBasis = map[Resource]int{}
		plan := []PlanRow{}
		for _, rec := range cfg.Recipes {
			if rng.Intn(2) == 0 {
				plan = append(plan, PlanRow{Recipe: rec.Key, Target: rng.Intn(3) * 15})
			}
		}
		rng.Shuffle(len(plan), func(a, b int) { plan[a], plan[b] = plan[b], plan[a] })
		if err := SetProductionPlan(&g, cfg, plan); err != nil {
			t.Fatal(err)
		}
		p := PreviewEndDay(g, cfg)
		before := g.Clone()
		report, err := EndDay(&g, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if p.LemonadeToProduce != g.Inventory[Lemonade]-before.Inventory[Lemonade] {
			t.Fatalf("case %d: preview says %d lemonade, made %d", i, p.LemonadeToProduce, g.Inventory[Lemonade]-before.Inventory[Lemonade])
		}
		total := 0
		for _, row := range p.Plan {
			total += row.Cases
		}
		if total != report.Produced {
			t.Fatalf("case %d: preview plan makes %d, EndDay made %d (plan %+v)", i, total, report.Produced, plan)
		}
		if !reflect.DeepEqual(p.WillSpoil, report.Spoiled) {
			t.Fatalf("case %d: preview spoils %v, EndDay %v", i, p.WillSpoil, report.Spoiled)
		}
	}
}

func TestPerishablesSpoilAfterTheirShelfLifeOldestFirst(t *testing.T) {
	g, cfg := stocked(t)
	learnAll(&g, cfg)
	g.Capital = 1_000_000
	if err := Buy(&g, cfg, "strawberry", 10); err != nil { // shelf life 3 days
		t.Fatal(err)
	}
	basis := g.CostBasis["strawberry"]
	if basis == 0 {
		t.Fatal("buying should record a cost basis")
	}
	g.ProductionQty = 1
	for day := 1; day <= 3; day++ {
		report, err := EndDay(&g, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if day < 3 && (g.Inventory["strawberry"] != 10 || report.Spoiled["strawberry"] != 0) {
			t.Fatalf("day %d: strawberries should still be good: %d, spoiled %v", day, g.Inventory["strawberry"], report.Spoiled)
		}
		if day == 3 && (g.Inventory["strawberry"] != 0 || report.Spoiled["strawberry"] != 10) {
			t.Fatalf("day 3: all 10 should spoil on the third night: %d, spoiled %v", g.Inventory["strawberry"], report.Spoiled)
		}
	}
	if g.CostBasis["strawberry"] != 0 {
		t.Fatalf("the basis falls with spoiled stock, got %d", g.CostBasis["strawberry"])
	}

	// Lemons keep.
	g2, _ := stocked(t)
	_ = Buy(&g2, cfg, Lemon, 10)
	for i := 0; i < 10; i++ {
		_, _ = EndDay(&g2, cfg)
	}
	if g2.Inventory[Lemon] != 10 {
		t.Fatalf("lemons keep, but %d are left", g2.Inventory[Lemon])
	}
}

func TestProductionUsesTheOldestPerishablesFirst(t *testing.T) {
	g, cfg := stocked(t)
	learnAll(&g, cfg)
	g.Recipes["strawberry_lemonade"] = true
	_ = Buy(&g, cfg, "strawberry", 5) // day 1
	_, _ = EndDay(&g, cfg)            // they age a night; nothing is planned to use them
	_ = Buy(&g, cfg, "strawberry", 5) // day 2: 5 old, 5 fresh
	for _, in := range []Resource{Lemon, Sugar, Ice, Cup} {
		_ = Buy(&g, cfg, in, 5)
	}
	if err := SetProductionPlan(&g, cfg, []PlanRow{{Recipe: "strawberry_lemonade"}}); err != nil {
		t.Fatal(err)
	}
	report, _ := EndDay(&g, cfg) // makes 5, using the 5 oldest strawberries
	if report.Produced != 5 {
		t.Fatalf("made %d, want 5", report.Produced)
	}
	if ages := g.Aged["strawberry"]; len(ages) < 2 || ages[1] != 5 {
		// After the night the 5 fresh ones aged into index 1; nothing is left at index 2.
		t.Fatalf("ages after using the oldest: %v", ages)
	}
}

func TestAColdRoomExtendsShelfLife(t *testing.T) {
	g, cfg := stocked(t)
	learnAll(&g, cfg)
	base := shelfDays(g, cfg, "strawberry")
	g.Upgrades["cold_room"] = 1
	if got := shelfDays(g, cfg, "strawberry"); got != base+2 {
		t.Fatalf("a cold room adds 2 days: %d to %d", base, got)
	}
	if shelfDays(g, cfg, Lemon) != 0 || shelfDays(g, cfg, "flour") != 0 {
		t.Fatal("keeping goods have no shelf life to extend")
	}
}

func TestClassesPoolCapacity(t *testing.T) {
	g, cfg := stocked(t)
	learnAll(&g, cfg)
	g.WarehouseQty[StorageCold] = 1 // 10 cases shared by lemons, strawberries, butter...
	g.Inventory[Lemon] = 6
	if got := FreeSpace(g, cfg, "strawberry"); got != 4 {
		t.Fatalf("free space in the cold room %d, want 4 (lemons share it)", got)
	}
	if err := Buy(&g, cfg, "strawberry", 5); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("5 strawberries into 4 free spaces: %v", err)
	}
	if err := Buy(&g, cfg, "strawberry", 4); err != nil {
		t.Fatalf("4 strawberries fit: %v", err)
	}
}

func TestAZesterSavesPeelAndCandiedPeelUsesIt(t *testing.T) {
	g, cfg := stocked(t)
	learnAll(&g, cfg)
	for _, in := range Inputs {
		g.Inventory[in] = 20
	}
	if err := SetProductionPlan(&g, cfg, []PlanRow{{Recipe: "lemonade"}, {Recipe: "candied_peel"}}); err != nil {
		t.Fatal(err)
	}
	g.Inventory[Sugar] = 40
	res := produce(&g, cfg)
	if g.Inventory[Lemon] != 0 || g.Inventory[LemonPeel] != 20-res[1].Output {
		t.Fatalf("20 lemons should leave 20 peel, of which the candied peel used %d: inventory %v", res[1].Output, g.Inventory)
	}
	// The peel of tonight's lemons is only there after the row ran, so tonight's candied
	// peel used none, and it is made from tomorrow.
	if res[1].Output != 0 {
		t.Fatalf("no peel existed when the candied peel row ran: %+v", res[1])
	}
	res = produce(&g, cfg)
	if res[1].Output == 0 {
		t.Fatalf("with peel and sugar in stock the candied peel row makes some: %+v", res)
	}
}
