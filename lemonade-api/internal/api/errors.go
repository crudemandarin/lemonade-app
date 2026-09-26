package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"lemonade-api/internal/domain"
	"lemonade-api/internal/store"
)

// abort writes the stable error shape from SPEC rule 23.
func abort(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorDTO{Error: code, Message: message})
}

// abortErr maps a domain or store error to a status, code, and user-facing message.
func abortErr(c *gin.Context, err error) {
	var refuses *domain.RivalRefusesError
	if errors.As(err, &refuses) {
		abort(c, http.StatusConflict, "rival_refuses", fmt.Sprintf("This rival refuses a friendly buyout until you hold %.0f%% of its territory. A hostile takeover costs more.", refuses.NeedShare))
		return
	}
	switch {
	case errors.Is(err, domain.ErrInvalidQuantity):
		abort(c, http.StatusBadRequest, "invalid_quantity", "Quantity must be a positive whole number.")
	case errors.Is(err, domain.ErrGameOver):
		abort(c, http.StatusConflict, "game_over", "The game is over. Start a new game to keep playing.")
	case errors.Is(err, domain.ErrInsufficientFunds):
		abort(c, http.StatusConflict, "insufficient_funds", "Not enough capital.")
	case errors.Is(err, domain.ErrInsufficientStock):
		abort(c, http.StatusConflict, "insufficient_stock", "Not enough stock to sell.")
	case errors.Is(err, domain.ErrCapacityExceeded):
		abort(c, http.StatusConflict, "capacity_exceeded", "Not enough warehouse space.")
	case errors.Is(err, domain.ErrMaxQuantity):
		abort(c, http.StatusConflict, "max_quantity", "Already at the maximum number of buildings.")
	case errors.Is(err, domain.ErrInvalidFacility):
		abort(c, http.StatusBadRequest, "invalid_facility_type", "Facility type must be warehouse or production.")
	case errors.Is(err, domain.ErrMinFacility):
		abort(c, http.StatusConflict, "min_facility", "You must keep at least one building of each kind.")
	case errors.Is(err, domain.ErrStockExceedsCapacity):
		var excess *domain.StockExceedsCapacityError
		errors.As(err, &excess)
		abort(c, http.StatusConflict, "stock_exceeds_capacity", fmt.Sprintf("Sell %d %s first: the remaining warehouses cannot hold your stock.", excess.Excess, plural(excess.Excess, "case", "cases")))
	case errors.Is(err, domain.ErrUnknownUpgrade):
		abort(c, http.StatusNotFound, "unknown_upgrade", "No such upgrade.")
	case errors.Is(err, domain.ErrUpgradeOwned):
		abort(c, http.StatusConflict, "upgrade_owned", "You already own this upgrade.")
	case errors.Is(err, domain.ErrUpgradeLocked):
		abort(c, http.StatusConflict, "upgrade_locked", lockMessage(err))
	case errors.Is(err, domain.ErrMaxLevel):
		abort(c, http.StatusConflict, "max_level", "Already at the maximum level.")
	case errors.Is(err, domain.ErrTerritoryLocked), errors.Is(err, domain.ErrNotEntered):
		abort(c, http.StatusConflict, "territory_locked", "Enter the previous territory first.")
	case errors.Is(err, domain.ErrAlreadyEntered):
		abort(c, http.StatusConflict, "already_entered", "You are already in this territory.")
	case errors.Is(err, domain.ErrUnknownTerritory), errors.Is(err, domain.ErrUnknownRival):
		abort(c, http.StatusNotFound, "not_found", "No such territory or rival.")
	case errors.Is(err, domain.ErrRivalGone):
		abort(c, http.StatusConflict, "rival_gone", "That rival is no longer in business.")
	case errors.Is(err, domain.ErrInvalidCampaign):
		abort(c, http.StatusBadRequest, "invalid_campaign", "Unknown campaign level.")
	case errors.Is(err, domain.ErrCampaignRunning):
		abort(c, http.StatusConflict, "campaign_running", "A campaign is already running there.")
	case errors.Is(err, domain.ErrUnknownRecipe):
		abort(c, http.StatusNotFound, "unknown_recipe", "No such recipe.")
	case errors.Is(err, domain.ErrRecipeKnown):
		abort(c, http.StatusConflict, "recipe_known", "You already know this recipe.")
	case errors.Is(err, domain.ErrRecipeLocked):
		var locked *domain.RecipeLockedError
		msg := "That recipe is locked."
		if errors.As(err, &locked) {
			msg = locked.Reason + "."
		}
		abort(c, http.StatusConflict, "recipe_locked", msg)
	case errors.Is(err, domain.ErrCommodityLocked):
		abort(c, http.StatusConflict, "commodity_locked", "You have no use for that yet: learn a recipe that needs it first.")
	case errors.Is(err, domain.ErrPlanRecipe), errors.Is(err, domain.ErrPlanDuplicate), errors.Is(err, domain.ErrPlanTarget):
		abort(c, http.StatusBadRequest, "invalid_plan", err.Error()+".")
	case errors.Is(err, store.ErrNotFound):
		abort(c, http.StatusNotFound, "not_found", "No game found for this user.")
	default:
		log.Printf("internal error: %v", err)
		abort(c, http.StatusInternalServerError, "internal", "Something went wrong.")
	}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
