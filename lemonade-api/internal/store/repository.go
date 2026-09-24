package store

import (
	"context"
	"errors"

	"lemonade-api/internal/domain"
)

// ErrNotFound is returned when a user or game does not exist.
var ErrNotFound = errors.New("not found")

// Repository persists users and their single game. The domain never sees it:
// handlers load a game, call a domain function, and save the result.
type Repository interface {
	// FindUser returns the user with this username, or ErrNotFound.
	FindUser(ctx context.Context, username string) (domain.User, error)

	// CreateUserWithGame creates a user and their first game atomically.
	CreateUserWithGame(ctx context.Context, username string, game domain.Game) (domain.User, error)

	// GetGame returns the user's game, or ErrNotFound.
	GetGame(ctx context.Context, userID uint) (domain.Game, error)

	// ReplaceGame overwrites the user's game with a fresh one.
	ReplaceGame(ctx context.Context, userID uint, game domain.Game) (domain.Game, error)

	// Mutate loads the user's game under a row lock, calls fn, and saves the result
	// in one transaction. If fn returns an error nothing is saved.
	Mutate(ctx context.Context, userID uint, fn func(g *domain.Game) error) (domain.Game, error)
}
