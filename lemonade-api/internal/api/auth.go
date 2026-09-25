package api

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"lemonade-api/internal/auth"
	"lemonade-api/internal/store"
)

const (
	usernameHeader = "X-Username"
	userKey        = "user"
	identityKey    = "identity"

	// A player gets claimAttempts claim tries per claimWindow: enough for typos,
	// too few to walk the list of legacy usernames.
	claimAttempts = 5
	claimWindow   = 10 * time.Minute
)

// Option configures the game API.
type Option func(*Game)

// WithVerifier sets how bearer tokens are verified.
func WithVerifier(v auth.TokenVerifier) Option {
	return func(h *Game) { h.verifier = v }
}

// WithDevAuth turns on the dev bypass: requests may identify themselves with the old
// X-Username header, and POST /api/login exists. main refuses this in production
// (auth.Config.Validate).
func WithDevAuth() Option {
	return func(h *Game) { h.devAuth = true }
}

// WithClock replaces the clock used by the claim rate limiter (for tests).
func WithClock(now func() time.Time) Option {
	return func(h *Game) { h.claims = newClaimLimiter(now) }
}

// bearerToken extracts the token from "Authorization: Bearer <token>". Tokens are
// read from this header only, never from URLs, and are never logged.
func bearerToken(c *gin.Context) (string, bool) {
	scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
	token = strings.TrimSpace(token)
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}
	return token, true
}

// requireIdentity verifies the bearer token and stores the Identity. It does not
// need a profile, so the profile endpoints use it directly.
func (h *Game) requireIdentity(c *gin.Context) {
	if h.verifyIdentity(c) {
		c.Next()
	}
}

// verifyIdentity is requireIdentity without the c.Next(); it aborts and returns false on failure.
func (h *Game) verifyIdentity(c *gin.Context) bool {
	token, ok := bearerToken(c)
	if !ok || h.verifier == nil {
		abort(c, http.StatusUnauthorized, "unauthorized", "Sign in first.")
		return false
	}
	id, err := h.verifier.Verify(c.Request.Context(), token)
	if err == nil {
		err = id.Acceptable()
	}
	if err != nil {
		abort(c, http.StatusUnauthorized, "unauthorized", "Your sign-in is invalid or expired. Sign in again.")
		return false
	}
	c.Set(identityKey, id)
	return true
}

func currentIdentity(c *gin.Context) auth.Identity {
	return c.MustGet(identityKey).(auth.Identity)
}

// requireUser resolves the caller to a player. In dev mode the X-Username header
// still works; otherwise a verified token's UID is mapped to a user row, and a valid
// token with no row gets 403 profile_required so the app can send them to onboarding.
func (h *Game) requireUser(c *gin.Context) {
	if h.devAuth && c.GetHeader(usernameHeader) != "" {
		h.requireDevUser(c)
		return
	}
	if !h.verifyIdentity(c) {
		return
	}
	user, err := h.repo.FindUserByUID(c.Request.Context(), currentIdentity(c).UID)
	if errors.Is(err, store.ErrNotFound) {
		abort(c, http.StatusForbidden, "profile_required", "Choose a username to start playing.")
		return
	}
	if err != nil {
		abortErr(c, err)
		return
	}
	c.Set(userKey, user)
	c.Next()
}

// requireDevUser is the old identification by X-Username (SPEC rule 22), kept for
// dev mode only.
func (h *Game) requireDevUser(c *gin.Context) {
	username, ok := normalizeUsername(c.GetHeader(usernameHeader))
	if !ok {
		abort(c, http.StatusUnauthorized, "unauthorized", "Sign in first: the X-Username header is required.")
		return
	}
	user, err := h.repo.FindUser(c.Request.Context(), username)
	if errors.Is(err, store.ErrNotFound) {
		abort(c, http.StatusUnauthorized, "unauthorized", "Unknown user. Sign in first.")
		return
	}
	if err != nil {
		abortErr(c, err)
		return
	}
	c.Set(userKey, user)
	c.Next()
}

// me returns the caller's public profile: username only, never email.
func (h *Game) me(c *gin.Context) {
	user, err := h.repo.FindUserByUID(c.Request.Context(), currentIdentity(c).UID)
	if errors.Is(err, store.ErrNotFound) {
		abort(c, http.StatusForbidden, "profile_required", "Choose a username to start playing.")
		return
	}
	if err != nil {
		abortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, userDTO{ID: user.ID, Username: user.Username})
}

type usernameRequest struct {
	Username string `json:"username"`
}

// bindUsername reads and validates the username of a profile request.
func bindUsername(c *gin.Context) (string, bool) {
	var req usernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "invalid_request", "Request body must be JSON with a username.")
		return "", false
	}
	username, ok := normalizeUsername(req.Username)
	if !ok {
		abort(c, http.StatusBadRequest, "invalid_username", "Username must be 3 to 40 standard ASCII characters, with no spaces.")
		return "", false
	}
	return username, true
}

// createProfile gives a signed-in player a username and their first game.
func (h *Game) createProfile(c *gin.Context) {
	username, ok := bindUsername(c)
	if !ok {
		return
	}
	id := currentIdentity(c)
	user, err := h.repo.CreateProfile(c.Request.Context(), username, id.UID, id.Email, h.freshGame())
	if err != nil {
		h.abortProfileErr(c, err)
		return
	}
	c.JSON(http.StatusOK, userDTO{ID: user.ID, Username: user.Username})
}

// claim links a legacy username to the signed-in player. First come, first served:
// the old login let anyone type any name (see README, known limitations).
func (h *Game) claim(c *gin.Context) {
	username, ok := bindUsername(c)
	if !ok {
		return
	}
	id := currentIdentity(c)
	if !h.claims.allow(id.UID) {
		abort(c, http.StatusTooManyRequests, "rate_limited", "Too many attempts. Try again in a few minutes.")
		return
	}
	user, err := h.repo.ClaimUser(c.Request.Context(), username, id.UID, id.Email)
	if err != nil {
		h.abortProfileErr(c, err)
		return
	}
	c.JSON(http.StatusOK, userDTO{ID: user.ID, Username: user.Username})
}

func (h *Game) abortProfileErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrUsernameTaken):
		abort(c, http.StatusConflict, "username_taken", "That username is taken.")
	case errors.Is(err, store.ErrAlreadyLinked):
		abort(c, http.StatusConflict, "already_linked", "This Google account already has a player.")
	case errors.Is(err, store.ErrAlreadyClaimed):
		abort(c, http.StatusConflict, "already_claimed", "That username already belongs to another Google account.")
	case errors.Is(err, store.ErrNotFound):
		abort(c, http.StatusNotFound, "unknown_username", "No player has that username.")
	default:
		abortErr(c, err)
	}
}

// claimLimiter allows claimAttempts calls per key per claimWindow. It lives in
// memory, so with several instances the limit is per instance.
type claimLimiter struct {
	mu   sync.Mutex
	now  func() time.Time
	hits map[string][]time.Time
}

func newClaimLimiter(now func() time.Time) *claimLimiter {
	return &claimLimiter{now: now, hits: map[string][]time.Time{}}
}

func (l *claimLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	recent := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if now.Sub(t) < claimWindow {
			recent = append(recent, t)
		}
	}
	if len(recent) >= claimAttempts {
		l.hits[key] = recent
		return false
	}
	l.hits[key] = append(recent, now)
	return true
}
