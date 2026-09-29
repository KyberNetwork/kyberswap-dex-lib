// Package cashcat integrates CashCatHookV2, the fee engine for CashCat pools.
//
// The hook takes one flat fee on the quote side of every swap -- skimmed
// from the quote input on buys and the quote output on sells -- fixed for
// the life of the pool (no tiers, no decay, no exemptions). By hook
// construction the quote asset is always currency0, so in dex-lib terms the
// fee is input-side when zeroForOne and output-side otherwise.
//
// Because the fee is read directly from the hook (currentFeeRate) instead of
// fitted from quoter probes, pools on this hook must NOT fall through to the
// auto-detection fallback: explicit registration in HookFactories takes
// precedence over it.
package cashcat

import (
	"context"
	"errors"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	// FeeDenom is the hook's fee scale: 1e6 = 100%.
	FeeDenom = big.NewInt(1e6)

	ErrNotCalibrated = errors.New("cashcat: fee rate not tracked yet")
	ErrFeeTooHigh    = errors.New("cashcat: fee rate exceeds 100%")
)

type Hook struct {
	uniswapv4.Hook `json:"-"`
	// FeeRate is the pool's flat fee in pips of the quote side (1e6 = 100%),
	// read once via Track (immutable for the pool's life) and persisted in
	// HookExtra.
	FeeRate *big.Int `json:"f,omitempty"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Cashcat},
	}
	if len(param.HookExtra) > 0 {
		var persisted Hook
		if err := param.HookExtra.Unmarshal(&persisted); err == nil {
			hook.FeeRate = persisted.FeeRate
		}
	}
	return hook
}, HookAddresses...)

func (h *Hook) feeRate() (*big.Int, error) {
	if h.FeeRate == nil {
		return nil, ErrNotCalibrated
	}
	if h.FeeRate.Cmp(FeeDenom) > 0 {
		return nil, ErrFeeTooHigh
	}
	return h.FeeRate, nil
}

// Track reads the pool's immutable fee rate. A present HookExtra in our own
// schema is returned as-is (the rate never changes, so re-reading is pure
// cost). Anything else -- empty, or a foreign schema such as the
// auto-detection model persisted before this hook was explicitly registered
// -- falls through to a fresh on-chain read, which is also what migrates
// existing pools off the auto model on the first track after deploy.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if len(param.HookExtra) > 0 {
		var persisted Hook
		if err := param.HookExtra.Unmarshal(&persisted); err == nil && persisted.FeeRate != nil {
			h.FeeRate = persisted.FeeRate
			return json.Marshal(h)
		}
	}
	if param.RpcClient == nil {
		return nil, ErrNotCalibrated
	}

	var feeRate *big.Int
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    cashCatHookV2ABI,
		Target: param.HookAddress.Hex(),
		Method: "currentFeeRate",
		Params: []any{common.HexToHash(param.Pool.Address), common.Address{}},
	}, []any{&feeRate}).TryBlockAndAggregate(); err != nil {
		return nil, err
	}
	if feeRate == nil {
		return nil, ErrNotCalibrated
	}
	if feeRate.Cmp(FeeDenom) > 0 {
		return nil, ErrFeeTooHigh
	}
	h.FeeRate = feeRate

	return json.Marshal(h)
}

// quoteSpecified reports whether the swap's specified side is the quote
// asset (currency0). Mirrors the hook's `ethSpecified`, where exact-input
// swaps carry a negative amountSpecified and exact-output a positive one:
// exact-input selling currency0, or exact-output buying currency0. In
// dex-lib terms exact-input is CalcOut and exact-output is !CalcOut.
func quoteSpecified(calcOut, zeroForOne bool) bool {
	return zeroForOne == calcOut
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	feeRate, err := h.feeRate()
	if err != nil {
		return nil, err
	}
	deltaSpecified := bignumber.ZeroBI
	if quoteSpecified(params.CalcOut, params.ZeroForOne) {
		if params.CalcOut {
			// Exact-input sell of the quote asset: skim the fee off the
			// full input, _feeFor(amount, rate, exactOutput=false).
			deltaSpecified = bignumber.MulDivDown(new(big.Int),
				params.AmountSpecified, feeRate, FeeDenom)
		} else {
			// Exact-output buy of the quote asset: the named amount is
			// what the trader keeps, so gross up to the leg it implies,
			// _feeFor(amount, rate, exactOutput=true).
			denom := new(big.Int).Sub(FeeDenom, feeRate)
			if denom.Sign() <= 0 {
				return nil, ErrFeeTooHigh
			}
			deltaSpecified = bignumber.MulDivDown(new(big.Int),
				params.AmountSpecified, feeRate, denom)
		}
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   deltaSpecified,
		DeltaUnspecified: bignumber.ZeroBI,
	}, nil
}

func (h *Hook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	feeRate, err := h.feeRate()
	if err != nil {
		return nil, err
	}
	hookFee := bignumber.ZeroBI
	if !quoteSpecified(params.CalcOut, params.ZeroForOne) {
		if params.CalcOut {
			// Exact-input sell of the base asset: fee on the actual quote
			// moved, _feeFor(amountOut, rate, exactOutput=false).
			hookFee = bignumber.MulDivDown(new(big.Int),
				params.AmountOut, feeRate, FeeDenom)
		} else {
			// Exact-output sell of the base asset: fee on the actual quote
			// moved, grossed up, _feeFor(amountIn, rate, exactOutput=true).
			// AmountIn here is the pre-fee requirement, which is exactly
			// the base the gross-up inverts.
			denom := new(big.Int).Sub(FeeDenom, feeRate)
			if denom.Sign() <= 0 {
				return nil, ErrFeeTooHigh
			}
			hookFee = bignumber.MulDivDown(new(big.Int),
				params.AmountIn, feeRate, denom)
		}
	}
	return &uniswapv4.AfterSwapResult{
		HookFee: hookFee,
	}, nil
}
