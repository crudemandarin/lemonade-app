package domain

import "errors"

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
)
