package ephemeral

import (
	"cmp"
	"context"
	"maps"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

// GuardConfig bounds the lens calls a tracker makes from its update path. Zero fields take the
// defaults: 2 calls a second with bursts of 5, 2 at once, a minute's cooldown, 10 seconds a call.
type GuardConfig struct {
	// Rate and Burst are a token bucket over calls started, per second.
	Rate  float64 `json:"rate,omitempty"`
	Burst int     `json:"burst,omitempty"`
	// Concurrency caps the calls running at once.
	Concurrency int `json:"concurrency,omitempty"`
	// CooldownSecs is how long a key waits after an attempt before another.
	CooldownSecs int `json:"cooldownSecs,omitempty"`
	// TimeoutSecs bounds one attempt, all its pages included.
	TimeoutSecs int `json:"timeoutSecs,omitempty"`
}

// Guard runs lens calls on a pool's update path without ever making the update wait: a call that
// would exceed the rate, the concurrency or the key's cooldown is not run at all, and the caller
// keeps to its path without the lens. A full scan takes seconds and many such calls can be due in
// one batch of blocks, which waits for every pool in it; queueing them would stall every pool.
type Guard struct {
	limiter  *rate.Limiter
	slots    *semaphore.Weighted
	cooldown time.Duration
	timeout  time.Duration

	mu   sync.Mutex
	last map[string]time.Time
	// kept is how many keys the last forgetting left: the next waits for twice as many, so a map
	// whose keys are all still cooling down is not walked on every admission.
	kept int
}

func NewGuard(cfg GuardConfig) *Guard {
	return &Guard{
		limiter:  rate.NewLimiter(rate.Limit(cmp.Or(cfg.Rate, 2)), cmp.Or(cfg.Burst, 5)),
		slots:    semaphore.NewWeighted(int64(cmp.Or(cfg.Concurrency, 2))),
		cooldown: time.Duration(cmp.Or(cfg.CooldownSecs, 60)) * time.Second,
		timeout:  time.Duration(cmp.Or(cfg.TimeoutSecs, 10)) * time.Second,
		last:     map[string]time.Time{},
	}
}

// Try runs fn for key when the guard allows it, under the attempt timeout, and reports whether it
// ran. It never blocks on the guard. A nil Guard runs fn unbounded.
func (g *Guard) Try(ctx context.Context, key string, fn func(ctx context.Context) error) (bool, error) {
	if g == nil {
		return true, fn(ctx)
	}
	if g.slots != nil {
		if !g.slots.TryAcquire(1) {
			return false, nil
		}
		defer g.slots.Release(1)
	}
	if !g.admit(key, time.Now()) {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()
	return true, fn(ctx)
}

// ReadyAt is when key may next be attempted: now if it has not been lately, else once its cooldown
// ends. The rate and the concurrency may still decline it then.
func (g *Guard) ReadyAt(key string, now time.Time) time.Time {
	if g == nil {
		return now
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if at := g.last[key].Add(g.cooldown); at.After(now) {
		return at
	}
	return now
}

// admit takes a token for key unless the key is cooling down or the bucket is empty.
func (g *Guard) admit(key string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if now.Sub(g.last[key]) < g.cooldown || g.limiter != nil && !g.limiter.AllowN(now, 1) {
		return false
	}
	// Forget keys past their cooldown once there are many, so the map stays the size of a burst.
	if len(g.last) >= max(1024, 2*g.kept) {
		maps.DeleteFunc(g.last, func(_ string, at time.Time) bool { return now.Sub(at) >= g.cooldown })
		g.kept = len(g.last)
	}
	g.last[key] = now
	return true
}
