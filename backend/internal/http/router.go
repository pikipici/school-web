package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type errorEnvelope struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(requestIDHeader)
	router.Use(middleware.Recoverer)

	router.Get("/healthz", healthz)
	router.Get("/api/v1/public/summary", publicSummary)
	router.NotFound(notFound)

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

func publicSummary(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"totalGroups":  0,
			"manualUpdate": true,
			"updatedAt":    nil,
		},
	})
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
