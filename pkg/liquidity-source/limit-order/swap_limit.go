package limitorder

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// bigSwapLimit exposes a legacy big.Int SwapLimitBig as a uint256 SwapLimit for the shared fill walk.
// The legacy CalcAmountOut and CalcAmountIn convert their limit with it once per call.
type bigSwapLimit struct {
	limit pool.SwapLimitBig
}

func swapLimitFromBig(limit pool.SwapLimitBig) pool.SwapLimit {
	if limit == nil {
		return nil
	}
	return bigSwapLimit{limit: limit}
}

func (b bigSwapLimit) Clone() pool.SwapLimit {
	return swapLimitFromBig(b.limit.Clone())
}

func (b bigSwapLimit) GetExchange() string {
	return b.limit.GetExchange()
}

func (b bigSwapLimit) GetLimit(key string) *uint256.Int {
	return u256FromBig(b.limit.GetLimit(key))
}

func (b bigSwapLimit) GetSwapped() map[string]*uint256.Int {
	swapped := b.limit.GetSwapped()
	if swapped == nil {
		return nil
	}
	out := make(map[string]*uint256.Int, len(swapped))
	for k, v := range swapped {
		out[k] = u256FromBig(v)
	}
	return out
}

func (b bigSwapLimit) GetAllowedSenders() string {
	return b.limit.GetAllowedSenders()
}

func (b bigSwapLimit) UpdateLimit(decreaseKey, increasedKey string,
	decreasedDelta, increasedDelta *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	after, before, err := b.limit.UpdateLimit(decreaseKey, increasedKey, toBigOrNil(decreasedDelta),
		toBigOrNil(increasedDelta))
	if err != nil {
		return nil, nil, err
	}
	return u256FromBig(after), u256FromBig(before), nil
}

// u256FromBig converts a limit balance like swaplimit.ToU256Balances: nil or negative is zero, and a value above
// 2^256 saturates to MaxUint256. It returns nil for nil.
func u256FromBig(b *big.Int) *uint256.Int {
	switch {
	case b == nil:
		return nil
	case b.Sign() < 0:
		return new(uint256.Int)
	case b.BitLen() > 256:
		return new(uint256.Int).SetAllOne()
	default:
		return uint256.MustFromBig(b)
	}
}

func toBigOrNil(u *uint256.Int) *big.Int {
	if u == nil {
		return nil
	}
	return u.ToBig()
}
