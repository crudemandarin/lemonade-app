package api

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"

	"lemonade-api/internal/domain"
)

// EmpireResponse (GET /api/game/empire) is the heavy empire data: every territory with its
// share, rivals, telegraphs and costs. The game view only carries the era, the reach per
// product and the next goal.
type empireDTO struct {
	Era          int    `json:"era"`
	EraName      string `json:"eraName"`
	HubUpkeep    int    `json:"hubUpkeep"`
	Acquisitions int    `json:"acquisitions"`
	// Reach is the lemonade depth the player holds now, summed over the territories.
	Reach       int            `json:"reach"`
	NextGoal    *goalDTO       `json:"nextGoal"`
	Territories []territoryDTO `json:"territories"`
}

type goalDTO struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
	Name string `json:"name"`
	Cost int    `json:"cost"`
}

type campaignOptionDTO struct {
	Level int     `json:"level"`
	Name  string  `json:"name"`
	Bonus float64 `json:"bonus"`
	Days  int     `json:"days"`
	Cost  int     `json:"cost"`
}

type territoryDTO struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Era         int    `json:"era"`
	Entered     bool   `json:"entered"`
	Depth       int    `json:"depth"`
	EntryCost   int    `json:"entryCost"`
	EntryShare  int    `json:"entryShare"`
	HubUpkeep   int    `json:"hubUpkeep"`
	BuildingCap int    `json:"buildingCap"`
	// Share is the player's share in percent, to one decimal.
	Share float64 `json:"share"`
	// Reach is the lemonade depth this territory gives now.
	Reach int `json:"reach"`
	// EnterBlocked is empty when the territory can be entered now, else territory_locked
	// (the one before is not entered), already_entered or insufficient_funds.
	EnterBlocked     string              `json:"enterBlocked"`
	CampaignDaysLeft int                 `json:"campaignDaysLeft"`
	CampaignBonus    float64             `json:"campaignBonus"`
	Campaigns        []campaignOptionDTO `json:"campaigns"`
	Rivals           []rivalDTO          `json:"rivals"`
}

type telegraphDTO struct {
	Key  string `json:"key"`
	Day  int    `json:"day"`
	Text string `json:"text"`
}

type rivalDTO struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Personality string  `json:"personality"`
	Flavor      string  `json:"flavor"`
	Status      string  `json:"status"`
	Mood        string  `json:"mood"`
	Share       float64 `json:"share"`
	Valuation   int     `json:"valuation"`
	// Trend is rising, falling or steady: which way its valuation is heading.
	Trend string `json:"trend"`
	// BuyoutPrice is the friendly price (a merger offer discounts it); HostilePrice ignores refusals.
	BuyoutPrice   int  `json:"buyoutPrice"`
	HostilePrice  int  `json:"hostilePrice"`
	Offer         bool `json:"offer"`
	OfferDaysLeft int  `json:"offerDaysLeft"`
	// Refuses is true when a friendly buyout would be refused now; NeedShare is the share of
	// the territory, in percent, that would change its mind.
	Refuses   bool          `json:"refuses"`
	NeedShare float64       `json:"needShare"`
	Telegraph *telegraphDTO `json:"telegraph"`
	CanAfford bool          `json:"canAfford"`
	PricePaid int           `json:"pricePaid"`
	Hostile   bool          `json:"hostile"`
}

func round1(x float64) float64 { return math.Round(x*10) / 10 }

var telegraphText = map[string]string{
	domain.EventPriceWar:      "is slashing prices",
	domain.EventRivalCampaign: "is planning a campaign",
}

func toGoal(g domain.Game, cfg domain.Config) *goalDTO {
	goal := domain.NextGoal(g, cfg)
	if goal.Kind == "" {
		return nil
	}
	return &goalDTO{Kind: goal.Kind, Key: goal.Key, Name: goal.Name, Cost: goal.Cost}
}

func toEmpire(g domain.Game, cfg domain.Config) empireDTO {
	domain.SeedEmpire(&g, cfg)
	out := empireDTO{
		Era: domain.Era(g, cfg), EraName: domain.EraName(g, cfg), HubUpkeep: domain.HubUpkeep(g, cfg),
		Acquisitions: domain.AcquisitionValue(g, cfg), Reach: domain.Reach(g, cfg, domain.Lemonade),
		NextGoal: toGoal(g, cfg), Territories: []territoryDTO{},
	}
	visible := domain.TelegraphVisibleDays(g, cfg)
	for i, d := range cfg.Territories {
		t := g.Territories[d.Key]
		dto := territoryDTO{
			Key: d.Key, Name: d.Name, Era: d.Era, Entered: t.Entered, Depth: d.Depth, EntryCost: d.EntryCost,
			EntryShare: int(d.EntryShare), HubUpkeep: d.HubUpkeep, BuildingCap: d.BuildingCap,
			Share: round1(t.Share), Reach: domain.ReachIn(g, cfg, d.Key, domain.Lemonade),
			CampaignDaysLeft: t.CampaignDaysLeft, CampaignBonus: t.CampaignBonus,
			Campaigns: []campaignOptionDTO{}, Rivals: []rivalDTO{},
		}
		switch {
		case t.Entered:
			dto.EnterBlocked = "already_entered"
		case i > 0 && !g.Territories[cfg.Territories[i-1].Key].Entered:
			dto.EnterBlocked = "territory_locked"
		case d.EntryCost > g.Capital:
			dto.EnterBlocked = "insufficient_funds"
		}
		if t.Entered {
			for l, c := range cfg.Campaigns {
				dto.Campaigns = append(dto.Campaigns, campaignOptionDTO{Level: l + 1, Name: c.Name, Bonus: c.Bonus, Days: c.Days, Cost: domain.CampaignCost(cfg, d.Key, l+1)})
			}
		}
		for _, rd := range cfg.Rivals {
			r, ok := g.Rivals[rd.Key]
			if !ok || rd.Territory != d.Key {
				continue
			}
			price, offer := domain.BuyoutPrice(g, cfg, rd.Key, false)
			hostile, _ := domain.BuyoutPrice(g, cfg, rd.Key, true)
			rv := rivalDTO{
				Key: rd.Key, Name: rd.Name, Personality: rd.Personality, Flavor: rd.Flavor, Status: r.Status, Mood: r.Mood,
				Share: round1(r.Share), Valuation: int(math.Round(r.Valuation)), Trend: "steady",
				BuyoutPrice: price, HostilePrice: hostile, Offer: offer, OfferDaysLeft: r.OfferDaysLeft,
				Refuses:   rd.RefuseBelowShare > 0 && t.Share < rd.RefuseBelowShare,
				NeedShare: rd.RefuseBelowShare, CanAfford: price <= g.Capital, PricePaid: r.PricePaid, Hostile: r.Hostile,
			}
			switch {
			case r.Momentum > 0.001:
				rv.Trend = "rising"
			case r.Momentum < -0.001:
				rv.Trend = "falling"
			}
			if r.TelegraphKey != "" && r.TelegraphDay-g.Day < visible+1 && r.TelegraphDay >= g.Day {
				rv.Telegraph = &telegraphDTO{Key: r.TelegraphKey, Day: r.TelegraphDay, Text: rd.Name + " " + telegraphText[r.TelegraphKey]}
			}
			dto.Rivals = append(dto.Rivals, rv)
		}
		out.Territories = append(out.Territories, dto)
	}
	return out
}

func (h *Game) getEmpire(c *gin.Context) {
	g, err := h.repo.GetGame(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		abortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, toEmpire(g, h.cfg))
}

func (h *Game) enterTerritory(c *gin.Context) {
	key := c.Param("key")
	h.mutate(c, func(g *domain.Game) error { return domain.EnterTerritory(g, h.cfg, key) })
}

func (h *Game) buyOut(c *gin.Context) {
	var req struct {
		Hostile bool `json:"hostile"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			abort(c, http.StatusBadRequest, "invalid_request", "Request body must be JSON like {\"hostile\": false}.")
			return
		}
	}
	key := c.Param("key")
	h.mutate(c, func(g *domain.Game) error { return domain.BuyOut(g, h.cfg, key, req.Hostile) })
}

func (h *Game) campaign(c *gin.Context) {
	var req struct {
		Level int `json:"level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "invalid_request", "Request body must be JSON with a campaign level.")
		return
	}
	key := c.Param("key")
	h.mutate(c, func(g *domain.Game) error { return domain.RunCampaign(g, h.cfg, key, req.Level) })
}
