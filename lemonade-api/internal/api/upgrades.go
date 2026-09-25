package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"lemonade-api/internal/domain"
	"lemonade-api/internal/domain/content"
)

// categoryNames are the headings of the Upgrades page.
var categoryNames = map[string]string{
	content.UpFreshness:  "Freshness and storage",
	content.UpProduction: "Production",
	content.UpBrand:      "Brand",
	content.UpIntel:      "Intelligence",
	content.UpResilience: "Event resilience",
	content.UpSupply:     "Supply",
	content.UpFinance:    "Finance",
	content.UpQoL:        "Convenience",
}

type upgradeCategoryDTO struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// upgradeDTO is one row of the Upgrades page. State is owned, available or locked;
// LockedReason (plain words) and LockCode (era, warehouse_level, production_level or
// requires_upgrade) are set only when locked.
type upgradeDTO struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	Era          int    `json:"era"`
	Cost         int    `json:"cost"`
	Upkeep       int    `json:"upkeep"`
	Text         string `json:"text"`
	State        string `json:"state"`
	LockCode     string `json:"lockCode"`
	LockedReason string `json:"lockedReason"`
}

// upgradesResponseDTO answers GET /api/game/upgrades.
type upgradesResponseDTO struct {
	Era          int                  `json:"era"`
	Categories   []upgradeCategoryDTO `json:"categories"`
	Upgrades     []upgradeDTO         `json:"upgrades"`
	OwnedCount   int                  `json:"ownedCount"`
	Spent        int                  `json:"spent"`
	UpkeepPerDay int                  `json:"upkeepPerDay"`
}

func lockText(l *domain.UpgradeLockedError) string {
	switch l.Code {
	case "era":
		return "Reach era " + l.Need + " first."
	case "warehouse_level":
		return "Needs warehouse level " + l.Need + "."
	case "production_level":
		return "Needs production level " + l.Need + "."
	case "requires_upgrade":
		for _, u := range content.Upgrades {
			if u.Key == l.Need {
				return "Buy " + u.Name + " first."
			}
		}
	}
	return "Locked."
}

func lockMessage(err error) string {
	var l *domain.UpgradeLockedError
	if errors.As(err, &l) {
		return lockText(l)
	}
	return "This upgrade is locked."
}

func toUpgradesResponse(g domain.Game, cfg domain.Config) upgradesResponseDTO {
	out := upgradesResponseDTO{
		Era:          domain.Era(g, cfg),
		Categories:   make([]upgradeCategoryDTO, 0, len(content.UpgradeCategories)),
		Upgrades:     make([]upgradeDTO, 0, len(cfg.Upgrades)),
		Spent:        g.UpgradeSpend,
		UpkeepPerDay: domain.UpgradeUpkeep(g, cfg),
	}
	for _, k := range content.UpgradeCategories {
		out.Categories = append(out.Categories, upgradeCategoryDTO{Key: k, Name: categoryNames[k]})
	}
	for _, u := range cfg.Upgrades {
		d := upgradeDTO{Key: u.Key, Name: u.Name, Category: u.Category, Era: u.Requires.Era, Cost: u.Cost, Upkeep: u.Upkeep, Text: u.Text, State: "available"}
		switch lock := domain.UpgradeLock(g, cfg, u); {
		case g.Owns(u.Key):
			d.State = "owned"
			out.OwnedCount++
		case lock != nil:
			d.State, d.LockCode, d.LockedReason = "locked", lock.Code, lockText(lock)
		}
		out.Upgrades = append(out.Upgrades, d)
	}
	return out
}

func (h *Game) listUpgrades(c *gin.Context) {
	g, err := h.repo.GetGame(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		abortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, toUpgradesResponse(g, h.cfg))
}

// buyUpgrade buys one upgrade and answers with the new game view (the page then reloads
// the upgrade list).
func (h *Game) buyUpgrade(c *gin.Context) {
	key := c.Param("key")
	h.mutate(c, func(g *domain.Game) error { return domain.BuyUpgrade(g, h.cfg, key) })
}
