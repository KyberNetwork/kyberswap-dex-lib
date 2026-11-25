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

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

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
