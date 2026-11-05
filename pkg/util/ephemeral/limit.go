package ephemeral

import (
	"context"
	"reflect"
	"sync"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

// NodeLimit bounds the lens calls made through one node, whichever tracker makes them. A lens call
// runs for seconds on the node, and a node or a gateway with a low cap refuses many at once.
type NodeLimit struct {
	// Rate and Burst are a token bucket over calls started, per second; Rate 0 is no bucket.
	Rate  float64
	Burst int
	// Concurrency caps the calls running at once; 0 is no cap.
	Concurrency int
}

// nodeLimits holds a *nodeLimiter per Caller that LimitNode was given.
var nodeLimits sync.Map

type nodeLimiter struct {
	calls *rate.Limiter       // nil: no rate
	slots *semaphore.Weighted // nil: no cap
}

// LimitNode keeps every lens call made through c to limit, from the next call on: Call, the pages
// of a scan, a batch and the reads it makes alone. A call waits for its turn instead of being
// refused by the node. c is the Caller the trackers read through, such as the *ethclient.Client of
// their ethrpc client; a Caller that cannot be a map key cannot be limited. A zero limit removes
// c's.
func LimitNode(c Caller, limit NodeLimit) {
	if !keyable(c) {
		return
	}
	if limit.Rate <= 0 && limit.Concurrency <= 0 {
		nodeLimits.Delete(c)
		return
	}
	var l nodeLimiter
	if limit.Rate > 0 {
		l.calls = rate.NewLimiter(rate.Limit(limit.Rate), max(limit.Burst, 1))
	}
	if limit.Concurrency > 0 {
		l.slots = semaphore.NewWeighted(int64(limit.Concurrency))
	}
	nodeLimits.Store(c, &l)
}

// keyable reports whether c can key the maps kept per Caller or Chain: a nil or a func cannot.
func keyable(c any) bool {
	return c != nil && reflect.TypeOf(c).Comparable()
}

// nodeLimit is the limit LimitNode gave c, nil for none.
func nodeLimit(c Caller) *nodeLimiter {
	if !keyable(c) {
		return nil
	}
	v, _ := nodeLimits.Load(c)
	l, _ := v.(*nodeLimiter)
	return l
}

// wait waits until l lets a call start, and returns what ends the call. A nil limit lets every
// call start at once.
func (l *nodeLimiter) wait(ctx context.Context) (func(), error) {
	if l == nil {
		return func() {}, nil
	}
	release := func() {}
	if l.slots != nil {
		if err := l.slots.Acquire(ctx, 1); err != nil {
			return nil, err
		}
		release = func() { l.slots.Release(1) }
	}
	if l.calls != nil {
		if err := l.calls.Wait(ctx); err != nil {
			release()
			return nil, err
		}
	}
	return release, nil
}
