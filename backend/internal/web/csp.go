package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

// CSPMode selects how the Content-Security-Policy header is sent on HTML pages.
type CSPMode int

const (
	// CSPEnforce sends Content-Security-Policy (the default).
	CSPEnforce CSPMode = iota
	// CSPReportOnly sends Content-Security-Policy-Report-Only, so browsers log
	// violations in the console without blocking anything.
	CSPReportOnly
	// CSPOff sends no policy.
	CSPOff
)

// ParseCSPMode reads CSP_MODE. Blank means enforce; unknown values are errors
// so a typo cannot silently switch the policy off.
func ParseCSPMode(value string) (CSPMode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "enforce":
		return CSPEnforce, nil
	case "report-only":
		return CSPReportOnly, nil
	case "off":
		return CSPOff, nil
	default:
		return CSPEnforce, errors.New("unknown CSP mode")
	}
}

// The static Next.js export puts its bootstrap data in inline <script> tags.
// Instead of allowing every inline script ('unsafe-inline'), each page gets a
// policy that allows exactly the inline scripts it contains, by SHA-256 hash.
// Styles keep 'unsafe-inline': Next's built-in 404 page uses style attributes,
// which hashes cannot cover, and injected CSS cannot run code.
const policyTail = "default-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"manifest-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'"

type cacheKey struct {
	name    string
	size    int64
	modTime time.Time
}

// policyCache remembers the policy per file version; releases are immutable,
// but keying on size and mtime keeps it correct if a file ever changes.
type policyCache struct {
	mu      sync.Mutex
	entries map[cacheKey]string
}

const maxCachedPolicies = 512

func (c *policyCache) get(key cacheKey, body []byte) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.entries[key]; ok {
		return p
	}
	if c.entries == nil || len(c.entries) >= maxCachedPolicies {
		c.entries = make(map[cacheKey]string)
	}
	p := buildPolicy(body)
	c.entries[key] = p
	return p
}

func buildPolicy(body []byte) string {
	hashes := inlineScriptHashes(body)
	var b strings.Builder
	b.WriteString("script-src 'self'")
	for _, h := range hashes {
		b.WriteString(" 'sha256-")
		b.WriteString(h)
		b.WriteString("'")
	}
	b.WriteString("; ")
	b.WriteString(policyTail)
	return b.String()
}

// inlineScriptHashes returns the sorted, de-duplicated base64 SHA-256 of the
// text of every <script> element without a src attribute, matching what
// browsers hash. The scanner handles quoted attribute values (which may
// contain '>') and is case-insensitive, as HTML is.
func inlineScriptHashes(body []byte) []string {
	lower := bytes.ToLower(body)
	seen := map[string]bool{}
	var out []string
	for i := 0; ; {
		start := indexTag(lower, i, "<script")
		if start < 0 {
			break
		}
		tagEnd, hasSrc := scanOpenTag(lower, start+len("<script"))
		if tagEnd < 0 {
			break
		}
		closeAt := bytes.Index(lower[tagEnd:], []byte("</script"))
		if closeAt < 0 {
			break
		}
		content := body[tagEnd : tagEnd+closeAt]
		i = tagEnd + closeAt + len("</script")
		if hasSrc {
			continue
		}
		sum := sha256.Sum256(content)
		h := base64.StdEncoding.EncodeToString(sum[:])
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	sort.Strings(out)
	return out
}

// indexTag finds name (e.g. "<script") at or after from, followed by a tag
// boundary, so "<scripts>" or "<scripting" do not match.
func indexTag(lower []byte, from int, name string) int {
	for from < len(lower) {
		at := bytes.Index(lower[from:], []byte(name))
		if at < 0 {
			return -1
		}
		at += from
		next := at + len(name)
		if next >= len(lower) {
			return -1
		}
		switch lower[next] {
		case ' ', '\t', '\n', '\r', '\f', '/', '>':
			return at
		}
		from = next
	}
	return -1
}

// scanOpenTag walks the attributes of an open tag starting at pos and returns
// the index just after its closing '>' and whether a src attribute exists.
func scanOpenTag(lower []byte, pos int) (end int, hasSrc bool) {
	for pos < len(lower) {
		c := lower[pos]
		switch {
		case c == '>':
			return pos + 1, hasSrc
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '/':
			pos++
		default:
			nameStart := pos
			for pos < len(lower) && !isAttrNameEnd(lower[pos]) {
				pos++
			}
			if string(lower[nameStart:pos]) == "src" {
				hasSrc = true
			}
			for pos < len(lower) && isSpace(lower[pos]) {
				pos++
			}
			if pos < len(lower) && lower[pos] == '=' {
				pos++
				for pos < len(lower) && isSpace(lower[pos]) {
					pos++
				}
				if pos < len(lower) && (lower[pos] == '"' || lower[pos] == '\'') {
					q := lower[pos]
					closeQ := bytes.IndexByte(lower[pos+1:], q)
					if closeQ < 0 {
						return -1, false
					}
					pos += closeQ + 2
				} else {
					for pos < len(lower) && !isSpace(lower[pos]) && lower[pos] != '>' {
						pos++
					}
				}
			}
		}
	}
	return -1, false
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }

func isAttrNameEnd(c byte) bool { return isSpace(c) || c == '/' || c == '>' || c == '=' }
