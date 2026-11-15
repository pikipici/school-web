package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func (p *progressFixture) group(name string) int64 {
	p.t.Helper()
	year := p.activeYearID()
	var class int64
	if err := p.pool.QueryRow(p.t.Context(), "select id from classes where code = '9A'").Scan(&class); err != nil {
		class = p.mustCreate("/api/v1/admin/classes", fmt.Sprintf(`{"yearId":%d,"code":"9A"}`, year)).Data.ID
	}
	return p.mustCreate("/api/v1/admin/groups", fmt.Sprintf(`{"classId":%d,"topicId":%d,"name":%q}`, class, p.topicID("K1"), name)).Data.ID
}

type roundData struct {
	Data struct {
		ID          int64   `json:"id"`
		GroupID     int64   `json:"groupId"`
		Number      int     `json:"roundNumber"`
		ReceivedAt  string  `json:"receivedAt"`
		SenderEmail *string `json:"senderEmail"`
		PPT         bool    `json:"ppt"`
		PDF         bool    `json:"pdf"`
		UpdatedAt   string  `json:"updatedAt"`
	} `json:"data"`
}

func TestAdminSubmissionsRequireSessionAndCSRF(t *testing.T) {
	f := newFixture(t, fixtureOpts{})
	for _, path := range []string{"/api/v1/admin/groups/1/submissions", "/api/v1/admin/groups/1/notebook"} {
		if rec := f.do(req{method: http.MethodGet, path: path}); rec.Code != http.StatusUnauthorized {
			t.Errorf("anonymous GET %s = %d", path, rec.Code)
		}
	}
	cookie, _ := f.loggedIn()
	for _, r := range []req{
		{method: http.MethodPost, path: "/api/v1/admin/groups/1/submissions", body: `{}`, cookie: cookie, origin: f.origin},
		{method: http.MethodPatch, path: "/api/v1/admin/submissions/1", body: `{}`, cookie: cookie, origin: f.origin},
		{method: http.MethodDelete, path: "/api/v1/admin/submissions/1", cookie: cookie, origin: f.origin},
		{method: http.MethodPut, path: "/api/v1/admin/groups/1/notebook", body: `{}`, cookie: cookie, origin: f.origin},
	} {
		if rec := f.do(r); rec.Code != http.StatusForbidden {
			t.Errorf("%s %s without CSRF = %d, want 403", r.method, r.path, rec.Code)
		}
	}
}

func TestAdminSubmissionLifecycle(t *testing.T) {
	p := newProgressFixture(t, fixtureOpts{persistAudit: true})
	g := p.group("Kelompok 1")
	base := fmt.Sprintf("/api/v1/admin/groups/%d", g)
	at := p.clk.now.Add(-time.Hour).Format(time.RFC3339)

	list := func() (string, int) {
		r := p.admin(http.MethodGet, base+"/submissions", "")
		var out struct {
			Data struct {
				Submission string            `json:"submission"`
				Rounds     []json.RawMessage `json:"rounds"`
			} `json:"data"`
		}
		if r.code != 200 || json.Unmarshal([]byte(r.body), &out) != nil {
			t.Fatalf("list = %d %s", r.code, r.body)
		}
		return out.Data.Submission, len(out.Data.Rounds)
	}
	if s, n := list(); s != "BELUM" || n != 0 {
		t.Fatalf("empty = %s %d", s, n)
	}

	r := p.admin(http.MethodPost, base+"/submissions", fmt.Sprintf(`{"roundNumber":0,"receivedAt":%q,"senderEmail":"Kelompok1@Gmail.com","subject":"Proyek","ppt":true,"pptFilename":"Graf.pptx"}`, at))
	if r.code != 201 {
		t.Fatalf("record round 0 = %d %s", r.code, r.body)
	}
	var r0 roundData
	_ = json.Unmarshal([]byte(r.body), &r0)
	if r0.Data.Number != 0 || r0.Data.SenderEmail == nil || *r0.Data.SenderEmail != "kelompok1@gmail.com" || !r0.Data.PPT || r0.Data.PDF {
		t.Fatalf("round 0 = %+v", r0.Data)
	}
	if s, _ := list(); s != "SEBAGIAN" {
		t.Errorf("after PPT = %s", s)
	}
	if r := p.admin(http.MethodPost, base+"/submissions", fmt.Sprintf(`{"roundNumber":0,"receivedAt":%q,"ppt":true}`, at)); r.code != 409 || errCode(t, r.body) != "CONFLICT" {
		t.Errorf("double submit = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodPost, base+"/submissions", fmt.Sprintf(`{"roundNumber":1,"receivedAt":%q}`, at)); r.code != 422 || errCode(t, r.body) != "VALIDATION_ERROR" {
		t.Errorf("no file = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodPost, base+"/submissions", `{"roundNumber":1,"receivedAt":"kemarin","pdf":true}`); r.code != 422 {
		t.Errorf("bad time = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodPost, "/api/v1/admin/groups/999999/submissions", fmt.Sprintf(`{"roundNumber":0,"receivedAt":%q,"ppt":true}`, at)); r.code != 404 {
		t.Errorf("missing group = %d %s", r.code, r.body)
	}
	r1 := p.admin(http.MethodPost, base+"/submissions", fmt.Sprintf(`{"roundNumber":1,"receivedAt":%q,"pdf":true}`, at))
	if r1.code != 201 {
		t.Fatalf("round 1 = %d %s", r1.code, r1.body)
	}
	var r1d roundData
	_ = json.Unmarshal([]byte(r1.body), &r1d)
	if s, n := list(); s != "LENGKAP" || n != 2 {
		t.Errorf("after PDF = %s %d", s, n)
	}

	p.clk.Advance(time.Second)
	patch := fmt.Sprintf(`{"receivedAt":%q,"ppt":true,"pdf":false,"note":"koreksi","updatedAt":%q}`, at, r0.Data.UpdatedAt)
	if r := p.admin(http.MethodPatch, fmt.Sprintf("/api/v1/admin/submissions/%d", r0.Data.ID), patch); r.code != 200 || !strings.Contains(r.body, `"note":"koreksi"`) {
		t.Errorf("patch = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodPatch, fmt.Sprintf("/api/v1/admin/submissions/%d", r0.Data.ID), patch); r.code != 409 || errCode(t, r.body) != "CONFLICT" {
		t.Errorf("stale patch = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodDelete, fmt.Sprintf("/api/v1/admin/submissions/%d", r0.Data.ID), ""); r.code != 409 || errCode(t, r.body) != "NOT_LAST_ROUND" {
		t.Errorf("delete non-last = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodDelete, fmt.Sprintf("/api/v1/admin/groups/%d", g), ""); r.code != 409 || errCode(t, r.body) != "IN_USE" {
		t.Errorf("delete group with rounds = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodDelete, fmt.Sprintf("/api/v1/admin/submissions/%d", r1d.Data.ID), ""); r.code != 204 {
		t.Errorf("delete last = %d %s", r.code, r.body)
	}

	// Notebook.
	if r := p.admin(http.MethodGet, base+"/notebook", ""); r.code != 200 || !strings.Contains(r.body, `"received":false`) || !strings.Contains(r.body, `"receivedAt":null`) {
		t.Errorf("default notebook = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodPut, base+"/notebook", `{"received":true,"note":"sampul biru"}`); r.code != 200 || !strings.Contains(r.body, `"received":true`) {
		t.Errorf("receive notebook = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodPut, base+"/notebook", `{"received":true,"receivedAt":"2999-01-01T00:00:00Z"}`); r.code != 422 {
		t.Errorf("future notebook = %d %s", r.code, r.body)
	}

	pub := p.do(req{method: http.MethodGet, path: "/api/v1/public/progress"})
	body := pub.Body.String()
	if !strings.Contains(body, `"submission":"SEBAGIAN"`) || !strings.Contains(body, `"notebook":"SUDAH_DITERIMA"`) || strings.Contains(body, "gmail") || strings.Contains(body, "koreksi") {
		t.Errorf("public after records = %s", body)
	}

	rows, err := p.pool.Query(t.Context(), `select action from audit_log where action like 'progress.round%' or action like 'progress.notebook%' order by id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var actions []string
	for rows.Next() {
		var a string
		_ = rows.Scan(&a)
		actions = append(actions, a)
	}
	if fmt.Sprint(actions) != "[progress.round_recorded progress.round_recorded progress.round_updated progress.round_deleted progress.notebook_updated]" {
		t.Errorf("audit = %v", actions)
	}
}
