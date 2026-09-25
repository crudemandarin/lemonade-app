package store

import (
	"context"
	"errors"
	"time"

	"lemonade-api/internal/domain"
)

// ErrNotFound is returned when a user or game does not exist.
var ErrNotFound = errors.New("not found")

var (
	// ErrUsernameTaken: another player already uses this username.
	ErrUsernameTaken = errors.New("username taken")
	// ErrAlreadyLinked: this Firebase UID already has a profile.
	ErrAlreadyLinked = errors.New("uid already linked")
	// ErrAlreadyClaimed: this username already belongs to a Firebase account.
	ErrAlreadyClaimed = errors.New("username already claimed")
)

// ScoreRow is one player's best finished run on the global board.
type ScoreRow struct {
	// Rank is 1-based: highest score first, an earlier finish wins a tie.
	Rank      int
	UserID    uint
	Username  string
	RunID     string
	Score     int
	Days      int
	NetWorth  int
	CreatedAt time.Time
}

// RunSummary is one line of a player's record history.
type RunSummary struct {
	RunID     string
	Score     int
	Days      int
	NetWorth  int
	Capital   int
	EndedBy   string
	CreatedAt time.Time
}

// RunDetail is a finished run in full.
type RunDetail struct {
	domain.RunRecord
	CreatedAt time.Time
}

// Repository persists users and their single game. The domain never sees it:
// handlers load a game, call a domain function, and save the result.
type Repository interface {
	// FindUser returns the user with this username, or ErrNotFound.
	FindUser(ctx context.Context, username string) (domain.User, error)

	// CreateUserWithGame creates a user and their first game atomically.
	CreateUserWithGame(ctx context.Context, username string, game domain.Game) (domain.User, error)

	// FindGuestUser returns the user a bare username identifies: ErrNotFound if there is
	// none, ErrAlreadyClaimed if the account is secured with Google (then only a token
	// identifies it). Username-only play is for accounts nobody has secured.
	FindGuestUser(ctx context.Context, username string) (domain.User, error)

	// FindUserByUID returns the user linked to this Firebase UID, or ErrNotFound.
	// An empty uid never matches (legacy users have none).
	FindUserByUID(ctx context.Context, uid string) (domain.User, error)

	// CreateProfile creates a user linked to a Firebase UID, with their first game,
	// atomically. ErrAlreadyLinked if the UID has a profile, ErrUsernameTaken if the
	// username is in use. The email is private: it is stored, never returned.
	CreateProfile(ctx context.Context, username, uid, email string, game domain.Game) (domain.User, error)

	// ClaimUser links a legacy user (one with no UID) to a Firebase UID, keeping their
	// game and runs. ErrAlreadyLinked if the UID has a profile, ErrNotFound if the
	// username does not exist, ErrAlreadyClaimed if it is already linked. Of two
	// simultaneous claims exactly one wins.
	ClaimUser(ctx context.Context, username, uid, email string) (domain.User, error)

	// GetGame returns the user's game, or ErrNotFound.
	GetGame(ctx context.Context, userID uint) (domain.Game, error)

	// ReplaceGame overwrites the user's game with a fresh one.
	ReplaceGame(ctx context.Context, userID uint, game domain.Game) (domain.Game, error)

	// RunBelongsTo reports whether a finished run was played by this user.
	RunBelongsTo(ctx context.Context, userID uint, runID string) (bool, error)

	// ListReports returns every stored day report of a run, oldest day first
	// (empty, not an error, when there are none).
	ListReports(ctx context.Context, runID string) ([]domain.DayReport, error)

	// GetReport returns one day's report of a run, or ErrNotFound.
	GetReport(ctx context.Context, runID string, day int) (domain.DayReport, error)

	// TopScores is the global board: each player's best finished run, best first,
	// at most limit rows. Runs still in progress never appear.
	TopScores(ctx context.Context, limit int) ([]ScoreRow, error)

	// BestScore is the player's own row on the board, with its rank however far down
	// it is, or nil when they have no finished run.
	BestScore(ctx context.Context, userID uint) (*ScoreRow, error)

	// UserRuns lists the player's finished runs, newest first (empty, not nil, when none).
	UserRuns(ctx context.Context, userID uint) ([]RunSummary, error)

	// GetRun returns one finished run of this player, or ErrNotFound (also for
	// someone else's run).
	GetRun(ctx context.Context, userID uint, runID string) (RunDetail, error)

	// Mutate loads the user's game under a row lock, calls fn, and saves the result
	// and fn's effects (a day report, a finished run) in one transaction. If fn
	// returns an error nothing is saved.
	Mutate(ctx context.Context, userID uint, fn func(g *domain.Game) (domain.Effects, error)) (domain.Game, error)
}
