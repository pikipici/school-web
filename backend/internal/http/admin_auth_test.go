package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pikipici/informatika-monitoring/backend/internal/auth/password"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/ratelimit"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/session"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/users"
	httpapi "github.com/pikipici/informatika-monitoring/backend/internal/http"
	"github.com/pikipici/informatika-monitoring/backend/internal/testdb"
)

const (
	goodPassword = "kopi-hangat-di-pagi-hari"
	host         = "example.test"
	origin       = "http://example.test"
	loginPath    = "/api/v1/admin/auth/login"
	logoutPath   = "/api/v1/admin/auth/logout"
	mePath       = "/api/v1/admin/auth/me"
)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time          { return c.now }
func (c *clock) Advance(d time.Duration) { c.now = c.now.Add(d) }

type recorder struct {
	mu     sync.Mutex
	events []httpapi.AuditEvent
}

func (r *recorder) Record(_ context.Context, e httpapi.AuditEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *recorder) all() []httpapi.AuditEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]httpapi.AuditEvent(nil), r.events...)
}

type fixture struct {
	t       *testing.T
	pool    *pgxpool.Pool
	clk     *clock
	hasher  *password.Hasher
	auth    *httpapi.AdminAuth
	handler http.Handler
	audit   *recorder
	userID  int64
	origin  string
}

type fixtureOpts struct {
	secure       bool
	gate         *ratelimit.Gate
	hashWait     time.Duration
	storedHasher *password.Hasher // hasher used to create the stored hash
	loginHasher  *password.Hasher // hasher configured on the handler
}

func cheapParams() password.Params {
	return password.Params{Memory: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
}

func mustHasher(t *testing.T, p password.Params) *password.Hasher {
	t.Helper()
	h, err := password.NewHasher(p)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func newFixture(t *testing.T, o fixtureOpts) *fixture {
	t.Helper()
	pool := testdb.OpenMigrated(t)

	if o.storedHasher == nil {
		o.storedHasher = mustHasher(t, cheapParams())
	}
	if o.loginHasher == nil {
		o.loginHasher = o.storedHasher
	}
	if o.gate == nil {
		o.gate = ratelimit.NewGate(2)
	}
	if o.hashWait == 0 {
		o.hashWait = 2 * time.Second
	}

	clk := &clock{now: time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)}
	hash, err := o.storedHasher.Hash(goodPassword)
	if err != nil {
		t.Fatal(err)
	}
	var uid int64
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := pool.QueryRow(ctx,
		"insert into admin_users (username, password_hash) values ('fikri', $1) returning id", hash).Scan(&uid); err != nil {
		t.Fatal(err)
	}

	audit := &recorder{}
	auth := httpapi.NewAdminAuth(httpapi.AdminAuthConfig{
		Users:        users.New(pool, clk.Now),
		Sessions:     session.New(pool, session.Options{Now: clk.Now}),
		Limiter:      ratelimit.New(ratelimit.Options{Now: clk.Now}),
		Hasher:       o.loginHasher,
		Gate:         o.gate,
		HashWait:     o.hashWait,
		SecureCookie: o.secure,
		Audit:        audit,
	})
	o2 := origin
	if o.secure {
		o2 = "https://example.test"
	}
	return &fixture{
		origin: o2,
		t:      t, pool: pool, clk: clk, hasher: o.loginHasher, auth: auth, audit: audit, userID: uid,
		handler: httpapi.NewRouterWithDeps(httpapi.Deps{Admin: auth}),
	}
}

type req struct {
	method      string
	path        string
	body        string
	contentType string
	origin      string
	csrf        string
	cookie      *http.Cookie
	remoteAddr  string
	xff         string
}

func (f *fixture) do(r req) *httptest.ResponseRecorder {
	f.t.Helper()
	var body *strings.Reader
	if r.body != "" {
		body = strings.NewReader(r.body)
	} else {
		body = strings.NewReader("")
	}
	hr := httptest.NewRequest(r.method, r.path, body)
	hr.Host = host
	if r.remoteAddr == "" {
		r.remoteAddr = "203.0.113.7:51234"
	}
	hr.RemoteAddr = r.remoteAddr
	if r.contentType == "" && r.body != "" {
		r.contentType = "application/json"
	}
	if r.contentType != "" {
		hr.Header.Set("Content-Type", r.contentType)
	}
	if r.origin != "" {
		hr.Header.Set("Origin", r.origin)
	}
	if r.csrf != "" {
		hr.Header.Set("X-CSRF-Token", r.csrf)
	}
	if r.xff != "" {
		hr.Header.Set("X-Forwarded-For", r.xff)
	}
	if r.cookie != nil {
		hr.AddCookie(r.cookie)
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, hr)
	return rec
}

func loginBody(user, pass string) string {
	b, _ := json.Marshal(map[string]string{"username": user, "password": pass})
	return string(b)
}

func (f *fixture) login(user, pass string) *httptest.ResponseRecorder {
	f.t.Helper()
	return f.do(req{method: "POST", path: loginPath, body: loginBody(user, pass), origin: f.origin})
}

type authData struct {
	Data struct {
		Username  string `json:"username"`
		CSRFToken string `json:"csrfToken"`
		ExpiresAt string `json:"expiresAt"`
	} `json:"data"`
}

type errBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if strings.HasSuffix(c.Name, "admin_session") {
			return c
		}
	}
	t.Fatal("no session cookie set")
	return nil
}

func (f *fixture) loggedIn() (*http.Cookie, string) {
	f.t.Helper()
	rec := f.login("fikri", goodPassword)
	if rec.Code != http.StatusOK {
		f.t.Fatalf("login = %d %s", rec.Code, rec.Body.String())
	}
	return sessionCookie(f.t, rec), decode[authData](f.t, rec).Data.CSRFToken
}

func TestLoginSuccessSetsHardenedCookieAndReturnsCSRFToken(t *testing.T) {
	f := newFixture(t, fixtureOpts{})
	rec := f.login("fikri", goodPassword)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	body := decode[authData](t, rec)
	if body.Data.Username != "fikri" || len(body.Data.CSRFToken) != 43 || body.Data.ExpiresAt == "" {
		t.Fatalf("unexpected body: %+v", body)
	}
	if strings.Contains(rec.Body.String(), "argon2") || strings.Contains(rec.Body.String(), goodPassword) {
		t.Fatal("response leaks password material")
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "no-store") {
		t.Fatalf("Cache-Control = %q; want no-store", cc)
	}

	c := sessionCookie(t, rec)
	if c.Name != "admin_session" {
		t.Fatalf("cookie name = %q", c.Name)
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.Domain != "" {
		t.Fatalf("cookie flags wrong: %+v", c)
	}
	if c.MaxAge != 0 || !c.Expires.IsZero() {
		t.Fatalf("cookie must be a browser-session cookie: %+v", c)
	}
	if c.Secure {
		t.Fatal("Secure flag set although SecureCookie is off")
	}
	if len(c.Value) != 43 || c.Value == body.Data.CSRFToken {
		t.Fatal("cookie value must be the session token, distinct from the CSRF token")
	}
}

func TestSecureModeUsesHostPrefixedCookie(t *testing.T) {
	f := newFixture(t, fixtureOpts{secure: true})
	c := sessionCookie(t, f.login("fikri", goodPassword))

	if c.Name != "__Host-admin_session" || !c.Secure || c.Path != "/" || c.Domain != "" || !c.HttpOnly {
		t.Fatalf("secure cookie wrong: %+v", c)
	}
}

func TestLoginUpdatesLastLogin(t *testing.T) {
	f := newFixture(t, fixtureOpts{})
	f.login("fikri", goodPassword)

	var got *time.Time
	if err := f.pool.QueryRow(context.Background(), "select last_login_at from admin_users where id=$1", f.userID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got == nil || !got.UTC().Equal(f.clk.Now()) {
		t.Fatalf("last_login_at = %v; want %v", got, f.clk.Now())
	}
}

func TestLoginFailuresAreIndistinguishable(t *testing.T) {
	f := newFixture(t, fixtureOpts{})

	wrong := f.login("fikri", "wrong-password-value")
	unknown := f.login("nosuchuser", "wrong-password-value")

	for name, rec := range map[string]*httptest.ResponseRecorder{"wrong password": wrong, "unknown user": unknown} {
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: status = %d; want 401", name, rec.Code)
		}
		if e := decode[errBody](t, rec); e.Error.Code != "INVALID_CREDENTIALS" {
			t.Fatalf("%s: code = %q", name, e.Error.Code)
		}
		if len(rec.Result().Cookies()) != 0 {
			t.Fatalf("%s: cookie set on failed login", name)
		}
	}
	if wrong.Body.String() != unknown.Body.String() {
		t.Fatalf("bodies differ:\n%s\n%s", wrong.Body.String(), unknown.Body.String())
	}
}

func TestLoginRejectsMalformedRequests(t *testing.T) {
	f := newFixture(t, fixtureOpts{})
	cases := []struct {
		name string
		r    req
		want int
	}{
		{"invalid json", req{method: "POST", path: loginPath, body: "{nope", origin: origin}, 400},
		{"unknown field", req{method: "POST", path: loginPath, body: `{"username":"fikri","password":"x","admin":true}`, origin: origin}, 400},
		{"missing password", req{method: "POST", path: loginPath, body: `{"username":"fikri"}`, origin: origin}, 400},
		{"missing username", req{method: "POST", path: loginPath, body: `{"password":"abc"}`, origin: origin}, 400},
		{"wrong content type", req{method: "POST", path: loginPath, body: loginBody("fikri", goodPassword), contentType: "text/plain", origin: origin}, 415},
		{"oversized body", req{method: "POST", path: loginPath, body: `{"username":"fikri","password":"` + strings.Repeat("a", 10000) + `"}`, origin: origin}, 413},
		{"get not allowed", req{method: "GET", path: loginPath}, 405},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rec := f.do(tc.r); rec.Code != tc.want {
				t.Fatalf("status = %d; want %d (%s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestLoginChecksOrigin(t *testing.T) {
	f := newFixture(t, fixtureOpts{})

	bad := f.do(req{method: "POST", path: loginPath, body: loginBody("fikri", goodPassword), origin: "http://evil.test"})
	if bad.Code != http.StatusForbidden {
		t.Fatalf("cross-origin login = %d; want 403", bad.Code)
	}
	if len(bad.Result().Cookies()) != 0 {
		t.Fatal("cookie set for cross-origin login")
	}

	none := f.do(req{method: "POST", path: loginPath, body: loginBody("fikri", goodPassword)})
	if none.Code != http.StatusOK {
		t.Fatalf("login without Origin (non-browser client) = %d; want 200", none.Code)
	}
}

func TestRateLimitBlocksEvenCorrectPasswordThenRecovers(t *testing.T) {
	f := newFixture(t, fixtureOpts{})
	for i := 0; i < 5; i++ {
		f.login("fikri", "wrong-password-value")
	}

	rec := f.login("fikri", goodPassword)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d; want 429", rec.Code)
	}
	if ra := rec.Header().Get("Retry-After"); ra != "60" {
		t.Fatalf("Retry-After = %q; want 60", ra)
	}
	if e := decode[errBody](t, rec); e.Error.Code != "RATE_LIMITED" {
		t.Fatalf("code = %q", e.Error.Code)
	}

	f.clk.Advance(time.Minute)
	if rec := f.login("fikri", goodPassword); rec.Code != http.StatusOK {
		t.Fatalf("login after lockout = %d; want 200", rec.Code)
	}
}

func TestSuccessfulLoginResetsFailureCount(t *testing.T) {
	f := newFixture(t, fixtureOpts{})
	for i := 0; i < 4; i++ {
		f.login("fikri", "wrong-password-value")
	}
	f.login("fikri", goodPassword)
	for i := 0; i < 4; i++ {
		f.login("fikri", "wrong-password-value")
	}
	if rec := f.login("fikri", goodPassword); rec.Code != http.StatusOK {
		t.Fatalf("login = %d; failures before a success must not accumulate", rec.Code)
	}
}

func TestClientIPUsesForwardedHeaderOnlyFromLoopbackProxy(t *testing.T) {
	f := newFixture(t, fixtureOpts{})

	// Spoofed XFF from a public peer is ignored: all five count against 203.0.113.7.
	for i := 0; i < 5; i++ {
		f.do(req{method: "POST", path: loginPath, body: loginBody("fikri", "wrong-password-value"),
			remoteAddr: "203.0.113.7:1000", xff: fmt.Sprintf("198.51.100.%d", i+1)})
	}
	rec := f.do(req{method: "POST", path: loginPath, body: loginBody("fikri", goodPassword),
		remoteAddr: "203.0.113.7:1000", xff: "198.51.100.99"})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("spoofed X-Forwarded-For evaded the limiter: status %d", rec.Code)
	}
}

func TestClientIPBehindLocalProxyUsesRightmostForwardedAddress(t *testing.T) {
	f := newFixture(t, fixtureOpts{})

	// Peer is the local reverse proxy; its appended (rightmost) entry is the real client.
	for i := 0; i < 5; i++ {
		f.do(req{method: "POST", path: loginPath, body: loginBody("fikri", "wrong-password-value"),
			remoteAddr: "127.0.0.1:1000", xff: "6.6.6.6, 198.51.100.50"})
	}
	blocked := f.do(req{method: "POST", path: loginPath, body: loginBody("fikri", goodPassword),
		remoteAddr: "127.0.0.1:1000", xff: "7.7.7.7, 198.51.100.50"})
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("real client behind proxy not rate limited: %d", blocked.Code)
	}

	other := f.do(req{method: "POST", path: loginPath, body: loginBody("fikri", goodPassword),
		remoteAddr: "127.0.0.1:1000", xff: "198.51.100.51"})
	if other.Code != http.StatusOK {
		t.Fatalf("a different real client was blocked: %d", other.Code)
	}
}

func TestMe(t *testing.T) {
	f := newFixture(t, fixtureOpts{})
	cookie, csrf := f.loggedIn()

	rec := f.do(req{method: "GET", path: mePath, cookie: cookie})
	if rec.Code != http.StatusOK {
		t.Fatalf("me = %d", rec.Code)
	}
	if got := decode[authData](t, rec).Data; got.Username != "fikri" || got.CSRFToken != csrf {
		t.Fatalf("me body = %+v", got)
	}
	if !strings.Contains(rec.Header().Get("Cache-Control"), "no-store") {
		t.Fatal("me must not be cacheable")
	}

	for name, c := range map[string]*http.Cookie{
		"no cookie":      nil,
		"garbage cookie": {Name: "admin_session", Value: "garbage"},
		"unknown token":  {Name: "admin_session", Value: strings.Repeat("A", 43)},
	} {
		rec := f.do(req{method: "GET", path: mePath, cookie: c})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: me = %d; want 401", name, rec.Code)
		}
		if e := decode[errBody](t, rec); e.Error.Code != "UNAUTHENTICATED" {
			t.Fatalf("%s: code = %q", name, e.Error.Code)
		}
	}
}

func TestSessionExpiresAfterIdleTimeout(t *testing.T) {
	f := newFixture(t, fixtureOpts{})
	cookie, _ := f.loggedIn()

	f.clk.Advance(31 * time.Minute)
	if rec := f.do(req{method: "GET", path: mePath, cookie: cookie}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("me after idle timeout = %d; want 401", rec.Code)
	}
}

func TestLogout(t *testing.T) {
	f := newFixture(t, fixtureOpts{})
	cookie, csrf := f.loggedIn()

	t.Run("requires CSRF token", func(t *testing.T) {
		rec := f.do(req{method: "POST", path: logoutPath, cookie: cookie, origin: origin})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("logout without CSRF = %d; want 403", rec.Code)
		}
		rec = f.do(req{method: "POST", path: logoutPath, cookie: cookie, origin: origin, csrf: strings.Repeat("x", 43)})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("logout with wrong CSRF = %d; want 403", rec.Code)
		}
		if rec := f.do(req{method: "GET", path: mePath, cookie: cookie}); rec.Code != http.StatusOK {
			t.Fatal("session was destroyed by a rejected logout")
		}
	})

	t.Run("rejects foreign origin", func(t *testing.T) {
		rec := f.do(req{method: "POST", path: logoutPath, cookie: cookie, origin: "http://evil.test", csrf: csrf})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("cross-origin logout = %d; want 403", rec.Code)
		}
	})

	t.Run("succeeds and clears the cookie", func(t *testing.T) {
		rec := f.do(req{method: "POST", path: logoutPath, cookie: cookie, origin: origin, csrf: csrf})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("logout = %d; want 204", rec.Code)
		}
		cleared := sessionCookie(t, rec)
		if cleared.MaxAge >= 0 || cleared.Value != "" {
			t.Fatalf("cookie not cleared: %+v", cleared)
		}
		if rec := f.do(req{method: "GET", path: mePath, cookie: cookie}); rec.Code != http.StatusUnauthorized {
			t.Fatalf("me after logout = %d; want 401", rec.Code)
		}
	})

	t.Run("is idempotent without a session", func(t *testing.T) {
		rec := f.do(req{method: "POST", path: logoutPath, origin: origin})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("logout without session = %d; want 204", rec.Code)
		}
	})
}

func (f *fixture) protected() http.Handler {
	return f.auth.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, ok := httpapi.AdminSessionFromContext(r.Context())
		if !ok {
			http.Error(w, "no session in context", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("hello " + s.Username))
	}))
}

func (f *fixture) doProtected(r req) *httptest.ResponseRecorder {
	f.t.Helper()
	hr := httptest.NewRequest(r.method, "/api/v1/admin/probe", strings.NewReader(""))
	hr.Host = host
	hr.RemoteAddr = "203.0.113.7:1"
	if r.origin != "" {
		hr.Header.Set("Origin", r.origin)
	}
	if r.csrf != "" {
		hr.Header.Set("X-CSRF-Token", r.csrf)
	}
	if r.cookie != nil {
		hr.AddCookie(r.cookie)
	}
	rec := httptest.NewRecorder()
	f.protected().ServeHTTP(rec, hr)
	return rec
}

func TestRequireMiddleware(t *testing.T) {
	f := newFixture(t, fixtureOpts{})
	cookie, csrf := f.loggedIn()

	t.Run("safe method needs a session only", func(t *testing.T) {
		if rec := f.doProtected(req{method: "GET", cookie: cookie}); rec.Code != 200 || rec.Body.String() != "hello fikri" {
			t.Fatalf("GET = %d %q", rec.Code, rec.Body.String())
		}
		if rec := f.doProtected(req{method: "GET"}); rec.Code != http.StatusUnauthorized {
			t.Fatalf("GET without session = %d; want 401", rec.Code)
		}
	})

	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		t.Run(method+" needs CSRF token and same origin", func(t *testing.T) {
			if rec := f.doProtected(req{method: method, cookie: cookie, origin: origin}); rec.Code != http.StatusForbidden {
				t.Fatalf("no CSRF token = %d; want 403", rec.Code)
			}
			if rec := f.doProtected(req{method: method, cookie: cookie, origin: origin, csrf: "wrong"}); rec.Code != http.StatusForbidden {
				t.Fatalf("wrong CSRF token = %d; want 403", rec.Code)
			}
			if rec := f.doProtected(req{method: method, cookie: cookie, origin: "http://evil.test", csrf: csrf}); rec.Code != http.StatusForbidden {
				t.Fatalf("foreign origin with valid token = %d; want 403", rec.Code)
			}
			if rec := f.doProtected(req{method: method, cookie: cookie, csrf: csrf}); rec.Code != http.StatusForbidden {
				t.Fatalf("missing Origin while a cookie is sent = %d; want 403", rec.Code)
			}
			if rec := f.doProtected(req{method: method, origin: origin, csrf: csrf}); rec.Code != http.StatusUnauthorized {
				t.Fatalf("no session = %d; want 401", rec.Code)
			}
			if rec := f.doProtected(req{method: method, cookie: cookie, origin: origin, csrf: csrf}); rec.Code != http.StatusOK {
				t.Fatalf("valid request = %d; want 200", rec.Code)
			}
		})
	}
}

func TestLoginRehashesWeakStoredHash(t *testing.T) {
	weak := mustHasher(t, cheapParams())
	p := cheapParams()
	p.Iterations = 2
	f := newFixture(t, fixtureOpts{storedHasher: weak, loginHasher: mustHasher(t, p)})

	if rec := f.login("fikri", goodPassword); rec.Code != http.StatusOK {
		t.Fatalf("login = %d", rec.Code)
	}
	var stored string
	if err := f.pool.QueryRow(context.Background(), "select password_hash from admin_users where id=$1", f.userID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored, "$argon2id$v=19$m=64,t=2,p=1$") {
		t.Fatalf("hash not upgraded: %.30s", stored)
	}
	if rec := f.login("fikri", goodPassword); rec.Code != http.StatusOK {
		t.Fatalf("login with upgraded hash = %d", rec.Code)
	}
}

func TestLoginReturns503WhenHashingCapacityIsExhausted(t *testing.T) {
	gate := ratelimit.NewGate(1)
	if err := gate.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	f := newFixture(t, fixtureOpts{gate: gate, hashWait: 50 * time.Millisecond})

	rec := f.login("fikri", goodPassword)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d; want 503", rec.Code)
	}
	if e := decode[errBody](t, rec); e.Error.Code != "SERVICE_BUSY" {
		t.Fatalf("code = %q", e.Error.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("503 should carry Retry-After")
	}
}

func TestAuditEvents(t *testing.T) {
	f := newFixture(t, fixtureOpts{})

	f.login("fikri", "wrong-password-value")
	f.login("Hunter2 secret pass!", "whatever")
	cookie, csrf := f.loggedIn()
	f.do(req{method: "POST", path: logoutPath, cookie: cookie, origin: origin, csrf: csrf})
	for i := 0; i < 5; i++ {
		f.login("fikri", "wrong-password-value")
	}
	f.login("fikri", goodPassword) // blocked

	events := f.audit.all()
	find := func(action, outcome string) []httpapi.AuditEvent {
		var out []httpapi.AuditEvent
		for _, e := range events {
			if e.Action == action && e.Outcome == outcome {
				out = append(out, e)
			}
		}
		return out
	}

	if got := find("auth.login", "success"); len(got) != 1 || got[0].UserID == nil || *got[0].UserID != f.userID || got[0].IP != "203.0.113.7" {
		t.Fatalf("login success events = %+v", got)
	}
	failures := find("auth.login", "failure")
	if len(failures) < 2 || failures[0].Detail["username"] != "fikri" {
		t.Fatalf("login failure events = %+v", failures)
	}
	if got := find("auth.logout", "success"); len(got) != 1 {
		t.Fatalf("logout events = %+v", got)
	}
	if got := find("auth.login_blocked", "failure"); len(got) != 1 {
		t.Fatalf("blocked events = %+v", got)
	}

	for _, e := range events {
		for k, v := range e.Detail {
			if strings.Contains(strings.ToLower(v), "hunter2") || strings.Contains(v, goodPassword) || strings.Contains(v, "wrong-password") {
				t.Fatalf("audit detail %q leaks input: %q", k, v)
			}
		}
	}
}
