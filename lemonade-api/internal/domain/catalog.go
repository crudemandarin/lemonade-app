package domain

import (
	"sort"

	"lemonade-api/internal/domain/content"
)

// Commodity is one tradable good, read from the content catalog into Config.
type Commodity struct {
	Key          Resource
	Name         string
	Category     string
	StorageClass string
	// ShelfLifeDays: 0 keeps, -1 melts nightly, otherwise days before it spoils.
	ShelfLifeDays int
	Input         bool
	Product       bool
	Order         int
	// NotBought marks a byproduct the market does not sell.
	NotBought bool
}

// Ingredient is Qty cases of a commodity used by one batch of a recipe.
type Ingredient struct {
	Resource Resource
	Qty      int
}

// Recipe turns its inputs, taken in order, into OutputQty cases of Output per batch.
type Recipe struct {
	Key       string
	Name      string
	Output    Resource
	OutputQty int
	Inputs    []Ingredient
	// Era, LearnCost, Unlock and MinProductionLevel say when the recipe can be learned (see
	// content.RecipeDef); Text describes it. Era 0 means known from the start.
	Era                int
	LearnCost          int
	Unlock             string
	MinProductionLevel int
	Text               string
}

// catalog converts the content tables into domain types, sorted by display order.
// Base prices are returned separately because they are a tuning knob on Config.
func catalog() ([]Commodity, []Recipe, map[Resource]int) {
	defs := append([]content.CommodityDef{}, content.Commodities...)
	sort.SliceStable(defs, func(i, j int) bool { return defs[i].Order < defs[j].Order })
	commodities := make([]Commodity, 0, len(defs))
	base := make(map[Resource]int, len(defs))
	for _, d := range defs {
		commodities = append(commodities, Commodity{
			Key: Resource(d.Key), Name: d.Name, Category: d.Category, StorageClass: d.StorageClass,
			ShelfLifeDays: d.ShelfLifeDays, Input: d.Input, Product: d.Product, Order: d.Order, NotBought: d.NotBought,
		})
		base[Resource(d.Key)] = d.BasePrice
	}
	recipes := make([]Recipe, 0, len(content.Recipes))
	for _, r := range content.Recipes {
		inputs := make([]Ingredient, 0, len(r.Inputs))
		for _, in := range r.Inputs {
			inputs = append(inputs, Ingredient{Resource: Resource(in.Key), Qty: in.Qty})
		}
		recipes = append(recipes, Recipe{Key: r.Key, Name: r.Name, Output: Resource(r.Output), OutputQty: r.OutputQty, Inputs: inputs,
			Era: r.Era, LearnCost: r.LearnCost, Unlock: r.Unlock, MinProductionLevel: r.MinProductionLevel, Text: r.Text})
	}
	return commodities, recipes, base
}

// Resources lists every commodity key in display order. Anything that draws random
// numbers per commodity must iterate this, never a map, so a seed replays exactly.
func (c Config) Resources() []Resource {
	out := make([]Resource, 0, len(c.Commodities))
	for _, m := range c.Commodities {
		out = append(out, m.Key)
	}
	return out
}

// Commodity looks up one commodity by key.
func (c Config) Commodity(r Resource) (Commodity, bool) {
	for _, m := range c.Commodities {
		if m.Key == r {
			return m, true
		}
	}
	return Commodity{}, false
}

// Valid reports whether r is a commodity in the catalog.
func (c Config) Valid(r Resource) bool {
	_, ok := c.Commodity(r)
	return ok
}

// MainRecipe is the recipe production runs. Until production plans exist
// (late game Products B) it is the first recipe: lemonade.
func (c Config) MainRecipe() Recipe {
	return c.Recipes[0]
}

// Inputs are the commodities the main recipe consumes, in recipe order.
func (c Config) Inputs() []Resource {
	r := c.MainRecipe()
	out := make([]Resource, 0, len(r.Inputs))
	for _, in := range r.Inputs {
		out = append(out, in.Resource)
	}
	return out
}

// baseFreeDepth is lemonade's free depth at warehouse level 1, in cases; every other
// commodity's is that times its DepthPct (the original five are all 80).
const baseFreeDepth = 80

func freeDepths() map[Resource]int {
	out := make(map[Resource]int, len(content.Commodities))
	for _, d := range content.Commodities {
		pct := d.DepthPct
		if pct == 0 {
			pct = 100
		}
		out[Resource(d.Key)] = max(1, baseFreeDepth*pct/100)
	}
	return out
}
