package ratelimit_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pikipici/informatika-monitoring/backend/internal/auth/ratelimit"
)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time          { return c.now }
func (c *clock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func newLimiter(opts ratelimit.Options) (*ratelimit.Limiter, *clock) {
	clk := &clock{now: time.Date(2026, 11, 13, 8, 0, 0, 0, time.UTC)}
	opts.Now = clk.Now
	return ratelimit.New(opts), clk
}

func fail(l *ratelimit.Limiter, ip, user string, n int) {
	for i := 0; i < n; i++ {
		l.RecordFailure(ip, user)
	}
}

func allowed(t *testing.T, l *ratelimit.Limiter, ip, user string) (bool, time.Duration) {
	t.Helper()
	return l.Check(ip, user)
}

func TestAllowsUntilIPThresholdThenLocksWithRetryAfter(t *testing.T) {
	l, _ := newLimiter(ratelimit.Options{})

	fail(l, "203.0.113.7", "fikri", 4)
	if ok, _ := allowed(t, l, "203.0.113.7", "fikri"); !ok {
		t.Fatal("blocked after only 4 failures; threshold is 5")
	}

	l.RecordFailure("203.0.113.7", "fikri")
	ok, retry := allowed(t, l, "203.0.113.7", "fikri")
	if ok {
		t.Fatal("not blocked after 5 failures from one IP")
	}
	if retry != time.Minute {
		t.Fatalf("retryAfter = %v; want 1m for the first lockout", retry)
	}
}

func TestLockoutEscalatesAndIsCapped(t *testing.T) {
	l, clk := newLimiter(ratelimit.Options{})
	ip := "203.0.113.7"

	want := []time.Duration{1 * time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 15 * time.Minute, 15 * time.Minute}
	for i, w := range want {
		fail(l, ip, "fikri", 5)
		ok, retry := allowed(t, l, ip, "fikri")
		if ok || retry != w {
			t.Fatalf("strike %d: allowed=%v retry=%v; want blocked for %v", i+1, ok, retry, w)
		}
		clk.Advance(w) // wait out the lock, then keep failing within the window
		if ok, _ := allowed(t, l, ip, "fikri"); !ok {
			t.Fatalf("strike %d: still blocked after waiting %v", i+1, w)
		}
	}
}

func TestLockoutIsTemporaryNotPermanent(t *testing.T) {
	l, clk := newLimiter(ratelimit.Options{})
	fail(l, "203.0.113.7", "fikri", 5)

	clk.Advance(time.Minute)
	if ok, _ := allowed(t, l, "203.0.113.7", "fikri"); !ok {
		t.Fatal("still blocked after the lockout elapsed")
	}
}

func TestRetryAfterRoundsUpToWholeSeconds(t *testing.T) {
	l, clk := newLimiter(ratelimit.Options{})
	fail(l, "203.0.113.7", "fikri", 5)
	clk.Advance(30*time.Second + 400*time.Millisecond)

	ok, retry := allowed(t, l, "203.0.113.7", "fikri")
	if ok || retry != 30*time.Second {
		t.Fatalf("allowed=%v retry=%v; want blocked, 30s (rounded up from 29.6s)", ok, retry)
	}
}

func TestOneIPLockDoesNotBlockOtherIPs(t *testing.T) {
	l, _ := newLimiter(ratelimit.Options{})
	fail(l, "203.0.113.7", "fikri", 5)

	if ok, _ := allowed(t, l, "198.51.100.9", "fikri"); !ok {
		t.Fatal("a different IP was blocked by someone else's failures")
	}
}

func TestDistributedAttackOnOneAccountLocksThatAccountOnly(t *testing.T) {
	l, _ := newLimiter(ratelimit.Options{})

	// 10 failures, each from a different IP, so no single IP reaches its limit.
	for i := 0; i < 10; i++ {
		l.RecordFailure(fmt.Sprintf("203.0.113.%d", i+1), "fikri")
	}

	ok, retry := allowed(t, l, "198.51.100.200", "fikri")
	if ok || retry <= 0 {
		t.Fatalf("account not locked after 10 distributed failures: allowed=%v retry=%v", ok, retry)
	}
	if ok, _ := allowed(t, l, "198.51.100.200", "someone-else"); !ok {
		t.Fatal("unrelated account was locked")
	}
}

func TestUsernameIsNormalized(t *testing.T) {
	l, _ := newLimiter(ratelimit.Options{})
	for i := 0; i < 10; i++ {
		user := []string{"Fikri", " fikri ", "FIKRI", "fikri", "Fikri"}[i%5]
		l.RecordFailure(fmt.Sprintf("203.0.113.%d", i+1), user)
	}
	if ok, _ := allowed(t, l, "198.51.100.5", "fikri"); ok {
		t.Fatal("case/space variants of one username were counted separately")
	}
}

func TestSuccessClearsCounters(t *testing.T) {
	l, _ := newLimiter(ratelimit.Options{})
	fail(l, "203.0.113.7", "fikri", 4)
	l.RecordSuccess("203.0.113.7", "fikri")

	fail(l, "203.0.113.7", "fikri", 4)
	if ok, _ := allowed(t, l, "203.0.113.7", "fikri"); !ok {
		t.Fatal("failures before a success still counted toward the lockout")
	}
}

func TestFailuresOutsideWindowDoNotAccumulate(t *testing.T) {
	l, clk := newLimiter(ratelimit.Options{})
	fail(l, "203.0.113.7", "fikri", 4)
	clk.Advance(16 * time.Minute)
	fail(l, "203.0.113.7", "fikri", 4)

	if ok, _ := allowed(t, l, "203.0.113.7", "fikri"); !ok {
		t.Fatal("old failures outside the 15 minute window were counted")
	}
}

func TestIPv6AddressesInOneSlash64ShareALimit(t *testing.T) {
	l, _ := newLimiter(ratelimit.Options{})
	for i := 1; i <= 5; i++ {
		l.RecordFailure(fmt.Sprintf("2001:db8:1:2::%x", i), fmt.Sprintf("user%d", i))
	}
	if ok, _ := allowed(t, l, "2001:db8:1:2:ffff::1", "another"); ok {
		t.Fatal("attacker rotating addresses inside one /64 evaded the IP limit")
	}
	if ok, _ := allowed(t, l, "2001:db8:1:3::1", "another"); !ok {
		t.Fatal("a different /64 was blocked")
	}
}

func TestInvalidIPSharesOneBucket(t *testing.T) {
	l, _ := newLimiter(ratelimit.Options{})
	for i := 0; i < 5; i++ {
		l.RecordFailure(fmt.Sprintf("garbage-%d", i), fmt.Sprintf("u%d", i))
	}
	if ok, _ := allowed(t, l, "also-garbage", "x"); ok {
		t.Fatal("unparseable client addresses must share a bucket, not bypass the limit")
	}
}

func TestMemoryIsBoundedUnderFlood(t *testing.T) {
	l, _ := newLimiter(ratelimit.Options{MaxKeys: 100})

	// Lock one real attacker first; its state must survive the flood.
	fail(l, "203.0.113.7", "fikri", 5)

	for i := 0; i < 20000; i++ {
		l.RecordFailure(fmt.Sprintf("10.%d.%d.%d", i/65536%256, i/256%256, i%256), fmt.Sprintf("random-user-%d", i))
	}

	ips, accounts := l.Size()
	if ips > 100 || accounts > 100 {
		t.Fatalf("tracked keys = %d IPs, %d accounts; want <= 100 each", ips, accounts)
	}
	if ok, _ := allowed(t, l, "203.0.113.7", "fikri"); ok {
		t.Fatal("an active lockout was evicted by a flood of junk keys")
	}
}

func TestConcurrentUseIsSafe(t *testing.T) {
	l, _ := newLimiter(ratelimit.Options{})

	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				ip := fmt.Sprintf("203.0.113.%d", g)
				l.Check(ip, "fikri")
				l.RecordFailure(ip, "fikri")
				if i%50 == 0 {
					l.RecordSuccess(ip, "fikri")
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestGateLimitsConcurrency(t *testing.T) {
	g := ratelimit.NewGate(2)
	ctx := context.Background()

	if err := g.Acquire(ctx); err != nil {
		t.Fatal(err)
	}
	if err := g.Acquire(ctx); err != nil {
		t.Fatal(err)
	}

	short, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if err := g.Acquire(short); err == nil {
		t.Fatal("third Acquire succeeded with a gate of 2")
	}

	g.Release()
	again, cancel2 := context.WithTimeout(ctx, time.Second)
	defer cancel2()
	if err := g.Acquire(again); err != nil {
		t.Fatalf("Acquire after Release failed: %v", err)
	}
}

func TestGateRejectsNonPositiveSize(t *testing.T) {
	g := ratelimit.NewGate(0)
	short, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := g.Acquire(short); err != nil {
		t.Fatalf("NewGate(0) must fall back to a usable size of 1: %v", err)
	}
}
