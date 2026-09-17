// Package hook registers Ring Protocol's FewToken wrapper-pool hooks
// (FewTokenHook, FewUSDTHook, FewETHHook) with uniswapv4.HookFactories.
//
// This must live outside the few/v1, few/v2 packages: uniswapv4's own
// pool_simulator.go imports few/v1 and few/v2 directly (for the
// ITokenWrapper wrap-substitution feature), so those packages — or few
// itself — cannot import uniswapv4 back without an import cycle. This
// package imports uniswapv4 and few/v1/few/v2 from the outside, the same
// direction every other hook package (idle, bunni-v2, ...) already uses.
package hook

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	few_v1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/few/v1"
	few_v2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/few/v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Hook models Ring Protocol's FewToken wrapper-pool hooks. Read from source:
// wrap/unwrap is an unconditional 1:1 identity swap, zero fee, both exact-in
// and exact-out (_getWrapInputRequired/_getUnwrapInputRequired always return
// their input unchanged) — see few/v1, few/v2 token_info.go doc comments for
// the verification trail.
//
// Registering these hook addresses explicitly matters beyond correctness: an
// unregistered swap-permission hook falls through to auto/calibrate.go's
// generic probe-and-fit fallback (in kyberswap-dex-lib-private), which sizes
// its test amounts off the underlying v3 curve/reserves — a premise that
// does not hold here, since beforeSwapReturnDelta fully replaces the swap
// outcome and the "curve" is a single static full-range position never
// actually touched by a real swap. That probe reverts here and pool-service
// marks the pool permanently unquotable, even though the price is fully
// known statically and needs no calibration at all.
type Hook struct {
	*uniswapv4.BaseHook
}

var _ = uniswapv4.RegisterHooksFactory(func(*uniswapv4.HookParam) uniswapv4.Hook {
	return &Hook{
		BaseHook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4},
	}
}, hookAddresses()...)

func hookAddresses() []common.Address {
	addrs := append(few_v1.HookAddresses(), few_v2.HookAddresses()...)
	return lo.Map(addrs, func(a string, _ int) common.Address { return common.HexToAddress(a) })
}

// AllowEmptyTicks lets the underlying v3 simulator accept the zero-amount
// swap BeforeSwap leaves it (the hook consumes the entire specified amount
// and supplies the entire unspecified amount itself), matching the same
// pattern auto.Hook uses for ModelCustomCurve/SKIP_SWAP-style hooks.
func (h *Hook) AllowEmptyTicks() bool {
	return true
}

// BeforeSwap fully replaces the swap outcome with an exact 1:1 passthrough:
// the entire specified amount is consumed here (DeltaSpecified), and the
// entire unspecified (output) amount is supplied here too (DeltaUnspecified),
// leaving nothing for the underlying v3 curve to price.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	amt := new(big.Int).Set(params.AmountSpecified)
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   amt,
		DeltaUnspecified: new(big.Int).Neg(amt),
	}, nil
}

// AfterSwap is a no-op: BeforeSwap's beforeSwapReturnDelta already produced
// the full 1:1 outcome, so there is nothing left to charge on the output side.
func (h *Hook) AfterSwap(*uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return &uniswapv4.AfterSwapResult{
		HookFee: bignumber.ZeroBI,
	}, nil
}
