package api

import (
	"errors"
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
	case errors.Is(err, domain.ErrMaxLevel):
		abort(c, http.StatusConflict, "max_level", "Already at the maximum level.")
	case errors.Is(err, store.ErrNotFound):
		abort(c, http.StatusNotFound, "not_found", "No game found for this user.")
	default:
		log.Printf("internal error: %v", err)
		abort(c, http.StatusInternalServerError, "internal", "Something went wrong.")
	}
}
