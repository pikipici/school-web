package web_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pikipici/informatika-monitoring/backend/internal/web"
)

func TestStaticExportHandler(t *testing.T) {
	publicDir := fixturePublicDir(t)
	handler := web.New(publicDir)

	tests := []struct {
		name        string
		method      string
		target      string
		status      int
		body        string
		contentType string
		cache       string
	}{
		{name: "root index", method: http.MethodGet, target: "/", status: 200, body: "HOME", contentType: "text/html", cache: "no-store"},
		{name: "extensionless html", method: http.MethodGet, target: "/monitoring", status: 200, body: "MONITORING FILE", contentType: "text/html", cache: "no-store"},
		{name: "extensionless directory index fallback", method: http.MethodGet, target: "/reports", status: 200, body: "REPORT INDEX", contentType: "text/html", cache: "no-store"},
		{name: "missing frontend route", method: http.MethodGet, target: "/does-not-exist", status: 404, body: "CUSTOM NOT FOUND", contentType: "text/html", cache: "no-store"},
		{name: "hashed static asset", method: http.MethodGet, target: "/_next/static/app.0123456789abcdef.js", status: 200, body: "asset", contentType: "text/javascript", cache: "public, max-age=31536000, immutable"},
		{name: "unhashed static asset", method: http.MethodGet, target: "/logo.svg", status: 200, body: "logo", contentType: "image/svg+xml", cache: "no-cache"},
		{name: "head", method: http.MethodHead, target: "/", status: 200, body: "", contentType: "text/html", cache: "no-store"},
		{name: "post rejected", method: http.MethodPost, target: "/", status: 405, body: "", contentType: "", cache: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(tt.method, tt.target, nil))
			if recorder.Code != tt.status {
				t.Fatalf("status = %d, want %d; body=%q", recorder.Code, tt.status, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), tt.body) {
				t.Fatalf("body = %q, want substring %q", recorder.Body.String(), tt.body)
			}
			if got := recorder.Header().Get("Content-Type"); tt.contentType != "" && !strings.Contains(got, tt.contentType) {
				t.Fatalf("Content-Type = %q, want containing %q", got, tt.contentType)
			}
			if got := recorder.Header().Get("Cache-Control"); got != tt.cache {
				t.Fatalf("Cache-Control = %q, want %q", got, tt.cache)
			}
		})
	}
}

func TestStaticExportHandlerRejectsTraversal(t *testing.T) {
	handler := web.New(fixturePublicDir(t))
	for _, target := range []string{"/../secret.txt", "/%2e%2e/secret.txt", "/%2e%2e%2fsecret.txt", "/..%2fsecret.txt", "/%252e%252e%252fsecret.txt"} {
		t.Run(target, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%q", recorder.Code, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "SECRET") {
				t.Fatal("traversal exposed a file outside public directory")
			}
		})
	}
}

func TestStaticExportHandlerNeverServesCanonicalAPINamespace(t *testing.T) {
	publicDir := fixturePublicDir(t)
	for _, name := range []string{"api/v1.html", "api/v1/index.html", "api/v1/secret.html"} {
		path := filepath.Join(publicDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("STATIC API LEAK"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	handler := web.New(publicDir)
	for _, target := range []string{
		"/api/v1", "/api/v1/secret", "/api%2fv1", "/api%2fv1%2fsecret",
		"/api%252fv1", "/api%252fv1%252fsecret", "/api%25252fv1%25252fsecret",
	} {
		t.Run(target, func(t *testing.T) {
			r := httptest.NewRecorder()
			handler.ServeHTTP(r, httptest.NewRequest(http.MethodGet, target, nil))
			if r.Code != http.StatusNotFound || r.Header().Get("Content-Type") != "application/json; charset=utf-8" || strings.Contains(r.Body.String(), "STATIC API LEAK") {
				t.Fatalf("response = (%d, %q, %q), want JSON non-static 404", r.Code, r.Header().Get("Content-Type"), r.Body.String())
			}
		})
	}
}

func TestStaticExportHandlerEncodedAPINamespaceReturnsStandardNotFoundJSON(t *testing.T) {
	handler := web.New(fixturePublicDir(t))
	for _, target := range []string{
		"/api%2fv1", "/api%2fv1%2fmissing",
		"/api%252fv1", "/api%252fv1%252fmissing",
		"/api%25252fv1", "/api%25252fv1%25252fmissing",
	} {
		t.Run(target, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))

			var response struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response body %q: %v", recorder.Body.String(), err)
			}
			if recorder.Code != http.StatusNotFound || response.Error.Code != "NOT_FOUND" || response.Error.Message != "Endpoint tidak ditemukan." {
				t.Fatalf("response = (%d, %#v), want standard NOT_FOUND envelope", recorder.Code, response.Error)
			}
		})
	}
}

func TestStaticExportHandlerRejectsSymlinkEscapes(t *testing.T) {
	root := t.TempDir()
	publicDir, outsideDir := filepath.Join(root, "public"), filepath.Join(root, "outside")
	if err := os.MkdirAll(publicDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(publicDir, "404.html"), []byte("CUSTOM NOT FOUND"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("SECRET"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outsideDir, "secret.txt"), filepath.Join(publicDir, "file-link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(publicDir, "dir-link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	handler := web.New(publicDir)
	for _, target := range []string{"/file-link.txt", "/dir-link/secret.txt"} {
		t.Run(target, func(t *testing.T) {
			r := httptest.NewRecorder()
			handler.ServeHTTP(r, httptest.NewRequest(http.MethodGet, target, nil))
			if r.Code != 404 || r.Body.String() != "CUSTOM NOT FOUND" || strings.Contains(r.Body.String(), "SECRET") {
				t.Fatalf("response = (%d, %q), want custom 404", r.Code, r.Body.String())
			}
		})
	}
}

func TestStaticExportHandlerStandardFileSemantics(t *testing.T) {
	handler := web.New(fixturePublicDir(t))
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/logo.svg", nil))
	if get.Header().Get("Content-Length") != "15" || get.Header().Get("Last-Modified") == "" || get.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatalf("GET metadata = %#v", get.Header())
	}
	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/logo.svg", nil))
	if head.Code != 200 || head.Body.Len() != 0 || head.Header().Get("Content-Length") != "15" || head.Header().Get("Last-Modified") == "" {
		t.Fatalf("HEAD = (%d, %q, %#v)", head.Code, head.Body.String(), head.Header())
	}
	req := httptest.NewRequest(http.MethodGet, "/logo.svg", nil)
	req.Header.Set("Range", "bytes=0-4")
	partial := httptest.NewRecorder()
	handler.ServeHTTP(partial, req)
	if partial.Code != 206 || partial.Body.String() != "<svg>" || partial.Header().Get("Content-Range") != "bytes 0-4/15" {
		t.Fatalf("Range = (%d, %q, %#v)", partial.Code, partial.Body.String(), partial.Header())
	}
}

func TestStaticExportHandlerCacheScopeAndMIMETypes(t *testing.T) {
	publicDir := fixturePublicDir(t)
	files := map[string]string{"outside.0123456789abcdef.js": "x", "style.css": "x", "font.woff2": "x", "app.webmanifest": "x", "bundle.map": "x", "photo.avif": "x", "_next/static/plain.js": "x", "_next/static/font.0123456789abcdef.woff": "x", "data.txt": "text", "icon.ico": "icon", "photo.gif": "gif", "module.wasm": "wasm", "feed.xml": "xml"}
	for name, body := range files {
		path := filepath.Join(publicDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	handler := web.New(publicDir)
	checks := []struct{ path, cache, mime string }{{"/outside.0123456789abcdef.js", "no-cache", "text/javascript; charset=utf-8"}, {"/style.css", "no-cache", "text/css; charset=utf-8"}, {"/font.woff2", "no-cache", "font/woff2"}, {"/app.webmanifest", "no-cache", "application/manifest+json"}, {"/bundle.map", "no-cache", "application/json"}, {"/photo.avif", "no-cache", "image/avif"}, {"/_next/static/plain.js", "no-cache", "text/javascript; charset=utf-8"}, {"/_next/static/font.0123456789abcdef.woff", "public, max-age=31536000, immutable", "font/woff"}, {"/data.txt", "no-cache", "text/plain; charset=utf-8"}, {"/icon.ico", "no-cache", "image/x-icon"}, {"/photo.gif", "no-cache", "image/gif"}, {"/module.wasm", "no-cache", "application/wasm"}, {"/feed.xml", "no-cache", "application/xml"}}
	for _, check := range checks {
		t.Run(check.path, func(t *testing.T) {
			r := httptest.NewRecorder()
			handler.ServeHTTP(r, httptest.NewRequest(http.MethodGet, check.path, nil))
			if r.Code != 200 || r.Header().Get("Cache-Control") != check.cache || r.Header().Get("Content-Type") != check.mime {
				t.Fatalf("response = (%d, cache %q, type %q)", r.Code, r.Header().Get("Cache-Control"), r.Header().Get("Content-Type"))
			}
		})
	}
}

func TestStaticExportHandlerBlankRootDoesNotServeWorkingDirectory(t *testing.T) {
	for _, root := range []string{"", "   "} {
		r := httptest.NewRecorder()
		web.New(root).ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/go.mod", nil))
		body, _ := io.ReadAll(r.Result().Body)
		if r.Code != 404 || strings.Contains(string(body), "module ") {
			t.Fatalf("root %q exposed working directory: (%d, %q)", root, r.Code, body)
		}
	}
}

func fixturePublicDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	publicDir := filepath.Join(root, "public")
	files := map[string]string{
		"index.html": "<html>HOME</html>", "monitoring.html": "<html>MONITORING FILE</html>",
		"monitoring/index.html": "<html>WRONG FALLBACK</html>", "reports/index.html": "<html>REPORT INDEX</html>",
		"404.html": "<html>CUSTOM NOT FOUND</html>", "_next/static/app.0123456789abcdef.js": "asset", "logo.svg": "<svg>logo</svg>",
	}
	for name, body := range files {
		path := filepath.Join(publicDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "secret.txt"), []byte("SECRET"), 0o644); err != nil {
		t.Fatal(err)
	}
	return publicDir
}
