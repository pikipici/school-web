package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

type contentData struct {
	Data struct {
		ID          int64  `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		UpdatedAt   string `json:"updatedAt"`
	} `json:"data"`
}

func TestAdminContentExactRoutesAndAuth(t *testing.T) {
	f := newFixture(t, fixtureOpts{})
	for _, path := range []string{"/api/v1/admin/announcements", "/api/v1/admin/materials"} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			rec := f.do(req{method: method, path: path, body: `{}`})
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("anonymous %s %s = %d, want 401 without redirect", method, path, rec.Code)
			}
		}
	}
	cookie, csrf := f.loggedIn()
	for name, r := range map[string]req{
		"missing csrf": {method: http.MethodPost, path: "/api/v1/admin/announcements", body: `{}`, cookie: cookie, origin: f.origin},
		"wrong origin": {method: http.MethodPost, path: "/api/v1/admin/announcements", body: `{}`, cookie: cookie, csrf: csrf, origin: "https://evil.test"},
	} {
		if rec := f.do(r); rec.Code != http.StatusForbidden {
			t.Errorf("%s = %d, want 403", name, rec.Code)
		}
	}
}

func TestAdminContentCRUDValidationAndAudit(t *testing.T) {
	f := newFixture(t, fixtureOpts{persistAudit: true})
	cookie, csrf := f.loggedIn()
	do := func(method, path, body, ct string) *response {
		r := f.do(req{method: method, path: path, body: body, contentType: ct, cookie: cookie, csrf: csrf, origin: f.origin})
		return &response{r.Code, r.Body.String()}
	}
	valid := func(kind, body string) contentData {
		r := do(http.MethodPost, "/api/v1/admin/"+kind, body, "application/json")
		if r.code != 201 {
			t.Fatalf("create %s = %d %s", kind, r.code, r.body)
		}
		var out contentData
		if err := json.Unmarshal([]byte(r.body), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	a := valid("announcements", `{"title":"Notice","body":"Safe body","status":"draft","pinned":true}`)
	m := valid("materials", `{"title":"Lesson","description":"Safe description","status":"draft","sortOrder":2}`)
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/api/v1/admin/announcements", 200}, {http.MethodGet, fmt.Sprintf("/api/v1/admin/announcements/%d", a.Data.ID), 200},
		{http.MethodGet, "/api/v1/admin/materials", 200}, {http.MethodGet, fmt.Sprintf("/api/v1/admin/materials/%d", m.Data.ID), 200},
	} {
		if r := do(tc.method, tc.path, "", ""); r.code != tc.want {
			t.Errorf("%s %s = %d %s", tc.method, tc.path, r.code, r.body)
		}
	}

	f.clk.Advance(time.Second)
	ap := fmt.Sprintf(`{"title":"Notice 2","body":"Changed","status":"draft","updatedAt":%q}`, a.Data.UpdatedAt)
	if r := do(http.MethodPatch, fmt.Sprintf("/api/v1/admin/announcements/%d", a.Data.ID), ap, "application/json"); r.code != 200 {
		t.Fatalf("update announcement = %d %s", r.code, r.body)
	}
	mp := fmt.Sprintf(`{"title":"Lesson 2","description":"Changed","status":"draft","sortOrder":3,"updatedAt":%q}`, m.Data.UpdatedAt)
	if r := do(http.MethodPatch, fmt.Sprintf("/api/v1/admin/materials/%d", m.Data.ID), mp, "application/json"); r.code != 200 {
		t.Fatalf("update material = %d %s", r.code, r.body)
	}
	beforeFailures := auditCount(t, f)
	if r := do(http.MethodPatch, fmt.Sprintf("/api/v1/admin/announcements/%d", a.Data.ID), ap, "application/json"); r.code != 409 {
		t.Errorf("stale update = %d %s", r.code, r.body)
	}
	if r := do(http.MethodPost, "/api/v1/admin/materials", `{"title":"Bad topic","description":"x","status":"draft","topicId":999999999}`, "application/json"); r.code != 422 {
		t.Errorf("invalid topic = %d %s", r.code, r.body)
	}
	for name, tc := range map[string]struct {
		body, ct string
		want     int
	}{
		"malformed": {`{"title":`, "application/json", 400}, "unknown": {`{"title":"x","wat":1}`, "application/json", 400},
		"wrong content type": {`{}`, "text/plain", 415}, "oversized": {`{"title":"x","body":"` + strings.Repeat("x", 33000) + `"}`, "application/json", 413},
	} {
		if r := do(http.MethodPost, "/api/v1/admin/announcements", tc.body, tc.ct); r.code != tc.want {
			t.Errorf("%s = %d, want %d: %s", name, r.code, tc.want, r.body)
		}
	}
	if got := auditCount(t, f); got != beforeFailures {
		t.Errorf("failed mutations emitted audit rows: before=%d after=%d", beforeFailures, got)
	}
	for _, tc := range []struct {
		kind string
		id   int64
	}{{"announcements", a.Data.ID}, {"materials", m.Data.ID}} {
		if r := do(http.MethodDelete, fmt.Sprintf("/api/v1/admin/%s/%d", tc.kind, tc.id), "", ""); r.code != 204 {
			t.Errorf("delete %s = %d %s", tc.kind, r.code, r.body)
		}
	}
	rows, err := f.pool.Query(t.Context(), `select action, actor_user_id, detail from audit_log where action like 'content.%' order by id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	want := []string{"content.announcement_created", "content.material_created", "content.announcement_updated", "content.material_updated", "content.announcement_deleted", "content.material_deleted"}
	var got []string
	for rows.Next() {
		var action string
		var actor int64
		var detail map[string]any
		if err := rows.Scan(&action, &actor, &detail); err != nil {
			t.Fatal(err)
		}
		got = append(got, action)
		if actor != f.userID {
			t.Errorf("actor=%d want %d", actor, f.userID)
		}
		if len(detail) != 2 || detail["entityId"] == nil || detail["title"] == nil {
			t.Errorf("unsafe/unexpected detail for %s: %#v", action, detail)
		}
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("actions=%v want %v", got, want)
	}
}

type response struct {
	code int
	body string
}

func auditCount(t *testing.T, f *fixture) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(t.Context(), `select count(*) from audit_log where action like 'content.%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
