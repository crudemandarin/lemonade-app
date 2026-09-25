// Package auth verifies who is calling. It knows nothing about the game: the API
// layer turns an Identity into a player.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrInvalidToken covers every way a token can be unusable: missing, malformed,
// expired, wrong audience, wrong provider, unverified email.
var ErrInvalidToken = errors.New("invalid token")

// Identity is what a verified token says about the caller. Email is private: it is
// stored for the account and never returned by any endpoint.
type Identity struct {
	UID           string
	Email         string
	EmailVerified bool
	Provider      string // Firebase sign_in_provider, e.g. "google.com"
}

// googleProvider is the only sign-in method this app accepts, even if others are
// enabled in the Firebase console later.
const googleProvider = "google.com"

// Acceptable applies the app's policy on top of a cryptographically valid token.
func (id Identity) Acceptable() error {
	switch {
	case id.UID == "":
		return fmt.Errorf("%w: no uid", ErrInvalidToken)
	case id.Provider != googleProvider:
		return fmt.Errorf("%w: sign-in provider %q not accepted", ErrInvalidToken, id.Provider)
	case !id.EmailVerified:
		return fmt.Errorf("%w: email not verified", ErrInvalidToken)
	}
	return nil
}

// TokenVerifier turns a bearer token into an Identity.
type TokenVerifier interface {
	Verify(ctx context.Context, idToken string) (Identity, error)
}

// Mode selects how requests are authenticated.
type Mode string

const (
	// ModeFirebase (the default) accepts Firebase ID tokens only.
	ModeFirebase Mode = "firebase"
	// ModeDev also accepts the old X-Username header and keeps POST /api/login,
	// for fast local loops and tests. Refused in production.
	ModeDev Mode = "dev"
)

// Config is the auth part of the process environment.
type Config struct {
	Mode              Mode
	FirebaseProjectID string
	AppEnv            string // APP_ENV
	OnCloudRun        bool   // K_SERVICE is set
}

// Validate refuses configurations that would be unsafe or unusable. Startup must
// fail on error.
func (c Config) Validate() error {
	switch c.Mode {
	case "", ModeFirebase:
		if c.FirebaseProjectID == "" {
			return errors.New("AUTH_MODE=firebase needs FIREBASE_PROJECT_ID")
		}
	case ModeDev:
		if c.OnCloudRun || strings.EqualFold(c.AppEnv, "production") {
			return errors.New("AUTH_MODE=dev is refused in production (Cloud Run or APP_ENV=production)")
		}
	default:
		return fmt.Errorf("AUTH_MODE must be %q or %q, got %q", ModeFirebase, ModeDev, c.Mode)
	}
	return nil
}

// Fake is a TokenVerifier for tests: tokens are registered up front.
type Fake struct {
	mu     sync.Mutex
	tokens map[string]Identity
}

func NewFake() *Fake { return &Fake{tokens: map[string]Identity{}} }

// Add registers a token that verifies to id.
func (f *Fake) Add(token string, id Identity) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokens[token] = id
}

func (f *Fake) Verify(_ context.Context, token string) (Identity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.tokens[token]
	if !ok {
		return Identity{}, ErrInvalidToken
	}
	return id, nil
}
