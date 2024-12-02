package metrics

import (
	"context"
	"sync"
	"sync/atomic"
)

type CalcAmountOutCounterContextKeyT string

type CalcAmountOutCounter struct {
	inner sync.Map
}

const (
	CalcAmountOutCounterContextKey = CalcAmountOutCounterContextKeyT("CalcAmountOutCounterContextKey")
)

func NewCalcAmountOutCounter() *CalcAmountOutCounter {
	return &CalcAmountOutCounter{inner: sync.Map{}}
}

func (c *CalcAmountOutCounter) Inc(dexType string, inc uint64) {
	var (
		counter any
		ok      bool
	)
	if counter, ok = c.inner.Load(dexType); !ok {
		counter, _ = c.inner.LoadOrStore(dexType, new(atomic.Uint64))
	}
	counter.(*atomic.Uint64).Add(inc)
}

func (c *CalcAmountOutCounter) CommitMetrics(ctx context.Context) {
	c.inner.Range(func(key, value any) bool {
		dexType := key.(string)
		counter := value.(*atomic.Uint64)
		RecordCalcAmountOutCountPerRequest(ctx, int64(counter.Load()), dexType)
		return true
	})
}
