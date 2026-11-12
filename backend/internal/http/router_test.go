package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	httpapi "github.com/pikipici/informatika-monitoring/backend/internal/http"
)

func TestHealthzReturnsSafeJSONAndRequestID(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	httpapi.NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	if got := recorder.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("X-Request-ID is empty")
	}

	var response struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "ok" {
		t.Fatalf("status body = %q, want ok", response.Status)
	}
}

func TestPublicSummaryReturnsHonestZeroState(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/summary", nil)

	httpapi.NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Data struct {
			TotalGroups  int     `json:"totalGroups"`
			ManualUpdate bool    `json:"manualUpdate"`
			UpdatedAt    *string `json:"updatedAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.TotalGroups != 0 {
		t.Fatalf("totalGroups = %d, want 0", response.Data.TotalGroups)
	}
	if !response.Data.ManualUpdate {
		t.Fatal("manualUpdate = false, want true")
	}
	if response.Data.UpdatedAt != nil {
		t.Fatalf("updatedAt = %q, want null", *response.Data.UpdatedAt)
	}
}

func TestRouterServesFrontendWithoutShadowingAPI(t *testing.T) {
	publicDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(publicDir, "index.html"), []byte("FRONTEND HOME"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(publicDir, "404.html"), []byte("FRONTEND NOT FOUND"), 0o644); err != nil {
		t.Fatal(err)
	}
	router := httpapi.NewRouterWithPublicDir(publicDir)

	frontend := httptest.NewRecorder()
	router.ServeHTTP(frontend, httptest.NewRequest(http.MethodGet, "/", nil))
	if frontend.Code != http.StatusOK || !strings.Contains(frontend.Body.String(), "FRONTEND HOME") {
		t.Fatalf("frontend response = (%d, %q), want (200, frontend home)", frontend.Code, frontend.Body.String())
	}

	api := httptest.NewRecorder()
	router.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil))
	if api.Code != http.StatusNotFound {
		t.Fatalf("API status = %d, want 404", api.Code)
	}
	if got := api.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("API Content-Type = %q, want JSON", got)
	}
	if strings.Contains(api.Body.String(), "FRONTEND") {
		t.Fatalf("API returned frontend HTML: %q", api.Body.String())
	}
}

func TestRouterReservesExactAPINamespace(t *testing.T) {
	publicDir := t.TempDir()
	for _, name := range []string{"api/v1.html", "api/v1/index.html", "404.html"} {
		path := filepath.Join(publicDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("FRONTEND"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	router := httpapi.NewRouterWithPublicDir(publicDir)
	for _, target := range []string{"/api/v1", "/api/v1/"} {
		t.Run(target, func(t *testing.T) {
			r := httptest.NewRecorder()
			router.ServeHTTP(r, httptest.NewRequest(http.MethodGet, target, nil))
			if r.Code != 404 || r.Header().Get("Content-Type") != "application/json; charset=utf-8" || strings.Contains(r.Body.String(), "FRONTEND") {
				t.Fatalf("response = (%d, %q, %q), want JSON 404", r.Code, r.Header().Get("Content-Type"), r.Body.String())
			}
		})
	}
}

func TestRouterReservesEncodedAPINamespace(t *testing.T) {
	publicDir := t.TempDir()
	for _, name := range []string{"api/v1.html", "api/v1/index.html", "api/v1/missing.html", "404.html"} {
		path := filepath.Join(publicDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("FRONTEND API LEAK"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	router := httpapi.NewRouterWithPublicDir(publicDir)
	for _, target := range []string{"/api%2fv1", "/api%2fv1%2fmissing", "/api%252fv1", "/api%252fv1%252fmissing"} {
		t.Run(target, func(t *testing.T) {
			r := httptest.NewRecorder()
			router.ServeHTTP(r, httptest.NewRequest(http.MethodGet, target, nil))
			if r.Code != 404 || r.Header().Get("Content-Type") != "application/json; charset=utf-8" || strings.Contains(r.Body.String(), "FRONTEND API LEAK") {
				t.Fatalf("response = (%d, %q, %q), encoded API reached static content", r.Code, r.Header().Get("Content-Type"), r.Body.String())
			}
		})
	}
}

func TestRouterBlankPublicDirKeepsJSONNotFound(t *testing.T) {
	for _, publicDir := range []string{"", "   "} {
		t.Run(publicDir, func(t *testing.T) {
			r := httptest.NewRecorder()
			httpapi.NewRouterWithPublicDir(publicDir).ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/missing", nil))
			if r.Code != 404 || r.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatalf("response = (%d, %q)", r.Code, r.Header().Get("Content-Type"))
			}
		})
	}
}

func TestUnknownAPIRouteReturnsJSONNotFrontend(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)

	httpapi.NewRouter().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}

	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != "NOT_FOUND" {
		t.Fatalf("error code = %q, want NOT_FOUND", response.Error.Code)
	}
}
