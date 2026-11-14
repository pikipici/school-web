package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

const testPDF = "%PDF-1.7\n1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n2 0 obj << /Type /Pages /Kids [] /Count 0 >> endobj\ntrailer << /Root 1 0 R >>\n%%EOF\n"

type attachmentData struct {
	ID          int64  `json:"id"`
	MaterialID  int64  `json:"materialId"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	CreatedAt   string `json:"createdAt"`
}

type upload struct {
	field, name string
	body        io.Reader
	contentType string // overrides the multipart content type when set
	cookie      *http.Cookie
	csrf        string
	origin      string
}

// sendUpload streams a multipart body through the router without holding the
// whole file in memory.
func (f *fixture) sendUpload(path string, u upload) *httptest.ResponseRecorder {
	f.t.Helper()
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		part, err := mw.CreateFormFile(u.field, u.name)
		if err == nil {
			_, err = io.Copy(part, u.body)
		}
		if err == nil {
			err = mw.Close()
		}
		pw.CloseWithError(err)
	}()
	hr := httptest.NewRequest(http.MethodPost, path, pr)
	hr.Host = host
	hr.RemoteAddr = "203.0.113.7:51234"
	ct := mw.FormDataContentType()
	if u.contentType != "" {
		ct = u.contentType
	}
	hr.Header.Set("Content-Type", ct)
	if u.origin != "" {
		hr.Header.Set("Origin", u.origin)
	}
	if u.csrf != "" {
		hr.Header.Set("X-CSRF-Token", u.csrf)
	}
	if u.cookie != nil {
		hr.AddCookie(u.cookie)
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, hr)
	// Drain whatever the handler did not read so the writer goroutine ends.
	_, _ = io.Copy(io.Discard, pr)
	return rec
}

func (f *fixture) createMaterial(t *testing.T, cookie *http.Cookie, csrf, title string) int64 {
	t.Helper()
	rec := f.do(req{method: http.MethodPost, path: "/api/v1/admin/materials", body: fmt.Sprintf(`{"title":%q,"description":"","status":"draft"}`, title), cookie: cookie, csrf: csrf, origin: f.origin})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create material = %d %s", rec.Code, rec.Body.String())
	}
	return decode[contentData](t, rec).Data.ID
}

func (f *fixture) attachFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(f.attachDir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func TestAdminAttachmentsRequireSessionAndCSRF(t *testing.T) {
	f := newFixture(t, fixtureOpts{attachments: true})
	cookie, csrf := f.loggedIn()
	mid := f.createMaterial(t, cookie, csrf, "Materi")
	path := fmt.Sprintf("/api/v1/admin/materials/%d/attachments", mid)

	if rec := f.sendUpload(path, upload{field: "file", name: "a.pdf", body: strings.NewReader(testPDF), origin: f.origin}); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous upload = %d, want 401", rec.Code)
	}
	if rec := f.sendUpload(path, upload{field: "file", name: "a.pdf", body: strings.NewReader(testPDF), cookie: cookie, origin: f.origin}); rec.Code != http.StatusForbidden {
		t.Errorf("upload without CSRF = %d, want 403", rec.Code)
	}
	if rec := f.sendUpload(path, upload{field: "file", name: "a.pdf", body: strings.NewReader(testPDF), cookie: cookie, csrf: csrf, origin: "https://evil.test"}); rec.Code != http.StatusForbidden {
		t.Errorf("cross-origin upload = %d, want 403", rec.Code)
	}
	if rec := f.do(req{method: http.MethodGet, path: path}); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous list = %d, want 401", rec.Code)
	}
	if rec := f.do(req{method: http.MethodDelete, path: "/api/v1/admin/attachments/1", cookie: cookie, origin: f.origin}); rec.Code != http.StatusForbidden {
		t.Errorf("delete without CSRF = %d, want 403", rec.Code)
	}
	if got := f.attachFiles(t); len(got) != 0 {
		t.Errorf("rejected requests stored files: %v", got)
	}
}

func TestAdminAttachmentLifecycleAndAudit(t *testing.T) {
	f := newFixture(t, fixtureOpts{attachments: true, persistAudit: true})
	cookie, csrf := f.loggedIn()
	mid := f.createMaterial(t, cookie, csrf, "Algoritma")
	path := fmt.Sprintf("/api/v1/admin/materials/%d/attachments", mid)
	send := func(u upload) *httptest.ResponseRecorder {
		u.cookie, u.csrf, u.origin = cookie, csrf, f.origin
		return f.sendUpload(path, u)
	}

	rec := send(upload{field: "file", name: "Materi K1.pdf", body: strings.NewReader(testPDF)})
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body.String())
	}
	var created struct{ Data attachmentData }
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	a := created.Data
	if a.ID < 1 || a.MaterialID != mid || a.Name != "Materi K1.pdf" || a.Kind != "pdf" ||
		a.ContentType != "application/pdf" || a.Size != int64(len(testPDF)) || a.CreatedAt == "" {
		t.Fatalf("unexpected upload response: %+v", a)
	}
	if loc := rec.Header().Get("Location"); loc != fmt.Sprintf("/api/v1/admin/attachments/%d", a.ID) {
		t.Errorf("Location = %q", loc)
	}
	if strings.Contains(rec.Body.String(), "storage") || strings.Contains(rec.Body.String(), f.attachFiles(t)[0]) {
		t.Errorf("response leaks the storage key: %s", rec.Body.String())
	}

	list := f.do(req{method: http.MethodGet, path: path, cookie: cookie})
	var listed struct{ Data []attachmentData }
	if list.Code != 200 || json.Unmarshal(list.Body.Bytes(), &listed) != nil || len(listed.Data) != 1 || listed.Data[0].ID != a.ID {
		t.Fatalf("list = %d %s", list.Code, list.Body.String())
	}

	before := auditCount(t, f)
	for name, tc := range map[string]struct {
		u    upload
		path string
		want int
		code string
	}{
		"json body":        {upload{field: "file", name: "a.pdf", body: strings.NewReader(testPDF), contentType: "application/json"}, path, 415, "UNSUPPORTED_MEDIA_TYPE"},
		"fake pdf":         {upload{field: "file", name: "a.pdf", body: strings.NewReader("<script>alert(1)</script>")}, path, 415, "UNSUPPORTED_FILE_TYPE"},
		"exe renamed":      {upload{field: "file", name: "a.pdf.exe", body: strings.NewReader(testPDF)}, path, 415, "UNSUPPORTED_FILE_TYPE"},
		"wrong field":      {upload{field: "berkas", name: "a.pdf", body: strings.NewReader(testPDF)}, path, 400, "INVALID_REQUEST"},
		"missing material": {upload{field: "file", name: "a.pdf", body: strings.NewReader(testPDF)}, "/api/v1/admin/materials/999999/attachments", 404, "NOT_FOUND"},
	} {
		tc.u.cookie, tc.u.csrf, tc.u.origin = cookie, csrf, f.origin
		r := f.sendUpload(tc.path, tc.u)
		if r.Code != tc.want || decode[errBody](t, r).Error.Code != tc.code {
			t.Errorf("%s = %d %s, want %d %s", name, r.Code, r.Body.String(), tc.want, tc.code)
		}
	}
	if got := len(f.attachFiles(t)); got != 1 {
		t.Errorf("rejected uploads left files: %d on disk, want 1", got)
	}
	if got := auditCount(t, f); got != before {
		t.Errorf("rejected uploads were audited: before=%d after=%d", before, got)
	}

	del := fmt.Sprintf("/api/v1/admin/attachments/%d", a.ID)
	if r := f.do(req{method: http.MethodDelete, path: del, cookie: cookie, csrf: csrf, origin: f.origin}); r.Code != http.StatusNoContent {
		t.Fatalf("delete = %d %s", r.Code, r.Body.String())
	}
	if got := f.attachFiles(t); len(got) != 0 {
		t.Errorf("file still on disk after delete: %v", got)
	}
	if r := f.do(req{method: http.MethodDelete, path: del, cookie: cookie, csrf: csrf, origin: f.origin}); r.Code != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", r.Code)
	}

	rows, err := f.pool.Query(t.Context(), `select action, actor_user_id, detail from audit_log where action like 'content.attachment_%' order by id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var actions []string
	for rows.Next() {
		var action string
		var actor int64
		var detail map[string]any
		if err := rows.Scan(&action, &actor, &detail); err != nil {
			t.Fatal(err)
		}
		actions = append(actions, action)
		if actor != f.userID || len(detail) != 3 || detail["entityId"] != fmt.Sprint(a.ID) || detail["materialId"] != fmt.Sprint(mid) || detail["title"] != "Materi K1.pdf" {
			t.Errorf("%s: actor=%d detail=%#v", action, actor, detail)
		}
	}
	if fmt.Sprint(actions) != "[content.attachment_uploaded content.attachment_deleted]" {
		t.Errorf("audit actions = %v", actions)
	}
}

// bigPDF yields a PDF header followed by zero bytes, n bytes in total, without allocating them.
func bigPDF(n int64) io.Reader {
	return io.MultiReader(strings.NewReader("%PDF-1.7\n"), io.LimitReader(zeros{}, n-9))
}

type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestAdminAttachmentSizeLimitCountAndQuota(t *testing.T) {
	f := newFixture(t, fixtureOpts{attachments: true, attachTotal: 3 * int64(len(testPDF))})
	cookie, csrf := f.loggedIn()
	send := func(mid int64, body io.Reader) *httptest.ResponseRecorder {
		return f.sendUpload(fmt.Sprintf("/api/v1/admin/materials/%d/attachments", mid), upload{field: "file", name: "a.pdf", body: body, cookie: cookie, csrf: csrf, origin: f.origin})
	}

	m1 := f.createMaterial(t, cookie, csrf, "Besar")
	if r := send(m1, bigPDF(25<<20+1)); r.Code != http.StatusRequestEntityTooLarge || decode[errBody](t, r).Error.Code != "PAYLOAD_TOO_LARGE" {
		t.Errorf("25 MiB + 1 byte = %d %s, want 413", r.Code, r.Body.String())
	}
	if got := f.attachFiles(t); len(got) != 0 {
		t.Errorf("oversized upload left files: %v", got)
	}

	for i := 0; i < 3; i++ {
		if r := send(m1, bytes.NewReader([]byte(testPDF))); r.Code != http.StatusCreated {
			t.Fatalf("upload %d = %d %s", i+1, r.Code, r.Body.String())
		}
	}
	if r := send(m1, bytes.NewReader([]byte(testPDF))); r.Code != http.StatusInsufficientStorage || decode[errBody](t, r).Error.Code != "STORAGE_QUOTA_EXCEEDED" {
		t.Errorf("over quota = %d %s, want 507", r.Code, r.Body.String())
	}
	if got := len(f.attachFiles(t)); got != 3 {
		t.Errorf("files = %d, want 3", got)
	}
}

func TestAdminAttachmentFiveFileLimit(t *testing.T) {
	f := newFixture(t, fixtureOpts{attachments: true})
	cookie, csrf := f.loggedIn()
	mid := f.createMaterial(t, cookie, csrf, "Lima")
	path := fmt.Sprintf("/api/v1/admin/materials/%d/attachments", mid)
	for i := 0; i < 5; i++ {
		if r := f.sendUpload(path, upload{field: "file", name: "a.pdf", body: strings.NewReader(testPDF), cookie: cookie, csrf: csrf, origin: f.origin}); r.Code != http.StatusCreated {
			t.Fatalf("upload %d = %d %s", i+1, r.Code, r.Body.String())
		}
	}
	r := f.sendUpload(path, upload{field: "file", name: "a.pdf", body: strings.NewReader(testPDF), cookie: cookie, csrf: csrf, origin: f.origin})
	if r.Code != http.StatusConflict || decode[errBody](t, r).Error.Code != "ATTACHMENT_LIMIT" {
		t.Errorf("sixth upload = %d %s, want 409 ATTACHMENT_LIMIT", r.Code, r.Body.String())
	}
}

func TestAdminDeleteMaterialRemovesAttachmentFiles(t *testing.T) {
	f := newFixture(t, fixtureOpts{attachments: true, persistAudit: true})
	cookie, csrf := f.loggedIn()
	mid := f.createMaterial(t, cookie, csrf, "Akan Dihapus")
	for i := 0; i < 2; i++ {
		if r := f.sendUpload(fmt.Sprintf("/api/v1/admin/materials/%d/attachments", mid), upload{field: "file", name: "a.pdf", body: strings.NewReader(testPDF), cookie: cookie, csrf: csrf, origin: f.origin}); r.Code != http.StatusCreated {
			t.Fatalf("upload = %d %s", r.Code, r.Body.String())
		}
	}
	if r := f.do(req{method: http.MethodDelete, path: fmt.Sprintf("/api/v1/admin/materials/%d", mid), cookie: cookie, csrf: csrf, origin: f.origin}); r.Code != http.StatusNoContent {
		t.Fatalf("delete material = %d %s", r.Code, r.Body.String())
	}
	if got := f.attachFiles(t); len(got) != 0 {
		t.Errorf("attachment files left after material delete: %v", got)
	}
	var n int
	if err := f.pool.QueryRow(t.Context(), `select count(*) from audit_log where action='content.material_deleted'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("material_deleted audit rows = %d (err %v), want 1", n, err)
	}
	if r := f.do(req{method: http.MethodDelete, path: fmt.Sprintf("/api/v1/admin/materials/%d", mid), cookie: cookie, csrf: csrf, origin: f.origin}); r.Code != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", r.Code)
	}
}
