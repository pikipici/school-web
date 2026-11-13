package web_test

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/web"
)

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func get(t *testing.T, h http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(method, target, nil))
	return r
}

// directive returns the value of one CSP directive, or "" if absent.
func directive(policy, name string) string {
	for _, part := range strings.Split(policy, ";") {
		fields := strings.Fields(part)
		if len(fields) > 0 && fields[0] == name {
			return strings.Join(fields[1:], " ")
		}
	}
	return ""
}

const page = `<!DOCTYPE html><html lang="id"><head>
<script src="/_next/static/chunks/a.js" async=""></script>
<SCRIPT>self.__next_f=self.__next_f||[]</SCRIPT>
<script type="text/javascript" data-x="a>b">self.__next_f.push([1,"x"])</script>
<script src="/b.js">ignored body</script>
<script></script>
</head><body>ok</body></html>`

func TestHTMLGetsHashBasedPolicy(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.html", page)
	h := web.New(dir)

	r := get(t, h, http.MethodGet, "/")
	if r.Code != 200 || !strings.Contains(r.Body.String(), "ok") {
		t.Fatalf("GET / = %d %q", r.Code, r.Body.String())
	}
	policy := r.Header().Get("Content-Security-Policy")
	if policy == "" {
		t.Fatal("no Content-Security-Policy header on HTML")
	}
	if r.Header().Get("Content-Security-Policy-Report-Only") != "" {
		t.Fatal("enforce mode must not also send Report-Only")
	}

	script := directive(policy, "script-src")
	for _, want := range []string{"'self'", sha("self.__next_f=self.__next_f||[]"), sha(`self.__next_f.push([1,"x"])`), sha("")} {
		if !strings.Contains(script, want) {
			t.Errorf("script-src %q lacks %s", script, want)
		}
	}
	if strings.Contains(script, sha("ignored body")) {
		t.Error("script with src must not be hashed: its body never runs")
	}
	for _, bad := range []string{"'unsafe-inline'", "'unsafe-eval'", "*", "data:", "http:", "https:"} {
		if strings.Contains(" "+script+" ", " "+bad+" ") {
			t.Errorf("script-src must not contain %s: %q", bad, script)
		}
	}

	want := map[string]string{
		"default-src":     "'self'",
		"object-src":      "'none'",
		"base-uri":        "'self'",
		"form-action":     "'self'",
		"frame-ancestors": "'none'",
		"connect-src":     "'self'",
		"font-src":        "'self'",
		"img-src":         "'self' data:",
		"style-src":       "'self' 'unsafe-inline'",
	}
	for name, value := range want {
		if got := directive(policy, name); got != value {
			t.Errorf("%s = %q; want %q", name, got, value)
		}
	}
}

func TestHashMatchesBrowserAlgorithmForUnicodeAndNewlines(t *testing.T) {
	dir := t.TempDir()
	body := "\n  console.log(\"héllo — 日本\");\n"
	writeFile(t, dir, "index.html", "<html><script>"+body+"</script></html>")

	policy := get(t, web.New(dir), http.MethodGet, "/").Header().Get("Content-Security-Policy")
	if !strings.Contains(directive(policy, "script-src"), sha(body)) {
		t.Fatalf("hash must cover the exact UTF-8 script text including whitespace; policy %q", policy)
	}
}

func TestNotFoundPageAndHeadAndRangeKeepPolicy(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.html", "<html><script>a()</script>HOME</html>")
	writeFile(t, dir, "404.html", "<html><script>nf()</script>NOT FOUND</html>")
	h := web.New(dir)

	nf := get(t, h, http.MethodGet, "/missing")
	if nf.Code != 404 || !strings.Contains(directive(nf.Header().Get("Content-Security-Policy"), "script-src"), sha("nf()")) {
		t.Fatalf("404 = %d, policy %q", nf.Code, nf.Header().Get("Content-Security-Policy"))
	}

	head := get(t, h, http.MethodHead, "/")
	if head.Code != 200 || head.Body.Len() != 0 || !strings.Contains(head.Header().Get("Content-Security-Policy"), sha("a()")) {
		t.Fatalf("HEAD = %d body %q policy %q", head.Code, head.Body.String(), head.Header().Get("Content-Security-Policy"))
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Range", "bytes=0-5")
	partial := httptest.NewRecorder()
	h.ServeHTTP(partial, req)
	if partial.Code != 206 || partial.Body.String() != "<html>" || partial.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("Range = %d %q", partial.Code, partial.Body.String())
	}
}

func TestNonHTMLHasNoPolicy(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.html", "<html></html>")
	writeFile(t, dir, "app.js", "console.log(1)")
	writeFile(t, dir, "data.txt", "<script>x()</script>")
	h := web.New(dir)
	for _, p := range []string{"/app.js", "/data.txt"} {
		r := get(t, h, http.MethodGet, p)
		if r.Code != 200 || r.Header().Get("Content-Security-Policy") != "" {
			t.Errorf("%s: code %d policy %q; non-HTML responses carry no page policy", p, r.Code, r.Header().Get("Content-Security-Policy"))
		}
	}
}

func TestPolicyFollowsFileChanges(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.html", "<html><script>one()</script></html>")
	h := web.New(dir)
	first := get(t, h, http.MethodGet, "/").Header().Get("Content-Security-Policy")

	writeFile(t, dir, "index.html", "<html><script>two(); two();</script></html>")
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(filepath.Join(dir, "index.html"), later, later); err != nil {
		t.Fatal(err)
	}
	second := get(t, h, http.MethodGet, "/").Header().Get("Content-Security-Policy")

	if !strings.Contains(first, sha("one()")) || !strings.Contains(second, sha("two(); two();")) || strings.Contains(second, sha("one()")) {
		t.Fatalf("policy did not follow the changed file:\nfirst  %q\nsecond %q", first, second)
	}
}

func TestPolicyModes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.html", "<html><script>a()</script></html>")

	ro := get(t, web.NewWithCSP(dir, web.CSPReportOnly), http.MethodGet, "/")
	if ro.Header().Get("Content-Security-Policy") != "" || !strings.Contains(ro.Header().Get("Content-Security-Policy-Report-Only"), sha("a()")) {
		t.Fatalf("report-only headers = %#v", ro.Header())
	}

	off := get(t, web.NewWithCSP(dir, web.CSPOff), http.MethodGet, "/")
	if off.Header().Get("Content-Security-Policy") != "" || off.Header().Get("Content-Security-Policy-Report-Only") != "" {
		t.Fatalf("off mode still sends a policy: %#v", off.Header())
	}

	enforce := get(t, web.NewWithCSP(dir, web.CSPEnforce), http.MethodGet, "/")
	if !strings.Contains(enforce.Header().Get("Content-Security-Policy"), sha("a()")) {
		t.Fatalf("enforce headers = %#v", enforce.Header())
	}
}

func TestParseCSPMode(t *testing.T) {
	for in, want := range map[string]web.CSPMode{"": web.CSPEnforce, "enforce": web.CSPEnforce, " Enforce ": web.CSPEnforce, "report-only": web.CSPReportOnly, "off": web.CSPOff} {
		got, err := web.ParseCSPMode(in)
		if err != nil || got != want {
			t.Errorf("ParseCSPMode(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"report", "true", "none", "disabled"} {
		if _, err := web.ParseCSPMode(bad); err == nil {
			t.Errorf("ParseCSPMode(%q) accepted an unknown mode", bad)
		}
	}
}
