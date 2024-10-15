package bebop

import "math/big"

// Limit: limit 1 bebop swap per route
type Limit struct {
	hasSwap bool
}

func NewLimit(_ map[string]*big.Int) *Limit {
	return &Limit{
		hasSwap: false,
	}
}

func (l *Limit) GetLimit(key string) *big.Int {
	if l.hasSwap {
		return big.NewInt(0)
	}

	return nil
}

func (l *Limit) UpdateLimit(
	_, _ string, _, _ *big.Int,
) (*big.Int, *big.Int, error) {
	l.hasSwap = true

	return nil, nil, nil
}
