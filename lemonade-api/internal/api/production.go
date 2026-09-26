package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"lemonade-api/internal/domain"
)

// learnRecipe pays for a recipe (an era-gated purchase) and answers with the new game view.
func (h *Game) learnRecipe(c *gin.Context) {
	key := c.Param("key")
	h.mutate(c, func(g *domain.Game) error { return domain.LearnRecipe(g, h.cfg, key) })
}

// setPlan replaces the production plan. An empty list restores the default.
func (h *Game) setPlan(c *gin.Context) {
	var req struct {
		Rows []planRowDTO `json:"rows"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "invalid_request", "Request body must be JSON like {\"rows\": [{\"recipe\": \"lemonade\", \"target\": 0}]}.")
		return
	}
	rows := make([]domain.PlanRow, 0, len(req.Rows))
	for _, r := range req.Rows {
		rows = append(rows, domain.PlanRow(r))
	}
	h.mutate(c, func(g *domain.Game) error { return domain.SetProductionPlan(g, h.cfg, rows) })
}
