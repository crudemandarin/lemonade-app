package domain

import "errors"

// errorsNew keeps the empire errors terse.
var errorsNew = errors.New

// Sentinel errors returned by mutating domain functions. The API layer maps
// each to a stable error code and HTTP status (SPEC rule 23).
var (
	ErrInvalidQuantity   = errors.New("quantity must be positive")
	ErrGameOver          = errors.New("game is over")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrCapacityExceeded  = errors.New("warehouse capacity exceeded")
	ErrMaxQuantity       = errors.New("facility already at max quantity")
	ErrMaxLevel          = errors.New("facility already at max level")
	ErrInvalidFacility   = errors.New("unknown facility type")
	// ErrMinFacility: the last production building and each resource's last warehouse stay.
	ErrMinFacility = errors.New("cannot sell the last building")
	// ErrStockExceedsCapacity: the remaining warehouses could not hold the current stock.
	ErrStockExceedsCapacity = errors.New("stock would exceed remaining capacity")
)

// StockExceedsCapacityError says how many cases must be sold before a warehouse can be.
type StockExceedsCapacityError struct{ Excess int }

func (e *StockExceedsCapacityError) Error() string { return ErrStockExceedsCapacity.Error() }
func (e *StockExceedsCapacityError) Is(target error) bool {
	return target == ErrStockExceedsCapacity
}
