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
