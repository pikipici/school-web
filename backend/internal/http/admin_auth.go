package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/audit"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/password"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/ratelimit"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/session"
	"github.com/pikipici/informatika-monitoring/backend/internal/auth/users"
)

const (
	cookieName       = "admin_session"
	secureCookieName = "__Host-admin_session"
	maxLoginBody     = 4096
	defaultHashWait  = 5 * time.Second
	csrfHeader       = "X-CSRF-Token"
)

// validUsername matches the database constraint. Only names in this shape are
// written to the audit log, so a password typed into the username field can
// never be persisted.
var validUsername = regexp.MustCompile(`^[a-z0-9._-]{3,32}$`)

// AuditEvent and AuditRecorder are the audit package's types; handlers report to them.
type (
	AuditEvent    = audit.Event
	AuditRecorder = audit.Recorder
)

// AdminAuthConfig wires the admin authentication handlers.
type AdminAuthConfig struct {
	Users    *users.Store
	Sessions *session.Service
	Limiter  *ratelimit.Limiter
	Hasher   *password.Hasher
	// Gate caps concurrent password hashing; HashWait is how long a login
	// waits for a slot before answering 503.
	Gate     *ratelimit.Gate
	HashWait time.Duration
	// SecureCookie must be true whenever the site is served over HTTPS.
	SecureCookie bool
	Audit        AuditRecorder // optional
}

// AdminAuth serves login, logout, and session identity, and protects routes.
type AdminAuth struct {
	cfg  AdminAuthConfig
	name string
}

// NewAdminAuth returns an AdminAuth.
func NewAdminAuth(cfg AdminAuthConfig) *AdminAuth {
	if cfg.HashWait <= 0 {
		cfg.HashWait = defaultHashWait
	}
	name := cookieName
	if cfg.SecureCookie {
		name = secureCookieName
	}
	return &AdminAuth{cfg: cfg, name: name}
}

type sessionKey struct{}

// AdminSessionFromContext returns the session attached by Require.
func AdminSessionFromContext(ctx context.Context) (*session.Session, bool) {
	s, ok := ctx.Value(sessionKey{}).(*session.Session)
	return s, ok
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Data authData `json:"data"`
}

type authData struct {
	Username  string `json:"username"`
	CSRFToken string `json:"csrfToken"`
	ExpiresAt string `json:"expiresAt"`
}

func (a *AdminAuth) login(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if !a.originAllowed(r) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Permintaan ditolak.")
		return
	}
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Gunakan Content-Type application/json.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var in loginRequest
	if err := dec.Decode(&in); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Permintaan terlalu besar.")
			return
		}
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Permintaan tidak valid.")
		return
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Permintaan tidak valid.")
		return
	}
	username := strings.ToLower(strings.TrimSpace(in.Username))
	if username == "" || in.Password == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Nama pengguna dan kata sandi wajib diisi.")
		return
	}

	ip := clientIP(r)
	detail := usernameDetail(username)

	if ok, retry := a.cfg.Limiter.Check(ip, username); !ok {
		a.audit(r, AuditEvent{Action: "auth.login_blocked", Outcome: "failure", IP: ip, Detail: detail})
		w.Header().Set("Retry-After", strconv.Itoa(int(retry/time.Second)))
		writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Terlalu banyak percobaan. Coba lagi nanti.")
		return
	}

	waitCtx, cancel := context.WithTimeout(r.Context(), a.cfg.HashWait)
	defer cancel()
	if err := a.cfg.Gate.Acquire(waitCtx); err != nil {
		w.Header().Set("Retry-After", "5")
		writeError(w, http.StatusServiceUnavailable, "SERVICE_BUSY", "Server sedang sibuk. Coba lagi sebentar.")
		return
	}
	defer a.cfg.Gate.Release()

	user, err := a.cfg.Users.FindByUsername(r.Context(), username)
	var match, rehash bool
	switch {
	case errors.Is(err, users.ErrNotFound):
		a.cfg.Hasher.DummyVerify(in.Password) // equalize timing with the real path
	case err != nil:
		slog.Error("login: user lookup failed", "error", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Terjadi kesalahan pada server.")
		return
	default:
		var verr error
		match, rehash, verr = a.cfg.Hasher.Verify(in.Password, user.PasswordHash)
		if verr != nil && !errors.Is(verr, password.ErrTooLong) {
			slog.Error("login: stored hash unusable", "user_id", user.ID, "error", verr)
		}
	}

	if !match {
		a.cfg.Limiter.RecordFailure(ip, username)
		a.audit(r, AuditEvent{Action: "auth.login", Outcome: "failure", IP: ip, Detail: detail})
		writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Nama pengguna atau kata sandi salah.")
		return
	}

	a.cfg.Limiter.RecordSuccess(ip, username)

	if rehash {
		if h, err := a.cfg.Hasher.Hash(in.Password); err == nil {
			if err := a.cfg.Users.SetPasswordHash(r.Context(), user.ID, h); err != nil {
				slog.Error("login: password rehash not saved", "user_id", user.ID, "error", err)
			}
		}
	}
	if err := a.cfg.Users.MarkLogin(r.Context(), user.ID); err != nil {
		slog.Error("login: last login not saved", "user_id", user.ID, "error", err)
	}

	issued, err := a.cfg.Sessions.Create(r.Context(), user.ID, ip, r.UserAgent())
	if err != nil {
		slog.Error("login: session create failed", "user_id", user.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Terjadi kesalahan pada server.")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     a.name,
		Value:    issued.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.SecureCookie,
		SameSite: http.SameSiteStrictMode,
	})
	uid := user.ID
	a.audit(r, AuditEvent{Action: "auth.login", Outcome: "success", UserID: &uid, IP: ip})
	writeJSON(w, http.StatusOK, authResponse{Data: authData{
		Username:  user.Username,
		CSRFToken: issued.CSRFToken,
		ExpiresAt: issued.ExpiresAt.UTC().Format(time.RFC3339),
	}})
}

func (a *AdminAuth) me(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	s, ok := a.authenticate(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Silakan masuk terlebih dahulu.")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Data: authData{
		Username:  s.Username,
		CSRFToken: s.CSRFToken,
		ExpiresAt: s.ExpiresAt.UTC().Format(time.RFC3339),
	}})
}

func (a *AdminAuth) logout(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if s, ok := a.authenticate(r); ok {
		if !a.originAllowed(r) || !csrfMatches(r, s) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Permintaan ditolak.")
			return
		}
		if c, err := r.Cookie(a.name); err == nil {
			if err := a.cfg.Sessions.Revoke(r.Context(), c.Value); err != nil {
				slog.Error("logout: revoke failed", "error", err)
			}
		}
		uid := s.UserID
		a.audit(r, AuditEvent{Action: "auth.logout", Outcome: "success", UserID: &uid, IP: clientIP(r)})
	}
	a.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// Require protects a handler: it needs a valid session, and unsafe methods
// also need a matching CSRF token and a same-origin request.
func (a *AdminAuth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		noStore(w)
		s, ok := a.authenticate(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Silakan masuk terlebih dahulu.")
			return
		}
		if !safeMethod(r.Method) && (!a.originAllowed(r) || !csrfMatches(r, s)) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Permintaan ditolak.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionKey{}, s)))
	})
}

func (a *AdminAuth) authenticate(r *http.Request) (*session.Session, bool) {
	c, err := r.Cookie(a.name)
	if err != nil {
		return nil, false
	}
	s, err := a.cfg.Sessions.Validate(r.Context(), c.Value)
	if err != nil {
		if !errors.Is(err, session.ErrInvalid) {
			slog.Error("session validation failed", "error", err)
		}
		return nil, false
	}
	return s, true
}

// originAllowed blocks cross-site requests. A browser always sends Origin on
// unsafe requests, so a present Origin must match this host. A request without
// Origin is accepted only if it carries no session cookie: non-browser clients
// have no ambient credentials, so there is nothing to forge.
func (a *AdminAuth) originAllowed(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		_, err := r.Cookie(a.name)
		return err != nil
	}
	u, err := url.Parse(o)
	if err != nil || u.Host == "" {
		return false
	}
	if a.cfg.SecureCookie && u.Scheme != "https" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func (a *AdminAuth) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.cfg.SecureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (a *AdminAuth) audit(r *http.Request, e AuditEvent) {
	if a.cfg.Audit != nil {
		a.cfg.Audit.Record(r.Context(), e)
	}
}

func csrfMatches(r *http.Request, s *session.Session) bool {
	got := r.Header.Get(csrfHeader)
	return got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(s.CSRFToken)) == 1
}

func safeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

func usernameDetail(normalized string) map[string]string {
	if validUsername.MatchString(normalized) {
		return map[string]string{"username": normalized}
	}
	return nil
}

// clientIP returns the caller's address. X-Forwarded-For is trusted only when
// the TCP peer is the local reverse proxy (loopback), and then only its
// rightmost entry, which is the one that proxy appended.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return ""
	}
	if peer.IsLoopback() {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if client, err := netip.ParseAddr(strings.TrimSpace(parts[len(parts)-1])); err == nil {
				return client.String()
			}
		}
	}
	return peer.String()
}

func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorEnvelope{Error: errorDetail{Code: code, Message: message}})
}
