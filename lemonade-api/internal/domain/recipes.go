package domain

import (
	"errors"
	"fmt"
)

// Recipes, learned by an era-gated purchase (late game Products B, decision 17). Lemonade
// is known from the start. Learning one costs its LearnCost once and needs its era, the
// upgrade that unlocks its kind (for example the oven for bakery goods) and, for some, a
// production tier. A recipe's goods appear in the market once it is learned.

// Recipe lock codes: why a recipe cannot be learned yet.
const (
	LockEra             = "era"
	LockUnlock          = "unlock"
	LockProductionLevel = "production_level"
)

var (
	ErrUnknownRecipe   = errors.New("unknown recipe")
	ErrRecipeKnown     = errors.New("recipe already learned")
	ErrRecipeLocked    = errors.New("recipe is locked")
	ErrCommodityLocked = errors.New("that commodity is not available yet")
)

// RecipeLockedError says why a recipe cannot be learned.
type RecipeLockedError struct {
	Code   string
	Reason string
}

func (e *RecipeLockedError) Error() string        { return e.Reason }
func (e *RecipeLockedError) Is(target error) bool { return target == ErrRecipeLocked }

// LookupRecipe finds a recipe by key.
func (c Config) LookupRecipe(key string) (Recipe, bool) {
	for _, r := range c.Recipes {
		if r.Key == key {
			return r, true
		}
	}
	return Recipe{}, false
}

// RecipeKnown reports whether the player can produce a recipe.
func RecipeKnown(g Game, rec Recipe) bool {
	return rec.Era == 0 || g.Recipes[rec.Key]
}

// KnownRecipes are the recipes the player can produce, in catalog order.
func KnownRecipes(g Game, cfg Config) []Recipe {
	var out []Recipe
	for _, r := range cfg.Recipes {
		if RecipeKnown(g, r) {
			out = append(out, r)
		}
	}
	return out
}

// RecipeLock reports why a recipe cannot be learned now: a code and words, or "" when it
// can. The order is the order a player would resolve them in.
func RecipeLock(g Game, cfg Config, rec Recipe) (code, reason string) {
	if Era(g, cfg) < rec.Era {
		return LockEra, fmt.Sprintf("Reach era %d first", rec.Era)
	}
	if rec.Unlock != "" && !HasUnlock(g, cfg, rec.Unlock) {
		return LockUnlock, unlockReason(cfg, rec.Unlock)
	}
	if g.ProductionLevel < rec.MinProductionLevel {
		return LockProductionLevel, fmt.Sprintf("Needs production level %d", rec.MinProductionLevel)
	}
	return "", ""
}

// unlockReason names the upgrade that provides a feature, in words.
func unlockReason(cfg Config, feature string) string {
	for _, u := range cfg.Upgrades {
		for _, e := range u.Effects {
			if e.Kind == "unlock" && e.Target == feature {
				return "Needs the " + u.Name
			}
		}
	}
	return "Needs an upgrade"
}

// LearnRecipe pays for a recipe and makes it producible.
func LearnRecipe(g *Game, cfg Config, key string) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	rec, ok := cfg.LookupRecipe(key)
	if !ok {
		return ErrUnknownRecipe
	}
	if RecipeKnown(*g, rec) {
		return ErrRecipeKnown
	}
	if code, reason := RecipeLock(*g, cfg, rec); code != "" {
		return &RecipeLockedError{Code: code, Reason: reason}
	}
	if rec.LearnCost > g.Capital {
		return ErrInsufficientFunds
	}
	g.Capital -= rec.LearnCost
	if g.Recipes == nil {
		g.Recipes = map[string]bool{}
	}
	g.Recipes[rec.Key] = true
	return nil
}

// CommodityUnlocked reports whether a commodity is part of the player's game: the main
// recipe's goods always are, and any other once a recipe they belong to is learned.
func CommodityUnlocked(g Game, cfg Config, r Resource) bool {
	for _, rec := range KnownRecipes(g, cfg) {
		if rec.Output == r {
			return true
		}
		for _, in := range rec.Inputs {
			if in.Resource == r {
				return true
			}
		}
	}
	return false
}

// checkTradable rejects a trade in a commodity that is not part of the player's game yet.
func checkTradable(g Game, cfg Config, r Resource) error {
	if _, ok := cfg.Commodity(r); !ok || !CommodityUnlocked(g, cfg, r) {
		return ErrCommodityLocked
	}
	return nil
}
