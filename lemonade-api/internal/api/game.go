package api

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"lemonade-api/internal/auth"
	"lemonade-api/internal/domain"
	"lemonade-api/internal/domain/content"
	"lemonade-api/internal/store"
)

const (
	minUsernameLen = 3
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

	verifier auth.TokenVerifier // nil: Google sign-in is off, no bearer token is accepted
	claims   *claimLimiter
}

// newRunID is a random v4 UUID naming one playthrough.
func newRunID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err) // the OS random source failing is not recoverable
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// freshGame starts a new run with its own ID.
func (h *Game) freshGame() domain.Game {
	g := domain.NewGame(h.cfg, h.newSeed())
	g.RunID = newRunID()
	return g
}

// NewGame builds the game API. Pass a nil newSeed to seed from the clock.
func NewGame(repo store.Repository, cfg domain.Config, newSeed func() int64, opts ...Option) *Game {
	if newSeed == nil {
		newSeed = func() int64 { return time.Now().UnixNano() }
	}
	h := &Game{repo: repo, cfg: cfg, newSeed: newSeed, claims: newClaimLimiter(time.Now)}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Register mounts the game routes under /api.
func (h *Game) Register(router gin.IRouter) {
	api := router.Group("/api")
	api.POST("/login", h.login)

	me := api.Group("/me", h.requireIdentity)
	me.GET("", h.me)
	me.POST("/username", h.createProfile)
	me.POST("/claim", h.claim)

	api.GET("/scores", h.requireUser, h.scores)
	api.GET("/achievements", h.requireUser, h.listAchievements)
	api.GET("/runs", h.requireUser, h.myRuns)
	api.GET("/runs/:runId", h.requireUser, h.myRun)

	g := api.Group("/game", h.requireUser)
	g.GET("", h.getGame)
	g.POST("/new", h.newGame)
	g.POST("/buy", h.trade(domain.Buy, domain.BuyClamped))
	g.POST("/sell", h.trade(domain.Sell, domain.SellClamped))
	g.POST("/facilities/:kind/expand", h.expand)
	g.POST("/facilities/:kind/upgrade", h.upgrade)
	g.POST("/facilities/:kind/sell", h.sellFacility)
	g.POST("/give-up", h.giveUp)
	g.GET("/quote", h.quote)
	g.GET("/reports", h.listReports)
	g.GET("/reports/:day", h.getReport)
	g.POST("/end-day", h.endDay)
}

func currentUser(c *gin.Context) domain.User {
	return c.MustGet(userKey).(domain.User)
}

// login starts or resumes a username-only (guest) player. A username that has been
// secured with Google cannot be logged into this way: it needs the token.
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
		abort(c, http.StatusBadRequest, "invalid_username", "Username must be 3 to 40 standard ASCII characters, with no spaces.")
		return
	}

	ctx := c.Request.Context()
	user, err := h.repo.FindGuestUser(ctx, username)
	if errors.Is(err, store.ErrNotFound) {
		user, err = h.repo.CreateUserWithGame(ctx, username, h.freshGame())
	}
	if errors.Is(err, store.ErrAlreadyClaimed) {
		abort(c, http.StatusConflict, "account_secured", "This username is protected. Sign in with Google to play it.")
		return
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
	h.respondView(c, g)
}

// newGame starts a fresh run, but only once the last one is over: an unfinished run
// must be given up first, so every run ends with a record.
func (h *Game) newGame(c *gin.Context) {
	current, err := h.repo.GetGame(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		abortErr(c, err)
		return
	}
	if current.Status == domain.StatusActive {
		abort(c, http.StatusConflict, "run_active", "Give up your current game before starting a new one.")
		return
	}
	g, err := h.repo.ReplaceGame(c.Request.Context(), currentUser(c).ID, h.freshGame())
	if err != nil {
		abortErr(c, err)
		return
	}
	h.respondView(c, g)
}

// view is the game view plus the caller's personal best, so the page can show it
// and celebrate a new one.
func (h *Game) view(c *gin.Context, g domain.Game) (gameViewDTO, error) {
	v := toGameView(g, h.cfg)
	best, err := h.repo.BestScore(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		return v, err
	}
	if best != nil {
		v.Best = &bestDTO{Score: best.Score, Days: best.Days, RunID: best.RunID}
	}
	return v, nil
}

// respondView writes the game view, or the error that stopped it.
func (h *Game) respondView(c *gin.Context, g domain.Game) {
	v, err := h.view(c, g)
	if err != nil {
		abortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

// respondViewUnlocking is respondView plus the achievements the mutation unlocked.
func (h *Game) respondViewUnlocking(c *gin.Context, g domain.Game, unlocked []string) {
	v, err := h.view(c, g)
	if err != nil {
		abortErr(c, err)
		return
	}
	v.Unlocked = toUnlocked(unlocked)
	c.JSON(http.StatusOK, v)
}

// mutate runs one domain action under the row lock and responds with the new view.
func (h *Game) mutate(c *gin.Context, action func(g *domain.Game) error) {
	g, e, ok := h.mutateWithEffects(c, func(g *domain.Game) (domain.Effects, error) { return domain.Effects{}, action(g) })
	if !ok {
		return
	}
	h.respondViewUnlocking(c, g, e.Unlocked)
}

// mutateWithEffects runs one action under the row lock and saves its effects with
// the game. A game saved before runs existed gets its run ID here, on its first
// mutation. On failure it has already answered, and ok is false.
func (h *Game) mutateWithEffects(c *gin.Context, action func(g *domain.Game) (domain.Effects, error)) (g domain.Game, e domain.Effects, ok bool) {
	return h.mutateAchieving(c, false, action)
}

// mutateAchieving is mutateWithEffects plus achievements: after the action succeeds
// they are evaluated on the game before and after it, and the new ones are saved in
// the same transaction (a failed action grants nothing). mayFinish says the action can
// end the run, which needs the player's run history and board rank, loaded first
// because the repository holds its lock during the action.
func (h *Game) mutateAchieving(c *gin.Context, mayFinish bool, action func(g *domain.Game) (domain.Effects, error)) (g domain.Game, e domain.Effects, ok bool) {
	user := currentUser(c)
	history, err := h.loadHistory(c.Request.Context(), user.ID, mayFinish)
	if err != nil {
		abortErr(c, err)
		return g, e, false
	}
	g, err = h.repo.Mutate(c.Request.Context(), user.ID, func(g *domain.Game) (domain.Effects, error) {
		if g.RunID == "" {
			g.RunID = newRunID()
		}
		before := g.Clone()
		var err error
		e, err = action(g)
		if err != nil {
			return e, err
		}
		e.Unlocked = domain.Evaluate(content.Achievements, before, *g, h.cfg, history.achievementContext(user.ID, e.Finished, time.Now()))
		return e, nil
	})
	if err != nil {
		abortErr(c, err)
		return g, e, false
	}
	return g, e, true
}

type tradeRequest struct {
	Resource domain.Resource `json:"resource"`
	Qty      int             `json:"qty"`
	// Clamp trades as many cases as possible, up to Qty, instead of failing
	// when the full quantity cannot be done.
	Clamp bool `json:"clamp"`
}

// trade builds the buy or sell handler; both share one request shape. The
// all-or-nothing action is used unless the request sets clamp.
func (h *Game) trade(exact, clamped func(*domain.Game, domain.Config, domain.Resource, int) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req tradeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			abort(c, http.StatusBadRequest, "invalid_request", "Request body must be JSON with a resource and qty.")
			return
		}
		if !h.cfg.Valid(req.Resource) {
			abort(c, http.StatusBadRequest, "invalid_resource", "Unknown resource.")
			return
		}
		action := exact
		if req.Clamp {
			action = clamped
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

	resource, ok := h.warehouseResource(c, kind)
	if !ok {
		return
	}
	h.mutate(c, func(g *domain.Game) error { return domain.Expand(g, h.cfg, kind, resource) })
}

// warehouseResource reads the {resource} body a warehouse action needs; production takes none.
func (h *Game) warehouseResource(c *gin.Context, kind domain.FacilityType) (domain.Resource, bool) {
	if kind != domain.Warehouse {
		return "", true
	}
	var req struct {
		Resource domain.Resource `json:"resource"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, http.StatusBadRequest, "invalid_request", "Request body must be JSON with a resource.")
		return "", false
	}
	if !h.cfg.Valid(req.Resource) {
		abort(c, http.StatusBadRequest, "invalid_resource", "Unknown resource.")
		return "", false
	}
	return req.Resource, true
}

func (h *Game) sellFacility(c *gin.Context) {
	kind, ok := facilityKind(c)
	if !ok {
		return
	}
	resource, ok := h.warehouseResource(c, kind)
	if !ok {
		return
	}
	h.mutate(c, func(g *domain.Game) error { return domain.SellFacility(g, h.cfg, kind, resource) })
}

func (h *Game) upgrade(c *gin.Context) {
	kind, ok := facilityKind(c)
	if !ok {
		return
	}
	h.mutate(c, func(g *domain.Game) error { return domain.Upgrade(g, h.cfg, kind) })
}

// reportRun picks the run whose reports are wanted: the current one, or a finished
// run of the same user given as ?runId=. Anyone else's run is a 404, the same as
// a run that does not exist. On failure it has already answered.
func (h *Game) reportRun(c *gin.Context) (string, bool) {
	ctx := c.Request.Context()
	current, err := h.repo.GetGame(ctx, currentUser(c).ID)
	if err != nil {
		abortErr(c, err)
		return "", false
	}
	runID := c.Query("runId")
	if runID == "" || runID == current.RunID {
		return current.RunID, true
	}
	owned, err := h.repo.RunBelongsTo(ctx, currentUser(c).ID, runID)
	if err != nil {
		abortErr(c, err)
		return "", false
	}
	if !owned {
		abortErr(c, store.ErrNotFound)
		return "", false
	}
	return runID, true
}

// listReports returns a light summary of every ended day of a run, oldest first.
func (h *Game) listReports(c *gin.Context) {
	runID, ok := h.reportRun(c)
	if !ok {
		return
	}
	reports, err := h.repo.ListReports(c.Request.Context(), runID)
	if err != nil {
		abortErr(c, err)
		return
	}
	out := make([]reportSummaryDTO, 0, len(reports))
	for _, r := range reports {
		out = append(out, toReportSummary(r))
	}
	c.JSON(http.StatusOK, out)
}

// getReport returns one day's full report.
func (h *Game) getReport(c *gin.Context) {
	day, err := strconv.Atoi(c.Param("day"))
	if err != nil || day < 1 {
		abort(c, http.StatusBadRequest, "invalid_day", "Day must be a positive whole number.")
		return
	}
	runID, ok := h.reportRun(c)
	if !ok {
		return
	}
	report, err := h.repo.GetReport(c.Request.Context(), runID, day)
	if err != nil {
		abortErr(c, err)
		return
	}
	c.JSON(http.StatusOK, toDayReport(report))
}

// maxQuoteQty bounds a quote request: it is a price check, not a trade.
const maxQuoteQty = 100_000

// quoteDTO answers GET /api/game/quote.
type quoteDTO struct {
	Resource domain.Resource `json:"resource"`
	Side     string          `json:"side"`
	tradeQuoteDTO
}

// quote prices a trade of some size, including price impact, without making it. With
// clamp=true it is cut down to what cash, space (buy) or stock (sell) allow, like a bulk trade.
func (h *Game) quote(c *gin.Context) {
	r := domain.Resource(c.Query("resource"))
	if !h.cfg.Valid(r) {
		abort(c, http.StatusBadRequest, "invalid_resource", "Unknown resource.")
		return
	}
	side := c.Query("side")
	if side != "buy" && side != "sell" {
		abort(c, http.StatusBadRequest, "invalid_side", "Side must be buy or sell.")
		return
	}
	qty, err := strconv.Atoi(c.Query("qty"))
	if err != nil || qty < 1 || qty > maxQuoteQty {
		abort(c, http.StatusBadRequest, "invalid_quantity", "Quantity must be a whole number from 1 to 100000.")
		return
	}
	clamp := c.Query("clamp") == "true"

	g, err := h.repo.GetGame(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		abortErr(c, err)
		return
	}
	var q domain.TradeQuote
	if side == "buy" {
		q = domain.QuoteBuy(g, h.cfg, r, qty, clamp)
	} else {
		q = domain.QuoteSell(g, h.cfg, r, qty, clamp)
	}
	c.JSON(http.StatusOK, quoteDTO{Resource: r, Side: side, tradeQuoteDTO: toTradeQuote(q)})
}

// giveUp ends the run and records it, in one transaction.
func (h *Game) giveUp(c *gin.Context) {
	g, e, ok := h.mutateAchieving(c, true, func(g *domain.Game) (domain.Effects, error) {
		if err := domain.GiveUp(g); err != nil {
			return domain.Effects{}, err
		}
		rec := domain.FinishRun(*g, h.cfg, domain.EndedByGaveUp)
		return domain.Effects{Finished: &rec}, nil
	})
	if !ok {
		return
	}
	h.respondViewUnlocking(c, g, e.Unlocked)
}

func (h *Game) endDay(c *gin.Context) {
	var report domain.DayReport
	g, e, ok := h.mutateAchieving(c, true, func(g *domain.Game) (domain.Effects, error) {
		before := g.Clone()
		var err error
		report, err = domain.EndDay(g, h.cfg)
		if err != nil {
			return domain.Effects{}, err
		}
		// Goal facts are recorded after EndDay, not inside it, so the rules never see them.
		domain.RecordDayFacts(before, g, h.cfg, report)
		effects := domain.Effects{Report: &report}
		if report.Bankrupt {
			// Same transaction as the status change: no bankrupt game without its record.
			rec := domain.FinishRun(*g, h.cfg, domain.EndedByBankrupt)
			effects.Finished = &rec
		}
		return effects, nil
	})
	if !ok {
		return
	}
	view, err := h.view(c, g)
	if err != nil {
		abortErr(c, err)
		return
	}
	view.Unlocked = toUnlocked(e.Unlocked)
	c.JSON(http.StatusOK, endDayResponseDTO{Report: toDayReport(report), Game: view})
}
