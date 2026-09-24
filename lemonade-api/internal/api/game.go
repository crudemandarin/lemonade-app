package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"lemonade-api/internal/domain"
	"lemonade-api/internal/store"
)

const (
	usernameHeader = "X-Username"
	userKey        = "user"
	minUsernameLen = 5
	maxUsernameLen = 40
)

// normalizeUsername trims and lowercases a username so logins are case-insensitive,
// and reports whether the result is minUsernameLen to maxUsernameLen printable ASCII characters
// (letters, digits and punctuation; no spaces or control characters).
func normalizeUsername(raw string) (string, bool) {
	name := strings.ToLower(strings.TrimSpace(raw))
	if len(name) < minUsernameLen || len(name) > maxUsernameLen {
		return "", false
	}
	for i := 0; i < len(name); i++ {
		if name[i] <= ' ' || name[i] > '~' {
			return "", false
		}
	}
	return name, true
}

// Game serves the /api game endpoints. Handlers stay thin: load, call a domain
// function, save, and map the result to a DTO.
type Game struct {
	repo    store.Repository
	cfg     domain.Config
	newSeed func() int64
}

// NewGame builds the game API. Pass a nil newSeed to seed from the clock.
func NewGame(repo store.Repository, cfg domain.Config, newSeed func() int64) *Game {
	if newSeed == nil {
		newSeed = func() int64 { return time.Now().UnixNano() }
	}
	return &Game{repo: repo, cfg: cfg, newSeed: newSeed}
}

// Register mounts the game routes under /api.
func (h *Game) Register(router gin.IRouter) {
	api := router.Group("/api")
	api.POST("/login", h.login)

	g := api.Group("/game", h.requireUser)
	g.GET("", h.getGame)
	g.POST("/new", h.newGame)
	g.POST("/buy", h.trade(domain.Buy))
	g.POST("/sell", h.trade(domain.Sell))
	g.POST("/facilities/:kind/expand", h.expand)
	g.POST("/facilities/:kind/upgrade", h.upgrade)
	g.POST("/end-day", h.endDay)
}

// requireUser identifies the player from X-Username. Intentionally not secure
// (SPEC rule 22).
func (h *Game) requireUser(c *gin.Context) {
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

func currentUser(c *gin.Context) domain.User {
	return c.MustGet(userKey).(domain.User)
}

func (h *Game) login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "invalid_request", "Request body must be JSON with a username.")
		return
	}
	username, ok := normalizeUsername(req.Username)
	if !ok {
		abort(c, http.StatusBadRequest, "invalid_username", "Username must be 5 to 40 standard ASCII characters, with no spaces.")
		return
	}

	ctx := c.Request.Context()
	user, err := h.repo.FindUser(ctx, username)
	if errors.Is(err, store.ErrNotFound) {
		user, err = h.repo.CreateUserWithGame(ctx, username, domain.NewGame(h.cfg, h.newSeed()))
	}
	if err != nil {
		abortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, userDTO{ID: user.ID, Username: user.Username})
}

func (h *Game) getGame(c *gin.Context) {
	g, err := h.repo.GetGame(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		abortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, toGameView(g, h.cfg))
}

func (h *Game) newGame(c *gin.Context) {
	g, err := h.repo.ReplaceGame(c.Request.Context(), currentUser(c).ID, domain.NewGame(h.cfg, h.newSeed()))
	if err != nil {
		abortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, toGameView(g, h.cfg))
}

// mutate runs one domain action under the row lock and responds with the new view.
func (h *Game) mutate(c *gin.Context, action func(g *domain.Game) error) {
	g, err := h.repo.Mutate(c.Request.Context(), currentUser(c).ID, action)
	if err != nil {
		abortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, toGameView(g, h.cfg))
}

type tradeRequest struct {
	Resource domain.Resource `json:"resource"`
	Qty      int             `json:"qty"`
}

// trade builds the buy or sell handler; both share one request shape.
func (h *Game) trade(action func(*domain.Game, domain.Config, domain.Resource, int) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req tradeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			abort(c, http.StatusBadRequest, "invalid_request", "Request body must be JSON with a resource and qty.")
			return
		}
		if !req.Resource.Valid() {
			abort(c, http.StatusBadRequest, "invalid_resource", "Unknown resource.")
			return
		}
		h.mutate(c, func(g *domain.Game) error { return action(g, h.cfg, req.Resource, req.Qty) })
	}
}

func facilityKind(c *gin.Context) (domain.FacilityType, bool) {
	kind := domain.FacilityType(c.Param("kind"))
	if !kind.Valid() {
		abort(c, http.StatusBadRequest, "invalid_facility_type", "Facility type must be warehouse or production.")
		return "", false
	}
	return kind, true
}

func (h *Game) expand(c *gin.Context) {
	kind, ok := facilityKind(c)
	if !ok {
		return
	}

	var resource domain.Resource
	if kind == domain.Warehouse {
		var req struct {
			Resource domain.Resource `json:"resource"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			abort(c, http.StatusBadRequest, "invalid_request", "Request body must be JSON with a resource.")
			return
		}
		if !req.Resource.Valid() {
			abort(c, http.StatusBadRequest, "invalid_resource", "Unknown resource.")
			return
		}
		resource = req.Resource
	}
	h.mutate(c, func(g *domain.Game) error { return domain.Expand(g, h.cfg, kind, resource) })
}

func (h *Game) upgrade(c *gin.Context) {
	kind, ok := facilityKind(c)
	if !ok {
		return
	}
	h.mutate(c, func(g *domain.Game) error { return domain.Upgrade(g, h.cfg, kind) })
}

func (h *Game) endDay(c *gin.Context) {
	var report domain.DayReport
	g, err := h.repo.Mutate(c.Request.Context(), currentUser(c).ID, func(g *domain.Game) error {
		var err error
		report, err = domain.EndDay(g, h.cfg)
		return err
	})
	if err != nil {
		abortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, endDayResponseDTO{Report: toDayReport(report), Game: toGameView(g, h.cfg)})
}
