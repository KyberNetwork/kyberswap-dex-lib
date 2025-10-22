package angstrom

import (
	"math/big"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/samber/lo"
)

type Hook struct {
	uniswapv4.Hook
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Angstrom},
	}

	return hook
}, HookAddresses...)

func (h *Hook) BeforeSwap(swapHookParams *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecific:   bignumber.ZeroBI,
		DeltaUnSpecific: bignumber.ZeroBI,
		SwapFee:         uniswapv4.FeeAmount(UnlockedFee),
	}, nil
}

func (h *Hook) AfterSwap(swapHookParams *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	exactIn := swapHookParams.ExactIn
	targetAmount := swapHookParams.AmountOut

	var tmp big.Int

	fee := lo.Ternary(
		exactIn,

		new(big.Int).Div(
			tmp.Mul(targetAmount, ProtocolUnlockedFee),
			ONE_E6,
		),

		new(big.Int).Sub(
			tmp.Div(
				tmp.Mul(targetAmount, ONE_E6),
				tmp.Sub(ONE_E6, ProtocolUnlockedFee),
			),
			targetAmount,
		),
	)

	return &uniswapv4.AfterSwapResult{
		HookFee: fee,
	}, nil
}
