package swaplimit

import (
	"math/big"
	"sync/atomic"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// SingleSwapLimit implements Swap Limit for dexes that can only swap once
type SingleSwapLimit struct {
	exchange string
	swapped  *atomic.Bool
}

// NewSingleSwapLimit creates a new SingleSwapLimit
func NewSingleSwapLimit(exchange string) *SingleSwapLimit {
	return &SingleSwapLimit{
		exchange: exchange,
		swapped:  new(atomic.Bool),
	}
}

// Clone clones SingleSwapLimit.
func (l *SingleSwapLimit) Clone() pool.SwapLimit {
	cloned := NewSingleSwapLimit(l.exchange)
	if l.swapped.Load() {
		cloned.swapped.Store(true)
	}
	return cloned
}

// GetExchange returns the exchange name.
func (l *SingleSwapLimit) GetExchange() string {
	return l.exchange
}

// GetLimit returns ZeroBI if a swap has already been done, nil otherwise
func (l *SingleSwapLimit) GetLimit(_ string) *big.Int {
	if l.swapped.Load() {
		return bignumber.ZeroBI
	}
	return nil
}

func (l *SingleSwapLimit) GetSwapped() map[string]*big.Int {
	return nil
}

func (i *SingleSwapLimit) GetAllowedSenders() string {
	return ""
}

// UpdateLimit updates the atomic bool to mark swap done
func (l *SingleSwapLimit) UpdateLimit(_, _ string, _, _ *big.Int) (*big.Int, *big.Int, error) {
	l.swapped.Store(true)
	return nil, nil, nil
}
