package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// publicGet sends an anonymous GET with optional extra headers.
func (f *fixture) publicGet(path string, headers map[string]string) *httptest.ResponseRecorder {
	f.t.Helper()
	hr := httptest.NewRequest(http.MethodGet, path, nil)
	hr.Host = host
	hr.RemoteAddr = "198.51.100.9:40000"
	for k, v := range headers {
		hr.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, hr)
	return rec
}

func (f *fixture) setStatus(t *testing.T, materialID int64, status string) {
	t.Helper()
	_, err := f.pool.Exec(t.Context(),
		`update materials set status=$2, published_at=coalesce(published_at, now()) where id=$1`, materialID, status)
	if err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) uploadAs(t *testing.T, cookie *http.Cookie, csrf string, materialID int64, name, body string) attachmentData {
	t.Helper()
	rec := f.sendUpload(fmt.Sprintf("/api/v1/admin/materials/%d/attachments", materialID),
		upload{field: "file", name: name, body: strings.NewReader(body), cookie: cookie, csrf: csrf, origin: f.origin})
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body.String())
	}
	var out struct{ Data attachmentData }
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.Data
}

func TestPublicAttachmentsOnlyForPublishedMaterials(t *testing.T) {
	f := newFixture(t, fixtureOpts{attachments: true})
	cookie, csrf := f.loggedIn()
	mid := f.createMaterial(t, cookie, csrf, "Draf")
	a := f.uploadAs(t, cookie, csrf, mid, "rahasia.pdf", testPDF)
	dl := fmt.Sprintf("/api/v1/public/attachments/%d", a.ID)
	list := fmt.Sprintf("/api/v1/public/materials/%d/attachments", mid)

	for _, p := range []string{dl, list} {
		if r := f.publicGet(p, nil); r.Code != http.StatusNotFound || decode[errBody](t, r).Error.Code != "NOT_FOUND" {
			t.Errorf("draft %s = %d %s, want JSON 404", p, r.Code, r.Body.String())
		}
	}

	f.setStatus(t, mid, "published")
	if r := f.publicGet(dl, nil); r.Code != http.StatusOK || r.Body.String() != testPDF {
		t.Fatalf("published download = %d (%d bytes)", r.Code, r.Body.Len())
	}
	r := f.publicGet(list, nil)
	if r.Code != http.StatusOK {
		t.Fatalf("published list = %d %s", r.Code, r.Body.String())
	}
	var listed struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &listed); err != nil || len(listed.Data) != 1 {
		t.Fatalf("list body %s (%v)", r.Body.String(), err)
	}
	item := listed.Data[0]
	if item["id"] != float64(a.ID) || item["name"] != "rahasia.pdf" || item["kind"] != "pdf" || item["size"] != float64(len(testPDF)) {
		t.Errorf("public item = %#v", item)
	}
	for k := range item {
		if k != "id" && k != "name" && k != "kind" && k != "size" {
			t.Errorf("public item exposes %q", k)
		}
	}

	f.setStatus(t, mid, "draft")
	for _, p := range []string{dl, list} {
		if r := f.publicGet(p, nil); r.Code != http.StatusNotFound {
			t.Errorf("after unpublish %s = %d, want 404", p, r.Code)
		}
	}
	for _, p := range []string{"/api/v1/public/attachments/999999", "/api/v1/public/attachments/abc", "/api/v1/public/attachments/0"} {
		if r := f.publicGet(p, nil); r.Code != http.StatusNotFound || !strings.HasPrefix(r.Header().Get("Content-Type"), "application/json") {
			t.Errorf("%s = %d %q, want JSON 404", p, r.Code, r.Header().Get("Content-Type"))
		}
	}
}

func TestPublicAttachmentDownloadHeadersAndRanges(t *testing.T) {
	f := newFixture(t, fixtureOpts{attachments: true})
	cookie, csrf := f.loggedIn()
	mid := f.createMaterial(t, cookie, csrf, "Terbit")
	a := f.uploadAs(t, cookie, csrf, mid, `Materi "Ä".pdf`, testPDF)
	f.setStatus(t, mid, "published")
	dl := fmt.Sprintf("/api/v1/public/attachments/%d", a.ID)

	r := f.publicGet(dl, nil)
	if r.Code != http.StatusOK {
		t.Fatalf("download = %d", r.Code)
	}
	h := r.Header()
	want := map[string]string{
		"Content-Type":            "application/pdf",
		"Content-Disposition":     `attachment; filename="Materi ___.pdf"; filename*=UTF-8''Materi%20%22%C3%84%22.pdf`,
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "default-src 'none'; sandbox",
		"Cache-Control":           "no-cache",
		"Accept-Ranges":           "bytes",
		"Content-Length":          fmt.Sprint(len(testPDF)),
	}
	for k, v := range want {
		if got := h.Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
	etag := h.Get("ETag")
	if len(etag) != 66 || !strings.HasPrefix(etag, `"`) {
		t.Errorf("ETag = %q, want quoted SHA-256 hex", etag)
	}

	part := f.publicGet(dl, map[string]string{"Range": "bytes=0-3"})
	if part.Code != http.StatusPartialContent || part.Body.String() != "%PDF" || part.Header().Get("Content-Range") != fmt.Sprintf("bytes 0-3/%d", len(testPDF)) {
		t.Errorf("range = %d %q %q", part.Code, part.Body.String(), part.Header().Get("Content-Range"))
	}
	if r := f.publicGet(dl, map[string]string{"Range": "bytes=9999-"}); r.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Errorf("unsatisfiable range = %d, want 416", r.Code)
	}
	if r := f.publicGet(dl, map[string]string{"If-None-Match": etag}); r.Code != http.StatusNotModified {
		t.Errorf("If-None-Match = %d, want 304", r.Code)
	}
	if r := f.publicGet(dl, map[string]string{"Range": "bytes=0-3", "If-Range": `"stale"`}); r.Code != http.StatusOK || r.Body.String() != testPDF {
		t.Errorf("stale If-Range = %d, want full 200", r.Code)
	}
}

func TestPublicAttachmentMissingFileIsNotFound(t *testing.T) {
	f := newFixture(t, fixtureOpts{attachments: true})
	cookie, csrf := f.loggedIn()
	mid := f.createMaterial(t, cookie, csrf, "Hilang")
	a := f.uploadAs(t, cookie, csrf, mid, "a.pdf", testPDF)
	f.setStatus(t, mid, "published")
	for _, name := range f.attachFiles(t) {
		if err := os.Remove(filepath.Join(f.attachDir, name)); err != nil {
			t.Fatal(err)
		}
	}
	r := f.publicGet(fmt.Sprintf("/api/v1/public/attachments/%d", a.ID), nil)
	if r.Code != http.StatusNotFound || decode[errBody](t, r).Error.Code != "NOT_FOUND" {
		t.Errorf("missing file = %d %s, want JSON 404", r.Code, r.Body.String())
	}
}
