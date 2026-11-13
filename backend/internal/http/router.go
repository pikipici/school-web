package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pikipici/informatika-monitoring/backend/internal/health"
	"github.com/pikipici/informatika-monitoring/backend/internal/web"
)

type errorEnvelope struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewRouter() http.Handler {
	return newRouter(Deps{})
}

func NewRouterWithPinger(pinger health.Pinger) http.Handler {
	return newRouter(Deps{Pinger: pinger})
}

func NewRouterWithPublicDir(publicDir string) http.Handler {
	return newRouter(Deps{PublicDir: publicDir})
}

func NewRouterWithPingerAndPublicDir(pinger health.Pinger, publicDir string) http.Handler {
	return newRouter(Deps{Pinger: pinger, PublicDir: publicDir})
}

// Deps carries optional collaborators. Unset ones disable their routes.
type Deps struct {
	Pinger    health.Pinger
	PublicDir string
	// Admin enables /api/v1/admin/auth/*.
	Admin *AdminAuth
	// CSPMode controls the policy header on HTML pages (zero value: enforce).
	CSPMode web.CSPMode
}

// NewRouterWithDeps builds the router from Deps.
func NewRouterWithDeps(deps Deps) http.Handler {
	return newRouter(deps)
}

func newRouter(deps Deps) http.Handler {
	pinger, publicDir := deps.Pinger, deps.PublicDir
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(requestIDHeader)
	router.Use(middleware.Recoverer)

	router.Get("/healthz", healthz)
	readiness := health.NewService(pinger)
	router.Get("/readyz", readyz(readiness))
	router.Get("/api/v1/public/summary", publicSummary)
	if deps.Admin != nil {
		router.Route("/api/v1/admin/auth", func(r chi.Router) {
			r.Post("/login", deps.Admin.login)
			r.Post("/logout", deps.Admin.logout)
			r.Get("/me", deps.Admin.me)
			r.NotFound(notFound)
			r.MethodNotAllowed(methodNotAllowed)
		})
	}
	router.HandleFunc("/api/v1", notFound)
	router.HandleFunc("/api/v1/", notFound)
	router.HandleFunc("/api/v1/*", notFound)
	if strings.TrimSpace(publicDir) == "" {
		router.NotFound(notFound)
	} else {
		static := web.NewWithCSP(publicDir, deps.CSPMode)
		router.NotFound(static.ServeHTTP)
	}

	return router
}

func requestIDHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := middleware.GetReqID(r.Context())
		if requestID != "" {
			w.Header().Set("X-Request-ID", requestID)
		}
		next.ServeHTTP(w, r)
	})
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readyz(service health.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		result := service.Readiness(ctx)
		if !result.Ready {
			writeJSON(w, http.StatusServiceUnavailable, result)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func publicSummary(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"totalGroups":  0,
			"manualUpdate": true,
			"updatedAt":    nil,
		},
	})
}

func methodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Metode tidak diizinkan.")
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotFound, errorEnvelope{Error: errorDetail{
		Code:    "NOT_FOUND",
		Message: "Endpoint tidak ditemukan.",
	}})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
