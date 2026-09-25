package content

import "testing"

func TestCommoditiesAreValid(t *testing.T) {
	seen := map[string]bool{}
	orders := map[int]bool{}
	classes := map[string]bool{StorageDry: true, StorageCold: true, StorageFrozen: true, StorageFinished: true}
	for _, c := range Commodities {
		if c.Key == "" || c.Name == "" {
			t.Errorf("commodity %+v needs a key and a name", c)
		}
		if seen[c.Key] {
			t.Errorf("duplicate commodity key %q", c.Key)
		}
		seen[c.Key] = true
		if orders[c.Order] {
			t.Errorf("duplicate display order %d (%s)", c.Order, c.Key)
		}
		orders[c.Order] = true
		if !classes[c.StorageClass] {
			t.Errorf("%s: unknown storage class %q", c.Key, c.StorageClass)
		}
		if c.BasePrice < 1 || c.BasePrice > 100000 {
			t.Errorf("%s: base price %d out of range", c.Key, c.BasePrice)
		}
		if c.ShelfLifeDays < MeltsNightly || c.ShelfLifeDays > 60 {
			t.Errorf("%s: shelf life %d out of range", c.Key, c.ShelfLifeDays)
		}
		if !c.Input && !c.Product {
			t.Errorf("%s: must be an input, a product, or both", c.Key)
		}
	}
	for i, k := range LegacyOrder {
		if i >= len(Commodities) || Commodities[i].Key != k {
			t.Errorf("catalog position %d must stay %q so stored arrays line up", i, k)
		}
	}
}

func TestRecipesAreValid(t *testing.T) {
	byKey := map[string]CommodityDef{}
	for _, c := range Commodities {
		byKey[c.Key] = c
	}
	seen := map[string]bool{}
	for _, r := range Recipes {
		if seen[r.Key] {
			t.Errorf("duplicate recipe key %q", r.Key)
		}
		seen[r.Key] = true
		if out, ok := byKey[r.Output]; !ok || !out.Product {
			t.Errorf("%s: output %q must be a product in the catalog", r.Key, r.Output)
		}
		if r.OutputQty < 1 || len(r.Inputs) == 0 {
			t.Errorf("%s: needs an output quantity and inputs", r.Key)
		}
		for _, in := range r.Inputs {
			if c, ok := byKey[in.Key]; !ok || !c.Input {
				t.Errorf("%s: input %q must be an input in the catalog", r.Key, in.Key)
			}
			if in.Qty < 1 {
				t.Errorf("%s: input %q needs a positive quantity", r.Key, in.Key)
			}
		}
	}
}

func TestUpgradesAreValid(t *testing.T) {
	classes := map[string]bool{StorageDry: true, StorageCold: true, StorageFrozen: true, StorageFinished: true}
	commodities := map[string]bool{}
	for _, c := range Commodities {
		commodities[c.Key] = true
	}
	recipes := map[string]bool{}
	for _, r := range Recipes {
		recipes[r.Key] = true
	}
	cats := map[string]bool{}
	for _, c := range UpgradeCategories {
		cats[c] = true
	}
	earlier := map[string]bool{}
	for _, u := range Upgrades {
		if u.Key == "" || u.Name == "" || u.Text == "" {
			t.Errorf("upgrade %+v needs a key, a name and a description", u)
		}
		if earlier[u.Key] {
			t.Errorf("duplicate upgrade key %q", u.Key)
		}
		if !cats[u.Category] {
			t.Errorf("%s: unknown category %q", u.Key, u.Category)
		}
		if u.Cost < 1 || u.Cost > 10_000_000 || u.Upkeep < 0 || u.Upkeep > 10_000 {
			t.Errorf("%s: cost %d or upkeep %d out of range", u.Key, u.Cost, u.Upkeep)
		}
		if u.Requires.Era < 1 || u.Requires.Era > 5 {
			t.Errorf("%s: era %d out of range", u.Key, u.Requires.Era)
		}
		for _, need := range u.Requires.Upgrades {
			if !earlier[need] {
				t.Errorf("%s: requires %q, which must be listed before it", u.Key, need)
			}
		}
		if len(u.Effects) == 0 {
			t.Errorf("%s: has no effect", u.Key)
		}
		for _, e := range u.Effects {
			checkEffect(t, u.Key, e, commodities, recipes, classes)
		}
		earlier[u.Key] = true
	}
}

func checkEffect(t *testing.T, key string, e EffectDef, commodities, recipes, classes map[string]bool) {
	t.Helper()
	bad := func(msg string) { t.Errorf("%s: effect %s: %s", key, e.Kind, msg) }
	switch e.Kind {
	case EffIceKeep, EffInputDepthPct, EffInputDiscountPct:
		if e.Value <= 0 {
			bad("needs a positive value")
		}
	case EffEventFloor:
		if e.Value <= 0 || e.Value >= 1 {
			bad("floor must be between 0 and 1")
		}
	case EffEventDamp:
		if e.Target == "" || e.Value < 0 {
			bad("needs an event and a non-negative factor")
		}
	case EffForecast:
		if (e.Target != "weather" && e.Target != "all") || e.Value < 1 {
			bad("target must be weather or all, with at least 1 day")
		}
	case EffDepthBonus, EffDepthBonusPct:
		if !commodities[e.Target] || e.Value <= 0 {
			bad("needs a catalog commodity and a positive value")
		}
	case EffUpkeepDiscountPct:
		if (e.Target != "all" && e.Target != "production" && e.Target != "warehouse") || e.Value <= 0 || e.Value >= 100 {
			bad("target must be all, production or warehouse, value under 100")
		}
	case EffYield:
		if !recipes[e.Target] || e.Value <= 0 {
			bad("needs a recipe and a positive value")
		}
	case EffUseDiscount:
		if !commodities[e.Target] || e.Value <= 0 || e.Value >= 100 {
			bad("needs a catalog commodity and a value under 100")
		}
	case EffStoragePct, EffShelfLife:
		if !classes[e.Target] || e.Value <= 0 {
			bad("needs a storage class and a positive value")
		}
	case EffMake:
		if !commodities[e.Target] || e.Value <= 0 || e.Aux < 1 {
			bad("needs a commodity, cases and a unit cost")
		}
	case EffUnlock, EffQoL:
		if e.Target == "" {
			bad("needs a target")
		}
	default:
		bad("unknown kind")
	}
}
