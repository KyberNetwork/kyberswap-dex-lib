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
// defaults: 2 calls a second with bursts of 5, 2 at once, a cooldown of 12 seconds that doubles up
// to 30 while a key's attempts fail, 10 seconds a call.
type GuardConfig struct {
	// Rate and Burst are a token bucket over calls started, per second.
	Rate  float64 `json:"rate,omitempty"`
	Burst int     `json:"burst,omitempty"`
	// Concurrency caps the calls running at once.
	Concurrency int `json:"concurrency,omitempty"`
	// CooldownSecs is how long a key waits after an attempt before another. Each attempt that fails
	// in a row doubles it, up to MaxCooldownSecs; one that succeeds brings it back.
	CooldownSecs    int `json:"cooldownSecs,omitempty"`
	MaxCooldownSecs int `json:"maxCooldownSecs,omitempty"`
	// TimeoutSecs bounds one attempt, all its pages included.
	TimeoutSecs int `json:"timeoutSecs,omitempty"`
}

// Guard runs lens calls on a pool's update path without ever making the update wait: a call that
// would exceed the rate, the concurrency or the key's cooldown is not run at all, and the caller
// keeps to its path without the lens. A full scan takes seconds and many such calls can be due in
// one batch of blocks, which waits for every pool in it; queueing them would stall every pool.
type Guard struct {
	limiter     *rate.Limiter
	slots       *semaphore.Weighted
	cooldown    time.Duration
	maxCooldown time.Duration
	timeout     time.Duration

	mu   sync.Mutex
	last map[string]attempt
	// kept is how many keys the last forgetting left: the next waits for twice as many, so a map
	// whose keys are all still cooling down is not walked on every admission.
	kept int
}

// attempt is a key's last attempt: when it started, how long the key waits after it, and how many
// attempts in a row have failed.
type attempt struct {
	at    time.Time
	wait  time.Duration
	fails int
}

func NewGuard(cfg GuardConfig) *Guard {
	cooldown := time.Duration(cmp.Or(cfg.CooldownSecs, 12)) * time.Second
	return &Guard{
		limiter:     rate.NewLimiter(rate.Limit(cmp.Or(cfg.Rate, 2)), cmp.Or(cfg.Burst, 5)),
		slots:       semaphore.NewWeighted(int64(cmp.Or(cfg.Concurrency, 2))),
		cooldown:    cooldown,
		maxCooldown: max(time.Duration(cmp.Or(cfg.MaxCooldownSecs, 30))*time.Second, cooldown),
		timeout:     time.Duration(cmp.Or(cfg.TimeoutSecs, 10)) * time.Second,
		last:        map[string]attempt{},
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
	err := fn(ctx)
	g.done(key, err)
	return true, err
}

// ReadyAt is when key may next be attempted: now if it has not been lately, else once its cooldown
// ends. The rate and the concurrency may still decline it then.
func (g *Guard) ReadyAt(key string, now time.Time) time.Time {
	if g == nil {
		return now
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if a := g.last[key]; a.at.Add(a.wait).After(now) {
		return a.at.Add(a.wait)
	}
	return now
}

// admit takes a token for key unless the key is cooling down or the bucket is empty. The key then
// waits at least the base cooldown, which done sets once the attempt is over.
func (g *Guard) admit(key string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.last[key]
	if now.Sub(a.at) < a.wait || g.limiter != nil && !g.limiter.AllowN(now, 1) {
		return false
	}
	// Forget keys not attempted for twice the longest cooldown once there are many, so the map
	// stays the size of a burst and a key failing again keeps its count of failures.
	if len(g.last) >= max(1024, 2*g.kept) {
		maps.DeleteFunc(g.last, func(_ string, a attempt) bool { return now.Sub(a.at) >= 2*g.maxCooldown })
		g.kept = len(g.last)
	}
	g.last[key] = attempt{at: now, wait: max(a.wait, g.cooldown), fails: a.fails}
	return true
}

// done sets how long key waits after the attempt that err ended: the base cooldown when it
// succeeded, doubled for each failure in a row before it otherwise, up to the longest cooldown.
func (g *Guard) done(key string, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.last[key]
	if err == nil {
		a.fails, a.wait = 0, g.cooldown
	} else {
		a.fails++
		// Past 2⁶ times the base, any cap a config sets is reached, and a larger shift would overflow.
		a.wait = min(g.cooldown<<min(a.fails-1, 6), g.maxCooldown)
	}
	g.last[key] = a
}
