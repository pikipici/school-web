package httpapi_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAdminPurgeGroup(t *testing.T) {
	p := newProgressFixture(t, fixtureOpts{persistAudit: true})
	g := p.group("Kelompok Hapus")
	base := fmt.Sprintf("/api/v1/admin/groups/%d", g)
	at := p.clk.now.Add(-time.Hour).Format(time.RFC3339)
	if r := p.admin(http.MethodPost, base+"/submissions", fmt.Sprintf(`{"roundNumber":0,"receivedAt":%q,"ppt":true}`, at)); r.code != 201 {
		t.Fatalf("round = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodDelete, base, ""); r.code != 409 || errCode(t, r.body) != "IN_USE" {
		t.Fatalf("plain delete = %d %s", r.code, r.body)
	}

	// Anonymous and CSRF-less requests are refused before anything else.
	if rec := p.do(req{method: http.MethodPost, path: base + "/purge", body: `{"confirmName":"Kelompok Hapus"}`, origin: p.origin}); rec.Code != http.StatusUnauthorized {
		t.Errorf("anonymous purge = %d", rec.Code)
	}
	if rec := p.do(req{method: http.MethodPost, path: base + "/purge", body: `{"confirmName":"Kelompok Hapus"}`, cookie: p.cookie, origin: p.origin}); rec.Code != http.StatusForbidden {
		t.Errorf("purge without CSRF = %d", rec.Code)
	}

	if r := p.admin(http.MethodPost, base+"/purge", `{"confirmName":"kelompok hapus"}`); r.code != 422 || errCode(t, r.body) != "CONFIRM_MISMATCH" {
		t.Errorf("wrong name = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodPost, base+"/purge", `{}`); r.code != 422 || errCode(t, r.body) != "CONFIRM_MISMATCH" {
		t.Errorf("missing name = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodPost, "/api/v1/admin/groups/999999/purge", `{"confirmName":"x"}`); r.code != 404 {
		t.Errorf("missing group = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodGet, base, ""); r.code != 200 {
		t.Fatalf("group gone after refused purges: %d", r.code)
	}

	if r := p.admin(http.MethodPost, base+"/purge", `{"confirmName":"Kelompok Hapus"}`); r.code != 204 {
		t.Fatalf("purge = %d %s", r.code, r.body)
	}
	if r := p.admin(http.MethodGet, base, ""); r.code != 404 {
		t.Errorf("after purge GET = %d", r.code)
	}
	if body := p.do(req{method: http.MethodGet, path: "/api/v1/public/progress"}).Body.String(); strings.Contains(body, "Kelompok Hapus") {
		t.Errorf("public still lists purged group: %s", body)
	}
	var n int
	if err := p.pool.QueryRow(t.Context(), `select count(*) from audit_log where action = 'progress.group_purged' and detail->>'title' = 'Kelompok Hapus' and detail->>'rounds' = '1'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("purge audit rows = %d, %v", n, err)
	}
}
