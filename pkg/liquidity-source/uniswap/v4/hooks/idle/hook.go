package idle

import (
	"math/big"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	feePct = bignumber.Three
)

// Hook of idle takes 3% fee of ETH token
type Hook struct {
	*uniswapv4.BaseHook
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	return &Hook{
		BaseHook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4},
	}
}, HookAddresses...)

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	deltaSpecific := bignumber.ZeroBI
	if params.ZeroForOne {
		deltaSpecific = bignumber.MulDivDown(new(big.Int), params.AmountSpecified, feePct, bignumber.B100)
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecific:   deltaSpecific,
		DeltaUnSpecific: bignumber.ZeroBI,
	}, nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	hookFeeAmt := bignumber.ZeroBI
	if !params.ZeroForOne {
		hookFeeAmt = bignumber.MulDivDown(new(big.Int), params.AmountOut, feePct, bignumber.B100)
	}
	return &uniswapv4.AfterSwapResult{
		HookFee: hookFeeAmt,
	}, nil
}
