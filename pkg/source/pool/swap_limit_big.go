package pool

import (
	"math/big"

	"github.com/holiman/uint256"
)

// swapLimitBigAdapter exposes a uint256 SwapLimit through the legacy big.Int SwapLimitBig interface.
// Legacy simulators call it at their boundary, and it converts on each call.
type swapLimitBigAdapter struct {
	limit SwapLimit
}

// AsSwapLimitBig returns a legacy SwapLimitBig view of limit. A nil limit returns nil.
func AsSwapLimitBig(limit SwapLimit) SwapLimitBig {
	if limit == nil {
		return nil
	}
	return swapLimitBigAdapter{limit: limit}
}

func (a swapLimitBigAdapter) Clone() SwapLimitBig {
	return AsSwapLimitBig(a.limit.Clone())
}

func (a swapLimitBigAdapter) GetExchange() string {
	return a.limit.GetExchange()
}

func (a swapLimitBigAdapter) GetLimit(key string) *big.Int {
	return toBigOrNil(a.limit.GetLimit(key))
}

func (a swapLimitBigAdapter) GetSwapped() map[string]*big.Int {
	swapped := a.limit.GetSwapped()
	if swapped == nil {
		return nil
	}
	out := make(map[string]*big.Int, len(swapped))
	for k, v := range swapped {
		out[k] = toBigOrNil(v)
	}
	return out
}

func (a swapLimitBigAdapter) GetAllowedSenders() string {
	return a.limit.GetAllowedSenders()
}

func (a swapLimitBigAdapter) UpdateLimit(decreaseKey, increasedKey string,
	decreasedDelta, increasedDelta *big.Int) (*big.Int, *big.Int, error) {
	dec, ok1 := fromBigOrNil(decreasedDelta)
	inc, ok2 := fromBigOrNil(increasedDelta)
	if !ok1 || !ok2 {
		return nil, nil, ErrNotEnoughInventory
	}
	after, before, err := a.limit.UpdateLimit(decreaseKey, increasedKey, dec, inc)
	if err != nil {
		return nil, nil, err
	}
	return toBigOrNil(after), toBigOrNil(before), nil
}

// fromBigOrNil converts a non-negative big.Int that fits 256 bits. A nil delta stays nil, because an inner limit
// that ignores its deltas (SingleSwapLimit) must still see the update. It reports false for negative or oversized.
func fromBigOrNil(b *big.Int) (*uint256.Int, bool) {
	if b == nil {
		return nil, true
	}
	if b.Sign() < 0 || b.BitLen() > 256 {
		return nil, false
	}
	return uint256.MustFromBig(b), true
}

func toBigOrNil(u *uint256.Int) *big.Int {
	if u == nil {
		return nil
	}
	return u.ToBig()
}
