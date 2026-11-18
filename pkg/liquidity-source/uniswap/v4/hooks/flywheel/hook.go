// Package flywheel integrates the October 13 native V4 deployment. The legacy
// flywheel-fun settlement source remains responsible for older launches.
package flywheel

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var HookAddress = common.HexToAddress("0x992118de23b4ebe9f527dacbbb7d8d4bbbf8e044")

var (
	ErrUnsupportedChain = errors.New("flywheel: native V4 hook is only supported on Robinhood")
	ErrInvalidAmount    = errors.New("flywheel: amount must be positive and fit the core int128 delta")
)

// The immutable 400/10000 fee simplifies exactly to 1/25, including rounding.
var feeDivisor = uint256.NewInt(25)
var netDivisor = uint256.NewInt(24)

// Fork execution measured up to 89,306 gas in afterSwap; allow headroom for
// cold fee-ledger writes plus the one-time graduation check/write.
const gasAfterSwap int64 = 130_000

type Hook struct {
	uniswapv4.BaseHook
	WrongChain bool
}

var _ = uniswapv4.RegisterHooksFactory(func(p *uniswapv4.HookParam) uniswapv4.Hook {
	return &Hook{
		BaseHook:   uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Flywheel},
		WrongChain: p.Cfg != nil && p.Cfg.ChainID != valueobject.ChainIDRobinhood,
	}
}, HookAddress)

// The hook only permits its factory to initialize registered pools. All such
// pools use the same immutable fee, before and after graduation; no extra RPC
// reads or synthetic reserves are needed beyond the standard V4 tracker.
func (h *Hook) Track(context.Context, *uniswapv4.HookParam) (json.RawMessage, error) {
	if h.WrongChain {
		return nil, ErrUnsupportedChain
	}
	return nil, nil
}

func (h *Hook) TracksActivity() {}

func (h *Hook) CloneState() uniswapv4.Hook {
	clone := *h
	return &clone
}

// This is simulator-only. The deployed hook has no beforeSwap callback.
// Kyber's CalcAmountIn reverses exact-input execution, not on-chain exact-out.
func (h *Hook) CanBeforeSwap(common.Address) bool { return true }

func (h *Hook) BeforeSwap(p *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.WrongChain {
		return nil, ErrUnsupportedChain
	}
	if p.AmountSpecified == nil || p.AmountSpecified.Sign() <= 0 || p.AmountSpecified.BitLen() > 127 {
		return nil, ErrInvalidAmount
	}
	result := &uniswapv4.BeforeSwapResult{DeltaSpecified: bignumber.ZeroBI, DeltaUnspecified: bignumber.ZeroBI}
	if !p.CalcOut {
		var net, gross uint256.Int
		net.SetFromBig(p.AmountSpecified)
		// Smallest gross with gross-floor(gross/25)>=net:
		// floor((net-1)*25/24)+1. net is <=127 bits, so the product fits.
		gross.SubUint64(&net, 1).Mul(&gross, feeDivisor).Div(&gross, netDivisor).AddUint64(&gross, 1)
		if gross.BitLen() > 127 {
			return nil, ErrInvalidAmount
		}
		result.DeltaSpecified = big256.ToBig(gross.Sub(&gross, &net))
	}
	return result, nil
}

func (h *Hook) AfterSwap(p *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if h.WrongChain {
		return nil, ErrUnsupportedChain
	}
	result := &uniswapv4.AfterSwapResult{HookFee: bignumber.ZeroBI, Gas: gasAfterSwap}
	if !p.CalcOut {
		// Output fee already inverted by BeforeSwap.
		return result, nil
	}
	if p.AmountOut == nil || p.AmountOut.Sign() <= 0 || p.AmountOut.BitLen() > 127 {
		return nil, ErrInvalidAmount
	}
	var output uint256.Int
	output.SetFromBig(p.AmountOut)
	result.HookFee = big256.ToBig(output.Div(&output, feeDivisor))
	return result, nil
}
