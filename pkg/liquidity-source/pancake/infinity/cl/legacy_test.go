package cl

import (
	"fmt"
	"math/big"

	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Legacy big.Int CalcAmountOut and CalcAmountIn, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (swapResult *pool.CalcAmountOutResult,
	err error) {
	originalTokenIn, originalTokenOut := param.TokenAmountIn.Token, param.TokenOut
	var wrapAdditionalGas int64
	var beforeSwapResult *BeforeSwapResult
	var afterSwapResult *AfterSwapResult

	// A native-flagged side trades at native's own decimals on-chain, not the wrapped-native
	// address's used in p.Tokens; only Arc's differ (18 vs 6). Rescale around the v3 math below.
	if idx := p.GetTokenIndex(originalTokenIn); idx >= 0 && p.staticExtra.IsNative[idx] {
		param.TokenAmountIn.Amount = valueobject.UnwrapNativeAmount(p.chainID, param.TokenAmountIn.Amount)
	}

	defer func() { // modify result before return
		if swapResult == nil {
			return
		}
		v4SwapInfo := SwapInfo{
			PoolSwapInfo: swapResult.SwapInfo.(PoolSwapInfo),
		}

		if swapResult.TokenAmountOut != nil {
			swapResult.TokenAmountOut.Token = originalTokenOut

			if beforeSwapResult != nil {
				swapResult.TokenAmountOut.Amount.Sub(swapResult.TokenAmountOut.Amount,
					big256.ToBig(beforeSwapResult.DeltaUnspecified))
				swapResult.Gas += beforeSwapResult.Gas
				v4SwapInfo.HookSwapInfo = beforeSwapResult.SwapInfo
			}

			if afterSwapResult != nil {
				swapResult.TokenAmountOut.Amount.Sub(swapResult.TokenAmountOut.Amount,
					big256.ToBig(afterSwapResult.HookFee))
				swapResult.Gas += afterSwapResult.Gas
			}

			if idx := p.GetTokenIndex(originalTokenOut); idx >= 0 && p.staticExtra.IsNative[idx] {
				swapResult.TokenAmountOut.Amount = valueobject.WrapNativeAmount(p.chainID, swapResult.TokenAmountOut.Amount)
			}
		}
		swapResult.SwapInfo = v4SwapInfo

		if swapResult.RemainingTokenAmountIn != nil {
			swapResult.RemainingTokenAmountIn.Token = originalTokenIn
		}

		swapResult.Gas += wrapAdditionalGas

		if swapResult.TokenAmountOut.Amount.Sign() < 0 {
			swapResult = nil
			err = ErrInvalidAmountOut
		}
	}()

	// If no hooks, just do swap
	poolSim := p.PoolSimulator
	if p.hook == nil {
		return v3CalcAmountOut(p.PoolSimulator, param)
	}

	tokenIn := param.TokenAmountIn.Token
	zeroForOne := p.GetTokenIndex(tokenIn) == 0
	amountIn := new(big.Int).Set(param.TokenAmountIn.Amount)

	if beforeSwapResult, err = p.hook.BeforeSwap(&BeforeSwapParams{
		CalcOut:         true,
		ZeroForOne:      zeroForOne,
		AmountSpecified: big256.FromBig(amountIn),
	}); err != nil {
		return nil, fmt.Errorf("[BeforeSwap] %w", err)
	} else if err = ValidateBeforeSwapResult(beforeSwapResult); err != nil {
		return nil, fmt.Errorf("[BeforeSwap] validation failed: %w", err)
	}

	if amountIn.Sub(amountIn, big256.ToBig(beforeSwapResult.DeltaSpecified)).Sign() < 0 {
		return nil, ErrInvalidAmountIn
	}

	if beforeSwapResult.SwapFee >= FeeMax {
		return nil, ErrInvalidFee
	} else if beforeSwapResult.SwapFee > 0 && beforeSwapResult.SwapFee != p.V3Pool.Fee {
		cloned := *poolSim
		clonedV3Pool := *poolSim.V3Pool
		cloned.V3Pool = &clonedV3Pool
		cloned.V3Pool.Fee = beforeSwapResult.SwapFee
		poolSim = &cloned
	}

	if amountIn.Sign() == 0 {
		// Hook consumed the full input — math is fully overridden by the
		// hook's BeforeSwap delta. Skip the v3 tick simulator (which would
		// error on amountIn=0 for pools with no real CL liquidity, e.g.
		// curve-style stable hooks). The deferred handler subtracts
		// DeltaUnspecified (negative for full overrides) from the zero
		// output to produce the hook-provided amountOut.
		//
		// PoolSwapInfo carries nil tick fields so UpdateBalance can detect
		// "v3 was skipped" and avoid touching v3 tick state.
		swapResult = &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: new(big.Int)},
			Fee:            &pool.TokenAmount{Token: param.TokenOut, Amount: new(big.Int)},
			Gas:            0,
			SwapInfo:       PoolSwapInfo{}, // all-nil → UpdateBalance skips v3
		}
	} else if swapResult, err = v3CalcAmountOut(poolSim, pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		TokenOut: param.TokenOut,
	}); err != nil {
		return nil, err
	}

	afterSwapResult, err = p.hook.AfterSwap(&AfterSwapParams{
		BeforeSwapParams: &BeforeSwapParams{
			CalcOut:         true,
			ZeroForOne:      zeroForOne,
			AmountSpecified: big256.FromBig(amountIn),
		},
		AmountIn:  big256.FromBig(amountIn),
		AmountOut: big256.FromBig(swapResult.TokenAmountOut.Amount),
	})
	if err != nil {
		return nil, fmt.Errorf("[AfterSwap] %w", err)
	} else if err = ValidateAfterSwapResult(afterSwapResult); err != nil {
		return nil, fmt.Errorf("[AfterSwap] validation failed: %w", err)
	}

	return
}

// v3CalcAmountOut quotes with the v3 pool's native CalcAmtOut and returns big.Int amounts, since the v3 legacy
// CalcAmountOut is itself a test-only method. Amounts above 256 bits fail as v3 ErrOverflow, like its legacy body.
func v3CalcAmountOut(sim *uniswapv3.PoolSimulator, param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var amountIn uint256.Int
	if overflow := amountIn.SetFromBig(param.TokenAmountIn.Amount); overflow {
		return nil, uniswapv3.ErrOverflow
	}
	res, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: param.TokenAmountIn.Token, Amount: amountIn},
		TokenOut:      param.TokenOut,
	})
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: res.TokenAmountOut.Token, Amount: res.TokenAmountOut.Amount.ToBig()},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: res.RemainingTokenAmountIn.Token, Amount: res.RemainingTokenAmountIn.Amount.ToBig()},
		Gas:                    res.Gas,
		SwapInfo:               res.SwapInfo,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (swapResult *pool.CalcAmountInResult, err error) {
	originalTokenOut, originalTokenIn := param.TokenAmountOut.Token, param.TokenIn
	var wrapAdditionalGas int64
	var beforeSwapResult *BeforeSwapResult
	var afterSwapResult *AfterSwapResult

	// A native-flagged side trades at native's own decimals on-chain, not the wrapped-native
	// address's used in p.Tokens; only Arc's differ (18 vs 6). Rescale around the v3 math below.
	if idx := p.GetTokenIndex(originalTokenOut); idx >= 0 && p.staticExtra.IsNative[idx] {
		param.TokenAmountOut.Amount = valueobject.UnwrapNativeAmount(p.chainID, param.TokenAmountOut.Amount)
	}

	defer func() { // modify result before return
		if swapResult == nil {
			return
		}
		v4SwapInfo := SwapInfo{
			PoolSwapInfo: swapResult.SwapInfo.(PoolSwapInfo),
		}

		if swapResult.TokenAmountIn != nil {
			swapResult.TokenAmountIn.Token = originalTokenIn

			if beforeSwapResult != nil {
				swapResult.TokenAmountIn.Amount.Add(swapResult.TokenAmountIn.Amount,
					big256.ToBig(beforeSwapResult.DeltaUnspecified))
				swapResult.Gas += beforeSwapResult.Gas
				v4SwapInfo.HookSwapInfo = beforeSwapResult.SwapInfo
			}

			if afterSwapResult != nil {
				swapResult.TokenAmountIn.Amount.Add(swapResult.TokenAmountIn.Amount,
					big256.ToBig(afterSwapResult.HookFee))
				swapResult.Gas += afterSwapResult.Gas
			}

			if idx := p.GetTokenIndex(originalTokenIn); idx >= 0 && p.staticExtra.IsNative[idx] {
				swapResult.TokenAmountIn.Amount = valueobject.WrapNativeAmount(p.chainID, swapResult.TokenAmountIn.Amount)
			}
		}
		swapResult.SwapInfo = v4SwapInfo

		if swapResult.RemainingTokenAmountOut != nil {
			swapResult.RemainingTokenAmountOut.Token = originalTokenOut
		}

		swapResult.Gas += wrapAdditionalGas

		if swapResult.TokenAmountIn.Amount.Sign() < 0 {
			swapResult = nil
			err = ErrInvalidAmountIn
		}
	}()

	poolSim := p.PoolSimulator
	if p.hook == nil {
		swapResult, err = v3CalcAmountIn(poolSim, param)
		return
	}

	tokenOut := param.TokenAmountOut.Token
	zeroForOne := p.GetTokenIndex(tokenOut) == 1
	amountOut := new(big.Int).Set(param.TokenAmountOut.Amount)

	if beforeSwapResult, err = p.hook.BeforeSwap(&BeforeSwapParams{
		CalcOut:         false,
		ZeroForOne:      zeroForOne,
		AmountSpecified: big256.FromBig(amountOut),
	}); err != nil {
		return nil, fmt.Errorf("[BeforeSwap] %w", err)
	} else if err = ValidateBeforeSwapResult(beforeSwapResult); err != nil {
		return nil, fmt.Errorf("[BeforeSwap] validation failed: %w", err)
	}

	if amountOut.Add(amountOut, big256.ToBig(beforeSwapResult.DeltaSpecified)).Sign() < 0 {
		return nil, ErrInvalidAmountOut
	}

	if beforeSwapResult.SwapFee >= FeeMax {
		return nil, ErrInvalidFee
	} else if beforeSwapResult.SwapFee > 0 && beforeSwapResult.SwapFee != p.V3Pool.Fee {
		cloned := *poolSim
		clonedV3Pool := *poolSim.V3Pool
		cloned.V3Pool = &clonedV3Pool
		cloned.V3Pool.Fee = beforeSwapResult.SwapFee
		poolSim = &cloned
	}

	if amountOut.Sign() == 0 {
		// Hook fully provides the output (full math override). See the matching
		// short-circuit in CalcAmountOut above for the rationale. The deferred
		// handler adds DeltaUnspecified (positive for full overrides) to the
		// zero input to produce the hook-provided amountIn.
		swapResult = &pool.CalcAmountInResult{
			TokenAmountIn: &pool.TokenAmount{Token: param.TokenIn, Amount: new(big.Int)},
			Fee:           &pool.TokenAmount{Token: param.TokenIn, Amount: new(big.Int)},
			Gas:           0,
			SwapInfo:      PoolSwapInfo{},
		}
	} else if swapResult, err = v3CalcAmountIn(poolSim, pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		TokenIn: param.TokenIn,
	}); err != nil {
		return nil, err
	}

	if afterSwapResult, err = p.hook.AfterSwap(&AfterSwapParams{
		BeforeSwapParams: &BeforeSwapParams{
			CalcOut:         false,
			ZeroForOne:      zeroForOne,
			AmountSpecified: big256.FromBig(amountOut),
		},
		AmountIn:  big256.FromBig(swapResult.TokenAmountIn.Amount),
		AmountOut: big256.FromBig(amountOut),
	}); err != nil {
		return nil, fmt.Errorf("[AfterSwap] %w", err)
	} else if err = ValidateAfterSwapResult(afterSwapResult); err != nil {
		return nil, fmt.Errorf("[AfterSwap] validation failed: %w", err)
	}

	return
}

// v3CalcAmountIn quotes with the v3 pool's native CalcAmtIn and returns big.Int amounts, since the v3 legacy
// CalcAmountIn is itself a test-only method. An amount above 256 bits is clamped to 2^256-1, which v3 rejects
// as above the reserves, as the big.Int body did.
func v3CalcAmountIn(sim *uniswapv3.PoolSimulator, param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	var amountOut uint256.Int
	if overflow := amountOut.SetFromBig(param.TokenAmountOut.Amount); overflow {
		amountOut.SetAllOne()
	}
	res, err := sim.CalcAmtIn(pool.CalcAmtInParams{
		TokenAmountOut: pool.TokenAmt{Token: param.TokenAmountOut.Token, Amount: amountOut},
		TokenIn:        param.TokenIn,
	})
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountInResult{
		TokenAmountIn:           &pool.TokenAmount{Token: res.TokenAmountIn.Token, Amount: res.TokenAmountIn.Amount.ToBig()},
		RemainingTokenAmountOut: &pool.TokenAmount{Token: res.RemainingTokenAmountOut.Token, Amount: res.RemainingTokenAmountOut.Amount.ToBig()},
		Fee:                     &pool.TokenAmount{Token: res.TokenAmountIn.Token, Amount: new(big.Int)},
		Gas:                     res.Gas,
		SwapInfo:                res.SwapInfo,
	}, nil
}
