package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"
)

type idData struct {
	Data struct {
		ID        int64   `json:"id"`
		Code      string  `json:"code"`
		Name      string  `json:"name"`
		Label     string  `json:"label"`
		Email     *string `json:"email"`
		Active    bool    `json:"active"`
		UpdatedAt string  `json:"updatedAt"`
	} `json:"data"`
}

type progressFixture struct {
	*fixture
	cookie *http.Cookie
	csrf   string
}

func newProgressFixture(t *testing.T, o fixtureOpts) *progressFixture {
	f := newFixture(t, o)
	c, csrf := f.loggedIn()
	return &progressFixture{f, c, csrf}
}

func (p *progressFixture) admin(method, path, body string) *response {
	p.t.Helper()
	r := p.do(req{method: method, path: path, body: body, cookie: p.cookie, csrf: p.csrf, origin: p.origin})
	return &response{r.Code, r.Body.String()}
}

func (p *progressFixture) mustCreate(path, body string) idData {
	p.t.Helper()
	r := p.admin(http.MethodPost, path, body)
	if r.code != http.StatusCreated {
		p.t.Fatalf("POST %s %s = %d %s", path, body, r.code, r.body)
	}
	var out idData
	if err := json.Unmarshal([]byte(r.body), &out); err != nil {
		p.t.Fatal(err)
	}
	return out
}

func (p *progressFixture) topicID(code string) int64 {
	p.t.Helper()
	var id int64
	if err := p.pool.QueryRow(p.t.Context(), "select id from topics where code=$1", code).Scan(&id); err != nil {
		p.t.Fatal(err)
	}
	return id
}

func (p *progressFixture) activeYearID() int64 {
	p.t.Helper()
	var id int64
	if err := p.pool.QueryRow(p.t.Context(), "select id from academic_years where is_active").Scan(&id); err != nil {
		p.t.Fatal(err)
	}
	return id
}

func errCode(t *testing.T, body string) string {
	t.Helper()
	var e errBody
	if err := json.Unmarshal([]byte(body), &e); err != nil {
		t.Fatalf("not an error envelope: %s", body)
	}
	return e.Error.Code
}

func TestAdminProgressRequiresSessionAndCSRF(t *testing.T) {
	f := newFixture(t, fixtureOpts{})
	for _, path := range []string{"/api/v1/admin/academic-years", "/api/v1/admin/classes", "/api/v1/admin/groups?class=1", "/api/v1/admin/classes/1", "/api/v1/admin/groups/1"} {
		if rec := f.do(req{method: http.MethodGet, path: path}); rec.Code != http.StatusUnauthorized {
			t.Errorf("anonymous GET %s = %d, want 401", path, rec.Code)
		}
	}
	cookie, csrf := f.loggedIn()
	for _, r := range []req{
		{method: http.MethodPost, path: "/api/v1/admin/classes", body: `{}`, cookie: cookie, origin: f.origin},
		{method: http.MethodPost, path: "/api/v1/admin/groups", body: `{}`, cookie: cookie, csrf: csrf, origin: "https://evil.test"},
		{method: http.MethodPost, path: "/api/v1/admin/academic-years/1/activate", cookie: cookie, origin: f.origin},
		{method: http.MethodDelete, path: "/api/v1/admin/classes/1", cookie: cookie, origin: f.origin},
	} {
		if rec := f.do(r); rec.Code != http.StatusForbidden {
			t.Errorf("%s %s without CSRF/origin = %d, want 403", r.method, r.path, rec.Code)
		}
	}
}

func TestAdminClassesAndGroupsLifecycleAndAudit(t *testing.T) {
	p := newProgressFixture(t, fixtureOpts{persistAudit: true})
	year := p.activeYearID()
	k1, k2 := p.topicID("K1"), p.topicID("K2")

	// Academic years.
	years := p.admin(http.MethodGet, "/api/v1/admin/academic-years", "")
	if years.code != 200 || !strings.Contains(years.body, `"label":"2026/2027"`) || !strings.Contains(years.body, `"active":true`) {
		t.Fatalf("years = %d %s", years.code, years.body)
	}
	next := p.mustCreate("/api/v1/admin/academic-years", `{"label":" 2027/2028 "}`)
	if next.Data.Label != "2027/2028" || next.Data.Active {
		t.Fatalf("created year = %+v", next.Data)
	}

	// Classes.
	c := p.mustCreate("/api/v1/admin/classes", fmt.Sprintf(`{"yearId":%d,"code":" 9A ","name":"Kelas pagi"}`, year))
	if c.Data.Code != "9A" || !c.Data.Active {
		t.Fatalf("created class = %+v", c.Data)
	}
	list := p.admin(http.MethodGet, "/api/v1/admin/classes", "")
	if list.code != 200 || !strings.Contains(list.body, `"code":"9A"`) || !strings.Contains(list.body, `"groupCount":0`) {
		t.Fatalf("classes (active year by default) = %d %s", list.code, list.body)
	}
	if r := p.admin(http.MethodGet, fmt.Sprintf("/api/v1/admin/classes?year=%d", next.Data.ID), ""); r.code != 200 || !strings.Contains(r.body, `"data":[]`) {
		t.Errorf("classes of new year = %d %s", r.code, r.body)
	}

	// Groups.
	g := p.mustCreate("/api/v1/admin/groups", fmt.Sprintf(`{"classId":%d,"topicId":%d,"name":"Kelompok 1","email":" Kelompok1.9A.2627.Informatika@Gmail.com ","sortOrder":1}`, c.Data.ID, k1))
	if g.Data.Email == nil || *g.Data.Email != "kelompok1.9a.2627.informatika@gmail.com" || !g.Data.Active {
		t.Fatalf("created group = %+v", g.Data)
	}
	mustGroup2 := p.mustCreate("/api/v1/admin/groups", fmt.Sprintf(`{"classId":%d,"topicId":%d,"name":"Kelompok 2"}`, c.Data.ID, k2))
	if mustGroup2.Data.Email != nil {
		t.Errorf("group without email = %+v", mustGroup2.Data)
	}
	if r := p.admin(http.MethodGet, fmt.Sprintf("/api/v1/admin/groups?class=%d", c.Data.ID), ""); r.code != 200 || strings.Index(r.body, "Kelompok 2") > strings.Index(r.body, "Kelompok 1") {
		t.Errorf("groups list = %d %s (want sort order: Kelompok 2 first)", r.code, r.body)
	}
	if r := p.admin(http.MethodGet, fmt.Sprintf("/api/v1/admin/groups/%d", g.Data.ID), ""); r.code != 200 || !strings.Contains(r.body, `"classId":`) || !strings.Contains(r.body, `"topicId":`) {
		t.Errorf("get group = %d %s", r.code, r.body)
	}

	before := progressAuditCount(t, p)
	for name, tc := range map[string]struct {
		method, path, body string
		want               int
		code               string
	}{
		"duplicate year":    {"POST", "/api/v1/admin/academic-years", `{"label":"2027/2028"}`, 409, "DUPLICATE"},
		"bad year":          {"POST", "/api/v1/admin/academic-years", `{"label":"2027/2029"}`, 422, "VALIDATION_ERROR"},
		"duplicate class":   {"POST", "/api/v1/admin/classes", fmt.Sprintf(`{"yearId":%d,"code":"9a"}`, year), 409, "DUPLICATE"},
		"unknown year":      {"POST", "/api/v1/admin/classes", `{"yearId":999999,"code":"9B"}`, 422, "INVALID_REFERENCE"},
		"bad class code":    {"POST", "/api/v1/admin/classes", fmt.Sprintf(`{"yearId":%d,"code":"9/A"}`, year), 422, "VALIDATION_ERROR"},
		"duplicate group":   {"POST", "/api/v1/admin/groups", fmt.Sprintf(`{"classId":%d,"topicId":%d,"name":"kelompok 1"}`, c.Data.ID, k1), 409, "DUPLICATE"},
		"unknown topic":     {"POST", "/api/v1/admin/groups", fmt.Sprintf(`{"classId":%d,"topicId":999999,"name":"X"}`, c.Data.ID), 422, "INVALID_REFERENCE"},
		"bad email":         {"POST", "/api/v1/admin/groups", fmt.Sprintf(`{"classId":%d,"topicId":%d,"name":"X","email":"bukan"}`, c.Data.ID, k1), 422, "VALIDATION_ERROR"},
		"malformed":         {"POST", "/api/v1/admin/groups", `{"name":`, 400, "INVALID_REQUEST"},
		"unknown field":     {"POST", "/api/v1/admin/classes", fmt.Sprintf(`{"yearId":%d,"code":"9B","members":["a"]}`, year), 400, "INVALID_REQUEST"},
		"groups no class":   {"GET", "/api/v1/admin/groups", "", 400, "VALIDATION_ERROR"},
		"groups bad class":  {"GET", "/api/v1/admin/groups?class=abc", "", 400, "VALIDATION_ERROR"},
		"classes bad year":  {"GET", "/api/v1/admin/classes?year=-1", "", 400, "VALIDATION_ERROR"},
		"missing class":     {"GET", "/api/v1/admin/classes/999999", "", 404, "NOT_FOUND"},
		"missing group":     {"GET", "/api/v1/admin/groups/999999", "", 404, "NOT_FOUND"},
		"activate missing":  {"POST", "/api/v1/admin/academic-years/999999/activate", "", 404, "NOT_FOUND"},
		"patch no active":   {"PATCH", fmt.Sprintf("/api/v1/admin/groups/%d", g.Data.ID), fmt.Sprintf(`{"classId":%d,"topicId":%d,"name":"Kelompok 1","updatedAt":%q}`, c.Data.ID, k1, g.Data.UpdatedAt), 422, "VALIDATION_ERROR"},
		"patch no stamp":    {"PATCH", fmt.Sprintf("/api/v1/admin/classes/%d", c.Data.ID), fmt.Sprintf(`{"yearId":%d,"code":"9A","active":true}`, year), 422, "VALIDATION_ERROR"},
		"delete busy class": {"DELETE", fmt.Sprintf("/api/v1/admin/classes/%d", c.Data.ID), "", 409, "IN_USE"},
	} {
		r := p.admin(tc.method, tc.path, tc.body)
		if r.code != tc.want || errCode(t, r.body) != tc.code {
			t.Errorf("%s = %d %s, want %d %s", name, r.code, r.body, tc.want, tc.code)
		}
	}
	if got := progressAuditCount(t, p); got != before {
		t.Errorf("failed requests were audited: before=%d after=%d", before, got)
	}

	// Updates with the optimistic-concurrency stamp.
	p.clk.Advance(time.Second)
	patch := fmt.Sprintf(`{"classId":%d,"topicId":%d,"name":"Kelompok Satu","email":"","sortOrder":1,"active":false,"updatedAt":%q}`, c.Data.ID, k2, g.Data.UpdatedAt)
	r := p.admin(http.MethodPatch, fmt.Sprintf("/api/v1/admin/groups/%d", g.Data.ID), patch)
	if r.code != 200 || !strings.Contains(r.body, `"active":false`) || !strings.Contains(r.body, `"email":null`) {
		t.Fatalf("patch group = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodPatch, fmt.Sprintf("/api/v1/admin/groups/%d", g.Data.ID), patch); r.code != 409 || errCode(t, r.body) != "CONFLICT" {
		t.Errorf("stale patch = %d %s", r.code, r.body)
	}
	cp := p.admin(http.MethodPatch, fmt.Sprintf("/api/v1/admin/classes/%d", c.Data.ID), fmt.Sprintf(`{"yearId":%d,"code":"9A","name":"","active":true,"updatedAt":%q}`, year, c.Data.UpdatedAt))
	if cp.code != 200 || !strings.Contains(cp.body, `"groupCount":2`) {
		t.Errorf("patch class = %d %s", cp.code, cp.body)
	}

	// Deletes.
	for _, id := range []int64{g.Data.ID, mustGroup2.Data.ID} {
		if r := p.admin(http.MethodDelete, fmt.Sprintf("/api/v1/admin/groups/%d", id), ""); r.code != 204 {
			t.Errorf("delete group = %d %s", r.code, r.body)
		}
	}
	if r := p.admin(http.MethodDelete, fmt.Sprintf("/api/v1/admin/classes/%d", c.Data.ID), ""); r.code != 204 {
		t.Errorf("delete class = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodDelete, fmt.Sprintf("/api/v1/admin/classes/%d", c.Data.ID), ""); r.code != 404 {
		t.Errorf("delete class again = %d", r.code)
	}

	// Activate the new year.
	if r := p.admin(http.MethodPost, fmt.Sprintf("/api/v1/admin/academic-years/%d/activate", next.Data.ID), ""); r.code != 204 {
		t.Fatalf("activate = %d %s", r.code, r.body)
	}
	if got := p.activeYearID(); got != next.Data.ID {
		t.Fatalf("active year = %d, want %d", got, next.Data.ID)
	}

	rows, err := p.pool.Query(t.Context(), `select action, actor_user_id, detail from audit_log where action like 'progress.%' order by id`)
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
		raw, _ := json.Marshal(detail)
		if actor != p.userID || detail["entityId"] == nil || detail["title"] == nil || strings.Contains(string(raw), "@") {
			t.Errorf("%s: actor=%d detail=%s", action, actor, raw)
		}
	}
	want := "[progress.year_created progress.class_created progress.group_created progress.group_created progress.group_updated progress.class_updated progress.group_deleted progress.group_deleted progress.class_deleted progress.year_activated]"
	if fmt.Sprint(actions) != want {
		t.Errorf("audit actions =\n%v\nwant\n%s", actions, want)
	}
}

func progressAuditCount(t *testing.T, p *progressFixture) int {
	t.Helper()
	var n int
	if err := p.pool.QueryRow(t.Context(), `select count(*) from audit_log where action like 'progress.%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPublicProgressAndSummary(t *testing.T) {
	p := newProgressFixture(t, fixtureOpts{})
	pub := func(path string) *response {
		r := p.do(req{method: http.MethodGet, path: path})
		return &response{r.Code, r.Body.String()}
	}

	empty := pub("/api/v1/public/summary")
	if empty.code != 200 || !strings.Contains(empty.body, `"totalGroups":0`) || !strings.Contains(empty.body, `"updatedAt":null`) || !strings.Contains(empty.body, `"manualUpdate":true`) {
		t.Fatalf("empty summary = %d %s", empty.code, empty.body)
	}

	year := p.activeYearID()
	a := p.mustCreate("/api/v1/admin/classes", fmt.Sprintf(`{"yearId":%d,"code":"9A"}`, year))
	b := p.mustCreate("/api/v1/admin/classes", fmt.Sprintf(`{"yearId":%d,"code":"9B"}`, year))
	k1 := p.topicID("K1")
	p.mustCreate("/api/v1/admin/groups", fmt.Sprintf(`{"classId":%d,"topicId":%d,"name":"Kelompok 1","email":"rahasia@gmail.com"}`, a.Data.ID, k1))
	p.mustCreate("/api/v1/admin/groups", fmt.Sprintf(`{"classId":%d,"topicId":%d,"name":"Kelompok 2"}`, b.Data.ID, k1))

	r := p.do(req{method: http.MethodGet, path: "/api/v1/public/progress"})
	if r.Code != 200 {
		t.Fatalf("progress = %d %s", r.Code, r.Body.String())
	}
	if cc := r.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
	body := r.Body.String()
	if strings.Contains(body, "rahasia") || strings.Contains(strings.ToLower(body), "email") {
		t.Fatalf("public progress leaks email: %s", body)
	}
	var env struct {
		Data struct {
			Year    string           `json:"year"`
			Classes []map[string]any `json:"classes"`
			Groups  []map[string]any `json:"groups"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.Year != "2026/2027" || len(env.Data.Classes) != 2 || len(env.Data.Groups) != 2 {
		t.Fatalf("progress body = %s", body)
	}
	var keys []string
	for k := range env.Data.Groups[0] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if strings.Join(keys, ",") != "classCode,complete,id,name,notebook,review,submission,topic,updatedAt" {
		t.Errorf("public row keys = %v", keys)
	}

	for path, want := range map[string]int{
		"/api/v1/public/progress?class=9a":              1,
		"/api/v1/public/progress?q=kelompok%202":        1,
		"/api/v1/public/progress?status=BELUM":          2,
		"/api/v1/public/progress?status=LENGKAP":        0,
		"/api/v1/public/progress?status=TUNTAS":         0,
		"/api/v1/public/progress?class=9B&status=BELUM": 1,
	} {
		r := pub(path)
		var e struct {
			Data struct {
				Groups []json.RawMessage `json:"groups"`
			} `json:"data"`
		}
		if r.code != 200 || json.Unmarshal([]byte(r.body), &e) != nil || len(e.Data.Groups) != want {
			t.Errorf("%s = %d %s, want %d groups", path, r.code, r.body, want)
		}
	}
	for _, path := range []string{"/api/v1/public/progress?status=SELESAI", "/api/v1/public/progress?q=" + strings.Repeat("a", 61), "/api/v1/public/progress?class=9%2FA"} {
		if r := pub(path); r.code != 400 || errCode(t, r.body) != "VALIDATION_ERROR" {
			t.Errorf("%s = %d %s, want 400", path, r.code, r.body)
		}
	}

	sum := pub("/api/v1/public/summary")
	var s struct {
		Data struct {
			TotalGroups  int     `json:"totalGroups"`
			None         int     `json:"none"`
			Partial      int     `json:"partial"`
			Full         int     `json:"full"`
			Complete     int     `json:"complete"`
			ManualUpdate bool    `json:"manualUpdate"`
			UpdatedAt    *string `json:"updatedAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(sum.body), &s); err != nil || sum.code != 200 {
		t.Fatalf("summary = %d %s", sum.code, sum.body)
	}
	if s.Data.TotalGroups != 2 || s.Data.None != 2 || s.Data.Partial != 0 || s.Data.Full != 0 || s.Data.Complete != 0 || !s.Data.ManualUpdate || s.Data.UpdatedAt == nil {
		t.Errorf("summary = %+v", s.Data)
	}
}
