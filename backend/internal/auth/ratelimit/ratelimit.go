// Package ratelimit slows down password guessing against the admin login.
//
// It tracks failures per client address and per account, in memory (the app is
// a single process). Lockouts are temporary and escalate (1m, 2m, 4m, ... up
// to a cap) so an attacker cannot lock the teacher out permanently. The IP
// limit is lower than the account limit, so one attacker is stopped by IP
// before the account is affected; only a distributed attack reaches the
// account limit.
package ratelimit

import (
	"context"
	"net/netip"
	"strings"
	"sync"
	"time"
)

const (
	defaultIPMaxFailures      = 5
	defaultAccountMaxFailures = 10
	defaultWindow             = 15 * time.Minute
	defaultBaseLockout        = time.Minute
	defaultMaxLockout         = 15 * time.Minute
	defaultMaxKeys            = 10000

	maxUsernameRunes = 64
	invalidIPKey     = "invalid"
)

// Options configures a Limiter. Zero values use the defaults above.
type Options struct {
	IPMaxFailures      int
	AccountMaxFailures int
	// Window is how long failures are remembered without new ones.
	Window      time.Duration
	BaseLockout time.Duration
	MaxLockout  time.Duration
	// MaxKeys bounds memory: at most this many IPs and this many accounts are tracked.
	MaxKeys int
	// Now overrides the clock, for tests.
	Now func() time.Time
}

type entry struct {
	failures    int
	strikes     int
	lastFailure time.Time
	lockUntil   time.Time
}

// Limiter is safe for concurrent use.
type Limiter struct {
	opts Options

	mu    sync.Mutex
	ips   map[string]*entry
	accts map[string]*entry
}

// New returns a Limiter.
func New(opts Options) *Limiter {
	if opts.IPMaxFailures <= 0 {
		opts.IPMaxFailures = defaultIPMaxFailures
	}
	if opts.AccountMaxFailures <= 0 {
		opts.AccountMaxFailures = defaultAccountMaxFailures
	}
	if opts.Window <= 0 {
		opts.Window = defaultWindow
	}
	if opts.BaseLockout <= 0 {
		opts.BaseLockout = defaultBaseLockout
	}
	if opts.MaxLockout <= 0 {
		opts.MaxLockout = defaultMaxLockout
	}
	if opts.MaxKeys <= 0 {
		opts.MaxKeys = defaultMaxKeys
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Limiter{opts: opts, ips: map[string]*entry{}, accts: map[string]*entry{}}
}

// Check reports whether a login attempt may proceed. When it may not,
// retryAfter says how long to wait, rounded up to whole seconds.
func (l *Limiter) Check(ip, username string) (allowed bool, retryAfter time.Duration) {
	ipKey, userKey := normalizeIP(ip), normalizeUser(username)
	now := l.opts.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	var remaining time.Duration
	for _, e := range []*entry{l.ips[ipKey], l.accts[userKey]} {
		if e != nil {
			if d := e.lockUntil.Sub(now); d > remaining {
				remaining = d
			}
		}
	}
	if remaining <= 0 {
		return true, 0
	}
	return false, ceilSecond(remaining)
}

// RecordFailure counts a failed attempt, including attempts for usernames
// that do not exist, so the limiter reveals nothing about valid accounts.
func (l *Limiter) RecordFailure(ip, username string) {
	ipKey, userKey := normalizeIP(ip), normalizeUser(username)
	now := l.opts.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.record(l.ips, ipKey, l.opts.IPMaxFailures, now)
	l.record(l.accts, userKey, l.opts.AccountMaxFailures, now)
}

// RecordSuccess clears the counters for this client and account.
func (l *Limiter) RecordSuccess(ip, username string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.ips, normalizeIP(ip))
	delete(l.accts, normalizeUser(username))
}

// Size returns how many IP and account entries are currently tracked.
func (l *Limiter) Size() (ips, accounts int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.ips), len(l.accts)
}

func (l *Limiter) record(m map[string]*entry, key string, threshold int, now time.Time) {
	e := m[key]
	if e == nil {
		if !l.makeRoom(m, now) {
			return // table full of active lockouts; stay bounded rather than evict them
		}
		e = &entry{}
		m[key] = e
	}

	if e.lockUntil.After(now) {
		return // already locked; extra failures neither extend nor escalate
	}
	if !e.lastFailure.IsZero() && now.Sub(e.lastFailure) > l.opts.Window {
		e.failures, e.strikes = 0, 0
	}

	e.failures++
	e.lastFailure = now
	if e.failures >= threshold {
		e.strikes++
		e.failures = 0
		e.lockUntil = now.Add(l.lockoutFor(e.strikes))
	}
}

func (l *Limiter) lockoutFor(strikes int) time.Duration {
	d := l.opts.BaseLockout
	for i := 1; i < strikes && d < l.opts.MaxLockout; i++ {
		d *= 2
	}
	if d > l.opts.MaxLockout {
		d = l.opts.MaxLockout
	}
	return d
}

// makeRoom ensures m can take one more key: it drops expired entries first,
// then the oldest entry that is not locked. It reports false if every entry is
// an active lockout.
func (l *Limiter) makeRoom(m map[string]*entry, now time.Time) bool {
	if len(m) < l.opts.MaxKeys {
		return true
	}

	for k, e := range m {
		if !e.lockUntil.After(now) && now.Sub(e.lastFailure) > l.opts.Window {
			delete(m, k)
		}
	}
	if len(m) < l.opts.MaxKeys {
		return true
	}

	var oldestKey string
	var oldest time.Time
	for k, e := range m {
		if e.lockUntil.After(now) {
			continue
		}
		if oldestKey == "" || e.lastFailure.Before(oldest) {
			oldestKey, oldest = k, e.lastFailure
		}
	}
	if oldestKey == "" {
		return false
	}
	delete(m, oldestKey)
	return true
}

// normalizeIP maps an address to its rate-limit bucket: IPv4 as-is, IPv6 by
// /64 (one customer commonly controls a whole /64), and anything unparseable
// to a single shared bucket so a bad value can never bypass the limit.
func normalizeIP(s string) string {
	addr, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil {
		return invalidIPKey
	}
	addr = addr.Unmap()
	if addr.Is6() {
		p, err := addr.Prefix(64)
		if err != nil {
			return invalidIPKey
		}
		return p.String()
	}
	return addr.String()
}

func normalizeUser(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if r := []rune(s); len(r) > maxUsernameRunes {
		s = string(r[:maxUsernameRunes])
	}
	return s
}

func ceilSecond(d time.Duration) time.Duration {
	return (d + time.Second - 1) / time.Second * time.Second
}

// Gate caps how many expensive operations (password hashing, 64 MiB each) run
// at once, so a burst of logins cannot exhaust the server's memory.
type Gate struct{ slots chan struct{} }

// NewGate returns a Gate allowing n concurrent holders (minimum 1).
func NewGate(n int) *Gate {
	if n < 1 {
		n = 1
	}
	return &Gate{slots: make(chan struct{}, n)}
}

// Acquire waits for a slot or until ctx is done.
func (g *Gate) Acquire(ctx context.Context) error {
	select {
	case g.slots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release frees a slot obtained with Acquire.
func (g *Gate) Release() { <-g.slots }
