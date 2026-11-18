package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestFindingsAdminAndAnonymousPublicDetail(t *testing.T) {
	p := newProgressFixture(t, fixtureOpts{persistAudit: true})
	c := p.mustCreate("/api/v1/admin/classes", fmt.Sprintf(`{"yearId":%d,"code":"9A","name":""}`, p.activeYearID()))
	for n := 1; n <= 2; n++ {
		g := p.mustCreate("/api/v1/admin/groups", fmt.Sprintf(`{"classId":%d,"topicId":%d,"name":"Kelompok %d","email":"private@example.test"}`, c.Data.ID, p.topicID("K1"), n))
		base := fmt.Sprintf("/api/v1/admin/groups/%d", g.Data.ID)
		pub := fmt.Sprintf("/api/v1/public/groups/%d", g.Data.ID)
		f := p.mustCreate(base+"/findings", `{"fileType":"PPT","location":"Slide 5","problem":"Diagram salah","action":"Perbaiki simpul","priority":"WAJIB"}`)
		r := p.do(req{method: "GET", path: pub})
		if r.Code != 200 || strings.Contains(r.Body.String(), "Diagram salah") {
			t.Fatalf("draft leak: %d %s", r.Code, r.Body.String())
		}
		assertFindingPublicKeys(t, r.Body.String())
		var review struct {
			Data struct {
				UpdatedAt string `json:"updatedAt"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(p.admin("GET", base+"/review-status", "").body), &review); err != nil {
			t.Fatal(err)
		}
		if r := p.admin("POST", base+"/publish", fmt.Sprintf(`{"status":"PERLU_REVISI","updatedAt":%q}`, review.Data.UpdatedAt)); r.code != 200 {
			t.Fatalf("publish: %+v", r)
		}
		r = p.do(req{method: "GET", path: pub})
		if r.Code != 200 || !strings.Contains(r.Body.String(), "Diagram salah") {
			t.Fatalf("anonymous publication: %d %s", r.Code, r.Body.String())
		}
		assertFindingPublicKeys(t, r.Body.String())
		r = p.do(req{method: "POST", path: base + "/publish"})
		if r.Code != 401 {
			t.Fatalf("anonymous write: %d", r.Code)
		}
		del := p.admin("DELETE", fmt.Sprintf("/api/v1/admin/findings/%d?updatedAt=%s", f.Data.ID, url.QueryEscape(f.Data.UpdatedAt)), "")
		if del.code != 409 {
			t.Fatalf("delete published: %+v", del)
		}
	}
	if r := p.do(req{method: "GET", path: "/api/v1/public/groups/999999"}); r.Code != 404 {
		t.Fatalf("missing detail: %d", r.Code)
	}
}

func TestFindingsRoutesRequireAdminAndCSRF(t *testing.T) {
	p := newProgressFixture(t, fixtureOpts{})
	for _, path := range []string{"/api/v1/admin/groups/1/findings", "/api/v1/admin/groups/1/review-status"} {
		r := p.do(req{method: http.MethodGet, path: path})
		if r.Code != 401 {
			t.Fatalf("anonymous %s: %d", path, r.Code)
		}
	}
	for _, v := range []struct{ method, path string }{{"POST", "/api/v1/admin/groups/1/findings"}, {"POST", "/api/v1/admin/groups/1/publish"}, {"PATCH", "/api/v1/admin/findings/1"}, {"DELETE", "/api/v1/admin/findings/1"}, {"PATCH", "/api/v1/admin/groups/1/review-status"}} {
		r := p.do(req{method: v.method, path: v.path, cookie: p.cookie, origin: p.origin, body: `{}`})
		if r.Code != 403 {
			t.Fatalf("missing CSRF %s: %d", v.path, r.Code)
		}
	}
}

// Keep exact public DTO checks independent of the administrator DTO.
func assertFindingPublicKeys(t *testing.T, body string) {
	t.Helper()
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatal(err)
	}
	want := []string{"id", "name", "classCode", "topic", "submission", "review", "notebook", "complete", "updatedAt", "findings"}
	if len(envelope.Data) != len(want) {
		t.Fatalf("public keys: %s", body)
	}
	for _, key := range want {
		if _, ok := envelope.Data[key]; !ok {
			t.Fatalf("missing %s", key)
		}
	}
	var findings []map[string]json.RawMessage
	if err := json.Unmarshal(envelope.Data["findings"], &findings); err != nil {
		t.Fatal(err)
	}
	keys := []string{"id", "fileType", "location", "problem", "action", "priority", "resolved", "publishedAt", "updatedAt"}
	for _, finding := range findings {
		if len(finding) != len(keys) {
			t.Fatalf("finding keys: %s", body)
		}
		for _, key := range keys {
			if _, ok := finding[key]; !ok {
				t.Fatalf("missing finding key %s", key)
			}
		}
	}
	for _, secret := range []string{"senderEmail", "subject", "note", "createdBy", "expectedEmail"} {
		if strings.Contains(body, `"`+secret+`"`) {
			t.Fatalf("leaked %s", secret)
		}
	}
}
