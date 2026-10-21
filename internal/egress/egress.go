// Package egress is the single exit toward model providers (principle 1).
// Every upstream request takes a slot here first: per-provider concurrency,
// a request-rate bucket, Retry-After cooldowns and a circuit breaker. People
// are served before background callers. Nothing retries on its own: a busy
// provider returns ErrBusy with a wait, and the caller decides.
package egress

import (
	"context"
	"sync"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
)

// Limits for one provider.
type Limits struct {
	MaxConcurrent     int           // default 2
	RequestsPerMinute int           // default 30
	MaxQueue          int           // waiting callers before ErrBusy, default 16
	Burst             int           // calls allowed back to back, default RequestsPerMinute/6
	BreakerFailures   int           // consecutive failures that open the breaker, default 5
	BreakerOpen       time.Duration // default 60s
	MaxCooldown       time.Duration // cap on a Retry-After, default 10 min
}

func (l Limits) withDefaults() Limits {
	if l.MaxConcurrent <= 0 {
		l.MaxConcurrent = 2
	}
	if l.RequestsPerMinute <= 0 {
		l.RequestsPerMinute = 30
	}
	if l.MaxQueue <= 0 {
		l.MaxQueue = 16
	}
	if l.BreakerFailures <= 0 {
		l.BreakerFailures = 5
	}
	if l.BreakerOpen <= 0 {
		l.BreakerOpen = time.Minute
	}
	if l.MaxCooldown <= 0 {
		l.MaxCooldown = 10 * time.Minute
	}
	return l
}

// Outcome is how an upstream call ended.
type Outcome int

const (
	OK          Outcome = iota
	RateLimited         // 429 / Retry-After
	// LimitReached: a subscription's usage limit (Claude Code, Codex). Like
	// RateLimited, but the wait runs to the reported reset (not capped by
	// MaxCooldown) and calls refused meanwhile say it is the limit.
	LimitReached
	Failed      // 5xx, network error, timeout
	ClientError // 4xx other than 429: not the provider's fault
)

// Limiter holds a lane per provider.
type Limiter struct {
	mu    sync.Mutex
	lanes map[string]*lane
	now   func() time.Time
	lim   func(provider string) Limits
}

// New makes a limiter. limits may be nil (defaults for everyone).
func New(limits func(provider string) Limits) *Limiter {
	if limits == nil {
		limits = func(string) Limits { return Limits{} }
	}
	return &Limiter{lanes: map[string]*lane{}, now: time.Now, lim: limits}
}

type waiter struct {
	ch   chan struct{}
	won  bool // a slot was handed over (counted in inUse)
	fail bool // woken only to fail fast; no slot
}

type lane struct {
	limits     Limits
	inUse      int
	people     []*waiter
	background []*waiter
	tokens     float64
	last       time.Time
	cooldown   time.Time // until: Retry-After
	coolMsg    string    // what a call refused during the cooldown says
	failures   int
	openUntil  time.Time // breaker
}

func (l *Limiter) lane(p string) *lane {
	ln := l.lanes[p]
	if ln == nil {
		lim := l.lim(p).withDefaults()
		ln = &lane{limits: lim, tokens: float64(burst(lim)), last: l.now()}
		l.lanes[p] = ln
	}
	return ln
}

func burst(lim Limits) int {
	if lim.Burst > 0 {
		return lim.Burst
	}
	b := lim.RequestsPerMinute / 6
	if b < 1 {
		b = 1
	}
	return b
}

func (ln *lane) refill(now time.Time) {
	ln.tokens += now.Sub(ln.last).Minutes() * float64(ln.limits.RequestsPerMinute)
	if max := float64(burst(ln.limits)); ln.tokens > max {
		ln.tokens = max
	}
	ln.last = now
}

func busy(provider, key string, wait time.Duration) error {
	// The wait is rendered from RetryAfter in the reader's language.
	return &omnillm.Error{Code: omnillm.CodeBusy, Msg: key, Args: []string{provider}, RetryAfter: wait}
}

// Release ends a slot with the call's outcome. retryAfter applies to
// RateLimited.
type Release func(o Outcome, retryAfter time.Duration)

// Acquire waits for a slot for provider. It fails fast with ErrBusy when the
// provider is cooling down, the breaker is open, or the queue is full.
func (l *Limiter) Acquire(ctx context.Context, provider string, caller omnillm.Caller) (Release, error) {
	l.mu.Lock()
	ln := l.lane(provider)
	now := l.now()
	if now.Before(ln.openUntil) {
		w := ln.openUntil.Sub(now)
		l.mu.Unlock()
		return nil, busy(provider, omnillm.MsgBreakerOpen, w)
	}
	if now.Before(ln.cooldown) {
		w := ln.cooldown.Sub(now)
		msg := ln.coolKey()
		l.mu.Unlock()
		return nil, busy(provider, msg, w)
	}
	if ln.inUse < ln.limits.MaxConcurrent && len(ln.people) == 0 && (caller == omnillm.Interactive || len(ln.background) == 0) {
		pause, err := l.takeToken(ln, provider, now)
		if err != nil {
			l.mu.Unlock()
			return nil, err
		}
		ln.inUse++
		l.mu.Unlock()
		return l.paced(ctx, ln, provider, pause)
	}
	if len(ln.people)+len(ln.background) >= ln.limits.MaxQueue {
		l.mu.Unlock()
		return nil, &omnillm.Error{Code: omnillm.CodeBusy, Msg: omnillm.MsgBusyQueue, RetryAfter: 5 * time.Second}
	}
	w := &waiter{ch: make(chan struct{})}
	if caller == omnillm.Interactive {
		ln.people = append(ln.people, w)
	} else {
		ln.background = append(ln.background, w)
	}
	l.mu.Unlock()

	select {
	case <-w.ch:
		// Either a slot was handed to us (counted in inUse) or we were
		// woken to fail fast while the provider cools down.
		l.mu.Lock()
		now := l.now()
		var err error
		if w.fail {
			wait, msg := time.Second, omnillm.MsgBusyUpstream
			if now.Before(ln.openUntil) {
				wait = ln.openUntil.Sub(now)
			} else if now.Before(ln.cooldown) {
				wait, msg = ln.cooldown.Sub(now), ln.coolKey()
			}
			l.mu.Unlock()
			return nil, busy(provider, msg, wait)
		}
		var pause time.Duration
		switch {
		case now.Before(ln.openUntil):
			err = busy(provider, omnillm.MsgBreakerOpen, ln.openUntil.Sub(now))
		case now.Before(ln.cooldown):
			err = busy(provider, ln.coolKey(), ln.cooldown.Sub(now))
		default:
			pause, err = l.takeToken(ln, provider, now)
		}
		if err != nil {
			ln.inUse--
			l.handOffLocked(ln)
		}
		l.mu.Unlock()
		if err != nil {
			return nil, err
		}
		return l.paced(ctx, ln, provider, pause)
	case <-ctx.Done():
		l.mu.Lock()
		if w.fail {
			// nothing to undo
		} else if w.won {
			ln.inUse--
			l.handOffLocked(ln)
		} else {
			ln.people = remove(ln.people, w)
			ln.background = remove(ln.background, w)
		}
		l.mu.Unlock()
		return nil, &omnillm.Error{Code: omnillm.CodeBusy, Msg: omnillm.MsgBusyQueue, RetryAfter: 5 * time.Second}
	}
}

// maxPace is the longest the limiter's own pacing makes a call wait; a
// longer wait is refused with a time to come back.
const maxPace = 20 * time.Second

// takeToken takes this minute's allowance for one call. When it is used
// up and the next falls due within maxPace, the call takes it in advance
// and waits that long (pause): a person sending two messages quickly is
// slowed, not refused.
func (l *Limiter) takeToken(ln *lane, provider string, now time.Time) (pause time.Duration, err error) {
	ln.refill(now)
	if ln.tokens < 1 {
		need := time.Duration((1 - ln.tokens) / float64(ln.limits.RequestsPerMinute) * float64(time.Minute))
		if need > maxPace {
			return 0, busy(provider, omnillm.MsgBusyPace, need)
		}
		ln.tokens--
		return need, nil
	}
	ln.tokens--
	return 0, nil
}

// paced returns the slot after pause; a caller that gives up meanwhile
// hands the slot and the allowance back.
func (l *Limiter) paced(ctx context.Context, ln *lane, provider string, pause time.Duration) (Release, error) {
	if pause <= 0 {
		return l.releaser(provider), nil
	}
	t := time.NewTimer(pause)
	defer t.Stop()
	select {
	case <-t.C:
		return l.releaser(provider), nil
	case <-ctx.Done():
		l.mu.Lock()
		ln.tokens++
		ln.inUse--
		l.handOffLocked(ln)
		l.mu.Unlock()
		return nil, ctx.Err()
	}
}

// coolKey is the message for a call refused while the lane cools down.
func (ln *lane) coolKey() string {
	if ln.coolMsg == "" {
		return omnillm.MsgBusyUpstream
	}
	return ln.coolMsg
}

func remove(ws []*waiter, w *waiter) []*waiter {
	for i, x := range ws {
		if x == w {
			return append(ws[:i], ws[i+1:]...)
		}
	}
	return ws
}

// handOffLocked gives a free slot to the next waiter, people first.
func (l *Limiter) handOffLocked(ln *lane) {
	for ln.inUse < ln.limits.MaxConcurrent {
		var w *waiter
		switch {
		case len(ln.people) > 0:
			w, ln.people = ln.people[0], ln.people[1:]
		case len(ln.background) > 0:
			w, ln.background = ln.background[0], ln.background[1:]
		default:
			return
		}
		ln.inUse++
		w.won = true
		close(w.ch)
	}
}

func (l *Limiter) releaser(provider string) Release {
	var once sync.Once
	return func(o Outcome, retryAfter time.Duration) {
		once.Do(func() {
			l.mu.Lock()
			defer l.mu.Unlock()
			ln := l.lane(provider)
			now := l.now()
			ln.inUse--
			switch o {
			case OK, ClientError:
				ln.failures = 0
			case RateLimited, LimitReached:
				if retryAfter <= 0 {
					retryAfter = 30 * time.Second
				}
				ln.coolMsg = omnillm.MsgBusyUpstream
				if o == LimitReached {
					ln.coolMsg = omnillm.MsgSubscriptionLimit
				} else if retryAfter > ln.limits.MaxCooldown {
					retryAfter = ln.limits.MaxCooldown
				}
				ln.cooldown = now.Add(retryAfter)
				ln.failures++
			case Failed:
				ln.failures++
			}
			if ln.failures >= ln.limits.BreakerFailures {
				ln.openUntil = now.Add(ln.limits.BreakerOpen)
				ln.failures = 0
			}
			// While cooling down or open, waiters are woken to fail fast
			// instead of hammering the provider the moment the wait ends.
			if now.Before(ln.cooldown) || now.Before(ln.openUntil) {
				for _, w := range append(ln.people, ln.background...) {
					w.fail = true
					close(w.ch)
				}
				ln.people, ln.background = nil, nil
				return
			}
			l.handOffLocked(ln)
		})
	}
}
