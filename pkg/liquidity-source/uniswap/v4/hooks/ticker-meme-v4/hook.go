// Package tickermemev4 registers Ticker's proxy with the existing Pons fee handler.
package tickermemev4

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const basisPoints = 10_000

type Hook struct {
	uniswapv4.BaseHook `json:"-"`
	ponsv2.Extra
}

var _ = uniswapv4.RegisterHooksFactory(NewHook, HookAddress)

func NewHook(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{BaseHook: uniswapv4.BaseHook{Exchange: DexType}}
	_ = param.HookExtra.Unmarshal(&h.Extra)
	return h
}

func (h *Hook) CanBeforeSwap(common.Address) bool { return true }

// BeforeSwap only serves CalcAmountIn, which reverses an exact-in swap. The hook takes
// its fee from the realized output, so the pool must output the grossed-up amount.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	delta := bignumber.ZeroBI
	if !params.CalcOut && h.Registered && h.FeeBps > 0 {
		gross := uniswapv4.GrossBeforeFee(params.AmountSpecified, big.NewInt(h.FeeBps), big.NewInt(basisPoints))
		delta = gross.Sub(gross, params.AmountSpecified)
	}
	return &uniswapv4.BeforeSwapResult{DeltaSpecified: delta, DeltaUnspecified: bignumber.ZeroBI}, nil
}

// AfterSwap charges nothing for CalcAmountIn: BeforeSwap already priced the output fee.
func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if !params.CalcOut {
		return &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI}, nil
	}
	return h.Extra.AfterSwap(params)
}
