package egress

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
)

func fast() *Limiter {
	return New(func(string) Limits { return Limits{MaxConcurrent: 2, RequestsPerMinute: 6000, MaxQueue: 4} })
}

func TestConcurrencyCap(t *testing.T) {
	l := fast()
	var cur, peak int32
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rel, err := l.Acquire(context.Background(), "p", omnillm.Background)
			if err != nil {
				t.Error(err)
				return
			}
			n := atomic.AddInt32(&cur, 1)
			for {
				p := atomic.LoadInt32(&peak)
				if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
					break
				}
			}
			time.Sleep(30 * time.Millisecond)
			atomic.AddInt32(&cur, -1)
			rel(OK, 0)
		}()
	}
	wg.Wait()
	if peak > 2 {
		t.Fatalf("peak concurrency %d > 2", peak)
	}
	if l.lanes["p"].inUse != 0 {
		t.Fatalf("slots leaked: %d", l.lanes["p"].inUse)
	}
}

func TestPeopleFirst(t *testing.T) {
	l := New(func(string) Limits { return Limits{MaxConcurrent: 1, RequestsPerMinute: 6000, MaxQueue: 8} })
	hold, _ := l.Acquire(context.Background(), "p", omnillm.Background)
	order := make(chan string, 2)
	go func() {
		r, err := l.Acquire(context.Background(), "p", omnillm.Background)
		if err == nil {
			order <- "background"
			r(OK, 0)
		}
	}()
	time.Sleep(20 * time.Millisecond)
	go func() {
		r, err := l.Acquire(context.Background(), "p", omnillm.Interactive)
		if err == nil {
			order <- "person"
			r(OK, 0)
		}
	}()
	time.Sleep(20 * time.Millisecond)
	hold(OK, 0)
	if first := <-order; first != "person" {
		t.Fatalf("a person must go before a waiting background call, got %s first", first)
	}
	<-order
}

func TestRetryAfterCoolsDownAndFailsFast(t *testing.T) {
	l := fast()
	rel, _ := l.Acquire(context.Background(), "p", omnillm.Background)
	rel(RateLimited, 30*time.Second)
	start := time.Now()
	_, err := l.Acquire(context.Background(), "p", omnillm.Interactive)
	var e *omnillm.Error
	if !errors.As(err, &e) || e.Code != omnillm.CodeBusy {
		t.Fatalf("want busy, got %v", err)
	}
	if e.RetryAfter < 29*time.Second || e.RetryAfter > 30*time.Second {
		t.Fatalf("RetryAfter %s", e.RetryAfter)
	}
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("cooldown must fail fast, not wait")
	}
	// Other providers are unaffected.
	r2, err := l.Acquire(context.Background(), "other", omnillm.Background)
	if err != nil {
		t.Fatal(err)
	}
	r2(OK, 0)
}

func TestRetryAfterIsCapped(t *testing.T) {
	l := fast()
	rel, _ := l.Acquire(context.Background(), "p", omnillm.Background)
	rel(RateLimited, 48*time.Hour)
	_, err := l.Acquire(context.Background(), "p", omnillm.Background)
	if e := omnillm.AsError(err); e.RetryAfter > 10*time.Minute {
		t.Fatalf("cooldown not capped: %s", e.RetryAfter)
	}
}

func TestWaitersWokenDuringCooldownDoNotProceed(t *testing.T) {
	l := New(func(string) Limits { return Limits{MaxConcurrent: 1, RequestsPerMinute: 6000, MaxQueue: 8} })
	hold, _ := l.Acquire(context.Background(), "p", omnillm.Background)
	errs := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			r, err := l.Acquire(context.Background(), "p", omnillm.Background)
			if err == nil {
				r(OK, 0)
			}
			errs <- err
		}()
	}
	time.Sleep(30 * time.Millisecond)
	hold(RateLimited, time.Minute)
	for i := 0; i < 3; i++ {
		if err := <-errs; !errors.Is(err, omnillm.ErrBusy) {
			t.Fatalf("waiter proceeded during cooldown: %v", err)
		}
	}
	if l.lanes["p"].inUse != 0 {
		t.Fatalf("slots leaked: %d", l.lanes["p"].inUse)
	}
}

func TestBreakerOpens(t *testing.T) {
	l := fast()
	for i := 0; i < 5; i++ {
		r, err := l.Acquire(context.Background(), "p", omnillm.Background)
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		r(Failed, 0)
	}
	_, err := l.Acquire(context.Background(), "p", omnillm.Background)
	if e := omnillm.AsError(err); e == nil || e.Msg != omnillm.MsgBreakerOpen {
		t.Fatalf("breaker should be open, got %v", err)
	}
}

func TestClientErrorsDoNotTripBreaker(t *testing.T) {
	l := fast()
	for i := 0; i < 10; i++ {
		r, err := l.Acquire(context.Background(), "p", omnillm.Background)
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		r(ClientError, 0)
	}
}

func TestQueueFullIsBusy(t *testing.T) {
	l := New(func(string) Limits { return Limits{MaxConcurrent: 1, RequestsPerMinute: 6000, MaxQueue: 2} })
	hold, _ := l.Acquire(context.Background(), "p", omnillm.Background)
	defer hold(OK, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for i := 0; i < 2; i++ {
		go func() { _, _ = l.Acquire(ctx, "p", omnillm.Background) }()
	}
	time.Sleep(30 * time.Millisecond)
	if _, err := l.Acquire(context.Background(), "p", omnillm.Background); !errors.Is(err, omnillm.ErrBusy) {
		t.Fatalf("full queue must be busy, got %v", err)
	}
}

func TestCancelledWaiterLeavesNoSlot(t *testing.T) {
	l := New(func(string) Limits { return Limits{MaxConcurrent: 1, RequestsPerMinute: 6000, MaxQueue: 8} })
	hold, _ := l.Acquire(context.Background(), "p", omnillm.Background)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := l.Acquire(ctx, "p", omnillm.Background); err == nil {
		t.Fatal("should time out")
	}
	hold(OK, 0)
	r, err := l.Acquire(context.Background(), "p", omnillm.Background)
	if err != nil {
		t.Fatal(err)
	}
	r(OK, 0)
	if l.lanes["p"].inUse != 0 || len(l.lanes["p"].background) != 0 {
		t.Fatal("cancelled waiter left state behind")
	}
}

func TestRateBucket(t *testing.T) {
	l := New(func(string) Limits { return Limits{MaxConcurrent: 10, RequestsPerMinute: 6, MaxQueue: 8} })
	r, err := l.Acquire(context.Background(), "p", omnillm.Background)
	if err != nil {
		t.Fatal(err)
	}
	r(OK, 0)
	if _, err := l.Acquire(context.Background(), "p", omnillm.Background); !errors.Is(err, omnillm.ErrBusy) {
		t.Fatalf("second request within the burst of 1 should be busy, got %v", err)
	}
}

func TestDoubleReleaseIsHarmless(t *testing.T) {
	l := fast()
	r, _ := l.Acquire(context.Background(), "p", omnillm.Background)
	r(OK, 0)
	r(OK, 0)
	if l.lanes["p"].inUse != 0 {
		t.Fatalf("inUse %d", l.lanes["p"].inUse)
	}
}
