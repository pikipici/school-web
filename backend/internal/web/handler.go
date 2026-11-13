package web

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var hashedAsset = regexp.MustCompile(`(?i)(?:^|[._-])[0-9a-f]{8,}(?:[._-]|$)`)

var portableMIMETypes = map[string]string{
	".js":          "text/javascript; charset=utf-8",
	".css":         "text/css; charset=utf-8",
	".woff":        "font/woff",
	".woff2":       "font/woff2",
	".wasm":        "application/wasm",
	".ico":         "image/x-icon",
	".gif":         "image/gif",
	".avif":        "image/avif",
	".webmanifest": "application/manifest+json",
	".map":         "application/json",
	".txt":         "text/plain; charset=utf-8",
	".xml":         "application/xml",
}

// maxHTMLBytes bounds how much of an HTML page is read into memory to compute
// its policy. Exported pages are a few tens of KiB.
const maxHTMLBytes = 8 << 20

type handler struct {
	publicDir string
	enabled   bool
	csp       CSPMode
	policies  *policyCache
}

// New returns a handler for a Next.js static export rooted at publicDir that
// enforces a hash-based Content-Security-Policy on HTML pages.
func New(publicDir string) http.Handler {
	return NewWithCSP(publicDir, CSPEnforce)
}

// NewWithCSP is New with an explicit policy mode.
func NewWithCSP(publicDir string, mode CSPMode) http.Handler {
	h := &handler{csp: mode, policies: &policyCache{}}
	if strings.TrimSpace(publicDir) == "" {
		return h
	}
	root, err := filepath.Abs(filepath.Clean(publicDir))
	if err != nil {
		return h
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return h
	}
	h.publicDir, h.enabled = root, true
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	requestPath, ok := safePath(r)
	if !ok {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if requestPath == "api/v1" || strings.HasPrefix(requestPath, "api/v1/") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"Endpoint tidak ditemukan."}}` + "\n"))
		}
		return
	}
	file, info, name, status := h.open(requestPath)
	if file == nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	setHeaders(w, requestPath, name)

	var content io.ReadSeeker = file
	if strings.EqualFold(filepath.Ext(name), ".html") && h.csp != CSPOff {
		body, err := io.ReadAll(io.LimitReader(file, maxHTMLBytes+1))
		if err != nil || len(body) > maxHTMLBytes {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		policy := h.policies.get(cacheKey{name: name, size: info.Size(), modTime: info.ModTime()}, body)
		if h.csp == CSPReportOnly {
			w.Header().Set("Content-Security-Policy-Report-Only", policy)
		} else {
			w.Header().Set("Content-Security-Policy", policy)
		}
		content = bytes.NewReader(body)
	}

	if status == http.StatusNotFound {
		w.WriteHeader(status)
		if r.Method == http.MethodGet {
			_, _ = io.Copy(w, content)
		}
		return
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), content)
}

func safePath(r *http.Request) (string, bool) {
	value := r.URL.EscapedPath()
	for range 8 {
		decoded, err := url.PathUnescape(value)
		if err != nil {
			return "", false
		}
		if decoded == value {
			break
		}
		value = decoded
	}
	if strings.Contains(value, "%") || strings.Contains(value, `\`) || strings.IndexByte(value, 0) >= 0 {
		return "", false
	}
	for _, part := range strings.Split(value, "/") {
		if part == ".." || part == "." {
			return "", false
		}
	}
	return strings.TrimPrefix(value, "/"), true
}

func (h *handler) open(requestPath string) (*os.File, os.FileInfo, string, int) {
	if !h.enabled {
		return nil, nil, "", http.StatusNotFound
	}
	candidates := []string{requestPath}
	if requestPath == "" {
		candidates = []string{"index.html"}
	}
	if requestPath != "" && filepath.Ext(requestPath) == "" {
		candidates = []string{requestPath + ".html", filepath.Join(requestPath, "index.html")}
	}
	for _, candidate := range candidates {
		if file, info, name := h.openCandidate(candidate); file != nil {
			return file, info, name, http.StatusOK
		}
	}
	if file, info, name := h.openCandidate("404.html"); file != nil {
		return file, info, name, http.StatusNotFound
	}
	return nil, nil, "", http.StatusNotFound
}

func (h *handler) openCandidate(candidate string) (*os.File, os.FileInfo, string) {
	name := filepath.Join(h.publicDir, filepath.FromSlash(candidate))
	resolved, err := filepath.EvalSymlinks(name)
	if err != nil || !contained(h.publicDir, resolved) {
		return nil, nil, ""
	}
	file, err := os.Open(resolved)
	if err != nil {
		return nil, nil, ""
	}
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		file.Close()
		return nil, nil, ""
	}
	return file, info, resolved
}

func contained(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func setHeaders(w http.ResponseWriter, requestPath, name string) {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".html" {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		return
	}
	cleanRequest := filepath.ToSlash(filepath.Clean("/" + requestPath))
	if strings.HasPrefix(cleanRequest, "/_next/static/") && hashedAsset.MatchString(filepath.Base(name)) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	contentType := portableMIMETypes[ext]
	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
}
