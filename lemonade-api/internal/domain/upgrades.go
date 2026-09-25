package domain

import (
	"errors"
	"fmt"

	"lemonade-api/internal/domain/content"
)

// UpgradeDef and Effect are rows of the content tables; see content/upgrades.go.
type (
	UpgradeDef = content.UpgradeDef
	Effect     = content.EffectDef
)

var (
	ErrUnknownUpgrade = errors.New("unknown upgrade")
	ErrUpgradeOwned   = errors.New("upgrade already owned")
	ErrUpgradeLocked  = errors.New("upgrade is locked")
)

// UpgradeLockedError says why an upgrade cannot be bought yet. Code is one of era,
// warehouse_level, production_level or requires_upgrade; Need is the level, era or
// upgrade key that is missing.
type UpgradeLockedError struct {
	Code string
	Need string
}

func (e *UpgradeLockedError) Error() string {
	return fmt.Sprintf("%s: %s %s", ErrUpgradeLocked, e.Code, e.Need)
}
func (e *UpgradeLockedError) Is(target error) bool { return target == ErrUpgradeLocked }

// Upgrade looks up one upgrade by key.
func (c Config) Upgrade(key string) (UpgradeDef, bool) {
	for _, u := range c.Upgrades {
		if u.Key == key {
			return u, true
		}
	}
	return UpgradeDef{}, false
}

// Owns says whether the player has bought an upgrade.
func (g Game) Owns(key string) bool { return g.Upgrades[key] > 0 }

// UpgradeLock returns nil when the upgrade's requirements are met, else why not.
func UpgradeLock(g Game, cfg Config, u UpgradeDef) *UpgradeLockedError {
	r := u.Requires
	switch {
	case r.Era > Era(g, cfg):
		return &UpgradeLockedError{Code: "era", Need: fmt.Sprint(r.Era)}
	case r.WarehouseLevel > g.WarehouseLevel:
		return &UpgradeLockedError{Code: "warehouse_level", Need: fmt.Sprint(r.WarehouseLevel)}
	case r.ProductionLevel > g.ProductionLevel:
		return &UpgradeLockedError{Code: "production_level", Need: fmt.Sprint(r.ProductionLevel)}
	}
	for _, need := range r.Upgrades {
		if !g.Owns(need) {
			return &UpgradeLockedError{Code: "requires_upgrade", Need: need}
		}
	}
	return nil
}

// BuyUpgrade buys one upgrade. Upgrades are permanent: they cannot be sold, and they are
// not part of net worth (DECISIONS: upgrades are not counted).
func BuyUpgrade(g *Game, cfg Config, key string) error {
	if g.Status != StatusActive {
		return ErrGameOver
	}
	u, ok := cfg.Upgrade(key)
	if !ok {
		return ErrUnknownUpgrade
	}
	if g.Owns(key) {
		return ErrUpgradeOwned
	}
	if lock := UpgradeLock(*g, cfg, u); lock != nil {
		return lock
	}
	if u.Cost > g.Capital {
		return ErrInsufficientFunds
	}
	g.Capital -= u.Cost
	if g.Upgrades == nil {
		g.Upgrades = make(map[string]int)
	}
	g.Upgrades[key] = 1
	g.UpgradeSpend += u.Cost
	g.notePeak()
	return nil
}

// UpgradeUpkeep is the daily upkeep of every upgrade owned.
func UpgradeUpkeep(g Game, cfg Config) int {
	total := 0
	for _, u := range cfg.Upgrades {
		if g.Owns(u.Key) {
			total += u.Upkeep
		}
	}
	return total
}
