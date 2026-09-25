package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"lemonade-api/internal/auth"
	"lemonade-api/internal/domain"
	"lemonade-api/internal/store"
)

type authEnv struct {
	t      *testing.T
	router *gin.Engine
	repo   *store.Memory
	fake   *auth.Fake
	now    time.Time
}

// newAuthEnv builds the API in firebase mode (no dev bypass) with a fake verifier.
// Tokens "tok-a" and "tok-b" are valid Google identities for uid-a and uid-b.
func newAuthEnv(t *testing.T, opts ...Option) *authEnv {
	gin.SetMode(gin.TestMode)
	e := &authEnv{t: t, repo: store.NewMemory(), fake: auth.NewFake(), now: time.Unix(1_000_000, 0)}
	for _, u := range []string{"a", "b"} {
		e.fake.Add("tok-"+u, auth.Identity{UID: "uid-" + u, Email: u + "@example.com", EmailVerified: true, Provider: "google.com"})
	}
	e.fake.Add("tok-password", auth.Identity{UID: "uid-p", Email: "p@example.com", EmailVerified: true, Provider: "password"})
	e.fake.Add("tok-unverified", auth.Identity{UID: "uid-u", Email: "u@example.com", EmailVerified: false, Provider: "google.com"})
	opts = append([]Option{WithVerifier(e.fake), WithClock(func() time.Time { return e.now })}, opts...)
	e.router = gin.New()
	NewGame(e.repo, domain.DefaultConfig(), func() int64 { return 42 }, opts...).Register(e.router)
	return e
}

// do sends a request. authz is the full Authorization header value (may be empty).
func (e *authEnv) do(method, path, authz string, body any) *httptest.ResponseRecorder {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			e.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

func (e *authEnv) wantError(rec *httptest.ResponseRecorder, status int, code string) {
	e.t.Helper()
	if rec.Code != status {
		e.t.Fatalf("status = %d, want %d (%s)", rec.Code, status, rec.Body)
	}
	if got := decode[errorDTO](e.t, rec).Error; got != code {
		e.t.Fatalf("error code = %q, want %q", got, code)
	}
}

func (e *authEnv) createProfile(token, username string) {
	e.t.Helper()
	if rec := e.do("POST", "/api/me/username", "Bearer "+token, map[string]string{"username": username}); rec.Code != http.StatusOK {
		e.t.Fatalf("create profile: %d %s", rec.Code, rec.Body)
	}
}

func TestProtectedRoutesNeedAValidToken(t *testing.T) {
	e := newAuthEnv(t)
	e.createProfile("tok-a", "alice")

	cases := []struct {
		name   string
		header string
		status int
		code   string
	}{
		{"missing", "", 401, "unauthorized"},
		{"not a bearer scheme", "Basic tok-a", 401, "unauthorized"},
		{"empty bearer", "Bearer ", 401, "unauthorized"},
		{"unknown or expired token", "Bearer nope", 401, "unauthorized"},
		{"wrong provider", "Bearer tok-password", 401, "unauthorized"},
		{"unverified email", "Bearer tok-unverified", 401, "unauthorized"},
		{"valid token, no profile", "Bearer tok-b", 403, "profile_required"},
		{"valid", "Bearer tok-a", 200, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := e.do("GET", "/api/game", tc.header, nil)
			if tc.code == "" {
				if rec.Code != tc.status {
					t.Fatalf("status = %d, want %d (%s)", rec.Code, tc.status, rec.Body)
				}
				return
			}
			e.wantError(rec, tc.status, tc.code)
		})
	}
	// Scores and runs are behind the same check.
	for _, path := range []string{"/api/scores", "/api/runs"} {
		e.wantError(e.do("GET", path, "", nil), 401, "unauthorized")
		e.wantError(e.do("GET", path, "Bearer tok-b", nil), 403, "profile_required")
	}
}

func TestBearerSchemeIsCaseInsensitive(t *testing.T) {
	e := newAuthEnv(t)
	e.createProfile("tok-a", "alice")
	if rec := e.do("GET", "/api/game", "bearer tok-a", nil); rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
}

// guest sends a request identified only by a username, as before Google sign-in.
func (e *authEnv) guest(method, path, username string, body any) *httptest.ResponseRecorder {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			e.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if username != "" {
		req.Header.Set("X-Username", username)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

func TestGuestsPlayWithJustAUsername(t *testing.T) {
	e := newAuthEnv(t)
	rec := e.guest("POST", "/api/login", "", map[string]string{"username": " Guesty "})
	if rec.Code != 200 || decode[userDTO](t, rec).Username != "guesty" {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	// Logging in again resumes the same player.
	if again := decode[userDTO](t, e.guest("POST", "/api/login", "", map[string]string{"username": "guesty"})); again.ID != decode[userDTO](t, rec).ID {
		t.Fatalf("second login made a new player: %+v", again)
	}
	if rec := e.guest("GET", "/api/game", "guesty", nil); rec.Code != 200 {
		t.Fatalf("guest game: %d %s", rec.Code, rec.Body)
	}
	e.wantError(e.guest("GET", "/api/game", "", nil), 401, "unauthorized")
	e.wantError(e.guest("GET", "/api/game", "nobody", nil), 401, "unauthorized")
}

func TestLinkingGoogleSecuresTheAccount(t *testing.T) {
	e := newAuthEnv(t)
	e.guest("POST", "/api/login", "", map[string]string{"username": "guesty"})
	e.guest("POST", "/api/game/end-day", "guesty", nil)

	rec := e.do("POST", "/api/me/claim", "Bearer tok-a", map[string]string{"username": "guesty"})
	if rec.Code != 200 {
		t.Fatalf("link: %d %s", rec.Code, rec.Body)
	}
	// Progress carried over, and the token now identifies the player.
	if v := decode[gameViewDTO](t, e.do("GET", "/api/game", "Bearer tok-a", nil)); v.Day != 2 {
		t.Fatalf("day = %d, want 2", v.Day)
	}
	// The bare username no longer works: no game, and no logging in as them either.
	e.wantError(e.guest("GET", "/api/game", "guesty", nil), 401, "account_secured")
	e.wantError(e.guest("POST", "/api/game/end-day", "guesty", nil), 401, "account_secured")
	e.wantError(e.guest("POST", "/api/login", "", map[string]string{"username": "guesty"}), 409, "account_secured")
	// A profile created through Google is secured from the start.
	e.createProfile("tok-b", "bob")
	e.wantError(e.guest("GET", "/api/game", "bob", nil), 401, "account_secured")
}

func TestABearerTokenIsNeverIgnored(t *testing.T) {
	e := newAuthEnv(t)
	e.guest("POST", "/api/login", "", map[string]string{"username": "guesty"})
	// A bad token does not fall back to a username header that also came along.
	req := httptest.NewRequest("GET", "/api/game", nil)
	req.Header.Set("Authorization", "Bearer nope")
	req.Header.Set("X-Username", "guesty")
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	e.wantError(rec, 401, "unauthorized")
	// A good token wins over the header.
	req = httptest.NewRequest("GET", "/api/game", nil)
	req.Header.Set("Authorization", "Bearer tok-b")
	req.Header.Set("X-Username", "guesty")
	rec = httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	e.wantError(rec, 403, "profile_required")
}

func TestWithoutFirebaseGuestsStillPlay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewGame(store.NewMemory(), domain.DefaultConfig(), nil).Register(r)
	do := func(method, path string, hdr map[string]string, body string) int {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}
	if code := do("POST", "/api/login", nil, `{"username":"guesty"}`); code != 200 {
		t.Fatalf("login = %d", code)
	}
	if code := do("GET", "/api/game", map[string]string{"X-Username": "guesty"}, ""); code != 200 {
		t.Fatalf("game = %d", code)
	}
	if code := do("GET", "/api/game", map[string]string{"Authorization": "Bearer tok-a"}, ""); code != 401 {
		t.Fatalf("a bearer token with no verifier = %d, want 401", code)
	}
}

func TestHealthStaysPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterHealth(r)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/api/health", nil))
	if rec.Code != 200 {
		t.Fatalf("health = %d", rec.Code)
	}
}

func TestMe(t *testing.T) {
	e := newAuthEnv(t)
	e.wantError(e.do("GET", "/api/me", "", nil), 401, "unauthorized")
	e.wantError(e.do("GET", "/api/me", "Bearer tok-password", nil), 401, "unauthorized")
	e.wantError(e.do("GET", "/api/me", "Bearer tok-a", nil), 403, "profile_required")

	e.createProfile("tok-a", "alice")
	rec := e.do("GET", "/api/me", "Bearer tok-a", nil)
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := decode[userDTO](t, rec); got.Username != "alice" || got.ID == 0 {
		t.Fatalf("me = %+v", got)
	}
}

func TestCreateProfile(t *testing.T) {
	e := newAuthEnv(t)
	post := func(token, username string) *httptest.ResponseRecorder {
		return e.do("POST", "/api/me/username", "Bearer "+token, map[string]string{"username": username})
	}

	e.wantError(e.do("POST", "/api/me/username", "", map[string]string{"username": "alice"}), 401, "unauthorized")
	for _, bad := range []string{"", "ab", "has space", strings.Repeat("x", 41), "héllo"} {
		e.wantError(post("tok-a", bad), 400, "invalid_username")
	}

	// Names are normalized like today: trimmed and lowercased.
	rec := post("tok-a", "  Alice ")
	if rec.Code != 200 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if got := decode[userDTO](t, rec); got.Username != "alice" {
		t.Fatalf("username = %q", got.Username)
	}
	// The new player has a game, ready to play.
	if rec := e.do("GET", "/api/game", "Bearer tok-a", nil); rec.Code != 200 {
		t.Fatalf("no game after profile: %d", rec.Code)
	}

	e.wantError(post("tok-b", "ALICE"), 409, "username_taken")
	e.wantError(post("tok-a", "another"), 409, "already_linked")
}

func TestClaimLegacyAccount(t *testing.T) {
	e := newAuthEnv(t)
	// A legacy account with progress, made the old way.
	legacy, err := e.repo.CreateUserWithGame(t.Context(), "oldtimer", domain.NewGame(domain.DefaultConfig(), 9))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.repo.Mutate(t.Context(), legacy.ID, func(g *domain.Game) (domain.Effects, error) {
		g.Capital = 777
		return domain.Effects{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	claim := func(token, username string) *httptest.ResponseRecorder {
		return e.do("POST", "/api/me/claim", "Bearer "+token, map[string]string{"username": username})
	}

	e.wantError(e.do("POST", "/api/me/claim", "", map[string]string{"username": "oldtimer"}), 401, "unauthorized")
	e.wantError(claim("tok-a", "x"), 400, "invalid_username")
	e.wantError(claim("tok-a", "nobody"), 404, "unknown_username")

	rec := claim("tok-a", " OldTimer ")
	if rec.Code != 200 {
		t.Fatalf("claim: %d %s", rec.Code, rec.Body)
	}
	if got := decode[userDTO](t, rec); got.Username != "oldtimer" || got.ID != legacy.ID {
		t.Fatalf("claimed = %+v", got)
	}
	// Progress carried over.
	if v := decode[gameViewDTO](t, e.do("GET", "/api/game", "Bearer tok-a", nil)); v.Capital != 777 {
		t.Fatalf("capital = %d, want 777", v.Capital)
	}

	e.wantError(claim("tok-b", "oldtimer"), 409, "already_claimed")
	e.createProfile("tok-b", "bob")
	e.wantError(claim("tok-b", "oldtimer"), 409, "already_linked") // bob has a profile already
	if _, err := e.repo.CreateUserWithGame(t.Context(), "second", domain.NewGame(domain.DefaultConfig(), 9)); err != nil {
		t.Fatal(err)
	}
	e.wantError(claim("tok-b", "second"), 409, "already_linked")
	e.wantError(claim("tok-a", "second"), 409, "already_linked")
}

func TestClaimIsRateLimitedPerUID(t *testing.T) {
	e := newAuthEnv(t)
	claim := func(token string) *httptest.ResponseRecorder {
		return e.do("POST", "/api/me/claim", "Bearer "+token, map[string]string{"username": "nobody"})
	}
	for i := 0; i < claimAttempts; i++ {
		e.wantError(claim("tok-a"), 404, "unknown_username")
	}
	e.wantError(claim("tok-a"), 429, "rate_limited")
	// Another player is unaffected, and the window eventually reopens.
	e.wantError(claim("tok-b"), 404, "unknown_username")
	e.now = e.now.Add(claimWindow + time.Second)
	e.wantError(claim("tok-a"), 404, "unknown_username")
}

// No response may carry an email or Google name: usernames are the only public identity.
func TestNoEmailInAnyResponseOrDTO(t *testing.T) {
	e := newAuthEnv(t)
	e.createProfile("tok-a", "alice")
	for _, path := range []string{"/api/me", "/api/game", "/api/scores", "/api/runs"} {
		rec := e.do("GET", path, "Bearer tok-a", nil)
		body := strings.ToLower(rec.Body.String())
		if strings.Contains(body, "email") || strings.Contains(body, "a@example.com") {
			t.Errorf("%s leaks an email: %s", path, rec.Body)
		}
	}
	for _, typ := range []reflect.Type{reflect.TypeOf(userDTO{}), reflect.TypeOf(scoreRowDTO{})} {
		for i := 0; i < typ.NumField(); i++ {
			if n := strings.ToLower(typ.Field(i).Name); strings.Contains(n, "email") || strings.Contains(n, "uid") {
				t.Errorf("%s has field %s", typ.Name(), typ.Field(i).Name)
			}
		}
	}
}
