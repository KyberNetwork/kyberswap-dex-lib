package ozlimitorder

import (
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// fillGas is the extra gas of the hook's afterSwap when a swap crosses a limit order: it removes that order's
// liquidity and takes its proceeds. Measured on a Monad fork: about 100k per filled order (a swap filling one order
// cost 258k against 99k for one crossing none, and three orders 455k).
const fillGas = 100_000

// Hook is OpenZeppelin's LimitOrderHook. It has afterInitialize and afterSwap permissions only, without
// afterSwapReturnDelta, so it never changes a swap's amounts: the swap prices exactly like a hookless v4 pool.
// After the swap it removes the liquidity of every limit order the swap fully crossed; the pool tracker picks up
// that change from the PoolManager's ModifyLiquidity events.
type Hook struct {
	*uniswapv4.BaseHook
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	return &Hook{
		BaseHook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4},
	}
}, HookAddresses...)

func (h *Hook) AfterSwap(_ *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return &uniswapv4.AfterSwapResult{
		HookFee: bignumber.ZeroBI,
		Gas:     fillGas,
	}, nil
}

func (h *Hook) CloneState() uniswapv4.Hook {
	return h
}
