package bebop

import (
	"math/big"
	"sync/atomic"
)

// Limit: limit 1 bebop swap per route
type Limit struct {
	hasSwap *atomic.Bool
}

func NewLimit(_ map[string]*big.Int) *Limit {
	b := new(atomic.Bool)
	return &Limit{
		hasSwap: b,
	}
}

func (l *Limit) GetLimit(key string) *big.Int {
	if l.hasSwap.Load() {
		return big.NewInt(0)
	}

	return nil
}

func (l *Limit) UpdateLimit(
	_, _ string, _, _ *big.Int,
) (*big.Int, *big.Int, error) {
	l.hasSwap.Store(true)

	return nil, nil, nil
}
