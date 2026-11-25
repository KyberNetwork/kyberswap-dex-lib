package uniswapv4

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

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (swapResult *pool.CalcAmountOutResult, err error) {
	originalTokenIn, originalTokenOut := param.TokenAmountIn.Token, param.TokenOut
	var wrapAdditionalGas int64
	var beforeSwapResult *BeforeSwapResult
	var afterSwapResult *AfterSwapResult
	unwrapIdx := -1 // pool index of the Few token this swap unwraps

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
		} else if v4SwapInfo.FewReserve, err = p.unwrapFew(unwrapIdx,
			swapResult.TokenAmountOut.Amount); err != nil {
			swapResult = nil
		} else if v4SwapInfo.FewReserve != nil {
			v4SwapInfo.FewIdx = unwrapIdx
			swapResult.SwapInfo = v4SwapInfo
		}
	}()

	// Wrap/unwrap tokens if needed and calculate wrap gas
	if p.GetTokenIndex(param.TokenAmountIn.Token) == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canWrap := wrapper.CanWrap(p.chainID, param.TokenAmountIn.Token)
			if canWrap {
				param.TokenAmountIn.Token = metadata.GetWrapToken()
				wrapAdditionalGas += p.Gas.BaseGas
				break
			}
		}
	}
	if p.GetTokenIndex(param.TokenOut) == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canUnwrap := wrapper.CanWrap(p.chainID, param.TokenOut)
			if canUnwrap {
				param.TokenOut = metadata.GetWrapToken()
				unwrapIdx = p.GetTokenIndex(param.TokenOut)
				wrapAdditionalGas += p.Gas.BaseGas
				break
			}
		}
	} else if p.fewWrapperIdx >= 0 && p.GetTokenIndex(param.TokenAmountIn.Token) == p.fewWrapperIdx {
		unwrapIdx = p.fewWrapperIdx
	}

	tokenIn := param.TokenAmountIn.Token
	zeroForOne := p.GetTokenIndex(tokenIn) == 0

	// If no hooks, just do swap
	if p.hook == nil {
		swapFee, _ := p.swapFee(zeroForOne, nil)
		swapResult, err = v3CalcAmountOut(p.withSwapFee(swapFee), param)
		zeroNilFeeAmount(swapResult)
		return
	}

	amountIn := new(big.Int).Set(param.TokenAmountIn.Amount)

	if p.hook.CanBeforeSwap(p.staticExtra.HooksAddress) {
		if beforeSwapResult, err = p.hook.BeforeSwap(&BeforeSwapParams{
			CalcOut:         true,
			ZeroForOne:      zeroForOne,
			AmountSpecified: big256.FromBig(amountIn),
			SqrtPriceX96:    &p.V3Pool.SqrtRatioX96,
		}); err != nil {
			return nil, fmt.Errorf("[BeforeSwap] %w", err)
		} else if err = ValidateBeforeSwapResult(beforeSwapResult); err != nil {
			return nil, fmt.Errorf("[BeforeSwap] validation failed: %w", err)
		}

		if amountIn.Sub(amountIn, big256.ToBig(beforeSwapResult.DeltaSpecified)).Sign() < 0 {
			return nil, ErrInvalidAmountIn
		}
	}

	swapFee, err := p.swapFee(zeroForOne, beforeSwapResult)
	if err != nil {
		return nil, err
	}

	if swapResult, err = v3CalcAmountOut(p.withSwapFee(swapFee), pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		TokenOut: param.TokenOut,
	}); err != nil {
		return nil, err
	}
	zeroNilFeeAmount(swapResult)

	if p.hook.CanAfterSwap(p.staticExtra.HooksAddress) {
		afterSwapResult, err = p.hook.AfterSwap(&AfterSwapParams{
			BeforeSwapParams: &BeforeSwapParams{
				CalcOut:         true,
				ZeroForOne:      zeroForOne,
				AmountSpecified: big256.FromBig(amountIn),
				SqrtPriceX96:    &p.V3Pool.SqrtRatioX96,
			},
			AmountIn:         big256.FromBig(amountIn),
			AmountOut:        big256.FromBig(swapResult.TokenAmountOut.Amount),
			BeforeSwapResult: beforeSwapResult,
		})
		if err != nil {
			return nil, fmt.Errorf("[AfterSwap] %w", err)
		} else if err = ValidateAfterSwapResult(afterSwapResult); err != nil {
			return nil, fmt.Errorf("[AfterSwap] validation failed: %w", err)
		}
	}

	return
}

// zeroNilFeeAmount sets the big.Int fee amount to zero. v3 leaves it nil on the big.Int path,
// but callers read it as a number; the uint256 path already reports zero.
func zeroNilFeeAmount(res *pool.CalcAmountOutResult) {
	if res != nil && res.Fee != nil && res.Fee.Amount == nil {
		res.Fee.Amount = new(big.Int)
	}
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

// CalcAmountIn reverses exact-in: hooks get CalcOut=false and must invert their exact-in fees,
// not simulate on-chain exact-out, since the swap is always executed exact-in.
func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (swapResult *pool.CalcAmountInResult, err error) {
	originalTokenOut, originalTokenIn := param.TokenAmountOut.Token, param.TokenIn
	var wrapAdditionalGas int64
	var beforeSwapResult *BeforeSwapResult
	var afterSwapResult *AfterSwapResult
	unwrapIdx := -1 // pool index of the Few token this swap unwraps
	var fewReserve *uint256.Int

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
			FewReserve:   fewReserve,
			FewIdx:       unwrapIdx,
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

	// Wrap/unwrap tokens if needed and calculate wrap gas
	if p.GetTokenIndex(param.TokenAmountOut.Token) == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canWrap := wrapper.CanWrap(p.chainID, param.TokenAmountOut.Token)
			if canWrap {
				param.TokenAmountOut.Token = metadata.GetWrapToken()
				unwrapIdx = p.GetTokenIndex(param.TokenAmountOut.Token)
				wrapAdditionalGas += p.Gas.BaseGas
				break
			}
		}
	} else if p.fewWrapperIdx >= 0 && p.GetTokenIndex(param.TokenIn) == p.fewWrapperIdx {
		unwrapIdx = p.fewWrapperIdx
	}
	if fewReserve, err = p.unwrapFew(unwrapIdx, param.TokenAmountOut.Amount); err != nil {
		return nil, err
	}
	if p.GetTokenIndex(param.TokenIn) == -1 {
		for _, wrapper := range p.tokenWrappers {
			metadata, canUnwrap := wrapper.CanWrap(p.chainID, param.TokenIn)
			if canUnwrap {
				param.TokenIn = metadata.GetWrapToken()
				wrapAdditionalGas += p.Gas.BaseGas
				break
			}
		}
	}

	tokenOut := param.TokenAmountOut.Token
	zeroForOne := p.GetTokenIndex(tokenOut) == 1

	if p.hook == nil {
		swapFee, _ := p.swapFee(zeroForOne, nil)
		swapResult, err = v3CalcAmountIn(p.withSwapFee(swapFee), param)
		return
	}

	amountOut := new(big.Int).Set(param.TokenAmountOut.Amount)

	if p.hook.CanBeforeSwap(p.staticExtra.HooksAddress) {
		if beforeSwapResult, err = p.hook.BeforeSwap(&BeforeSwapParams{
			CalcOut:         false,
			ZeroForOne:      zeroForOne,
			AmountSpecified: big256.FromBig(amountOut),
			SqrtPriceX96:    &p.V3Pool.SqrtRatioX96,
		}); err != nil {
			return nil, fmt.Errorf("[BeforeSwap] %w", err)
		} else if err = ValidateBeforeSwapResult(beforeSwapResult); err != nil {
			return nil, fmt.Errorf("[BeforeSwap] validation failed: %w", err)
		}

		if amountOut.Add(amountOut, big256.ToBig(beforeSwapResult.DeltaSpecified)).Sign() < 0 {
			return nil, ErrInvalidAmountOut
		}
	}

	swapFee, err := p.swapFee(zeroForOne, beforeSwapResult)
	if err != nil {
		return nil, err
	}

	if swapResult, err = v3CalcAmountIn(p.withSwapFee(swapFee), pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		TokenIn: param.TokenIn,
	}); err != nil {
		return nil, err
	}

	if p.hook.CanAfterSwap(p.staticExtra.HooksAddress) {
		if afterSwapResult, err = p.hook.AfterSwap(&AfterSwapParams{
			BeforeSwapParams: &BeforeSwapParams{
				CalcOut:         false,
				ZeroForOne:      zeroForOne,
				AmountSpecified: big256.FromBig(amountOut),
				SqrtPriceX96:    &p.V3Pool.SqrtRatioX96,
			},
			AmountIn:         big256.FromBig(swapResult.TokenAmountIn.Amount),
			AmountOut:        big256.FromBig(amountOut),
			BeforeSwapResult: beforeSwapResult,
		}); err != nil {
			return nil, fmt.Errorf("[AfterSwap] %w", err)
		} else if err = ValidateAfterSwapResult(afterSwapResult); err != nil {
			return nil, fmt.Errorf("[AfterSwap] validation failed: %w", err)
		}
	}

	return
}

// v3CalcAmountIn quotes with the v3 pool's native CalcAmtIn and returns big.Int amounts, since the v3 legacy
// CalcAmountIn is itself a test-only method. An amount above 256 bits is clamped to 2^256-1, which v3 rejects
// as above the reserves, as the big.Int body did.
func v3CalcAmountIn(sim *uniswapv3.PoolSimulator, param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	var amountOut uint256.Int
	if overflow := amountOut.SetFromBig(param.TokenAmountOut.Amount); overflow {
		amountOut.Set(big256.UMax)
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

// unwrapFew returns the underlying balance left in the contract of the Few token at pool index
// idx after unwrapping amount of it, or ErrFewUnwrapExceeded when the balance can't cover it
// (FewWrappedToken.unwrapTo would revert). nil, nil when idx is not a tracked Few token.
func (p *PoolSimulator) unwrapFew(idx int, amount *big.Int) (*uint256.Int, error) {
	if idx < 0 || p.fewReserves[idx] == nil {
		return nil, nil
	}
	amt, overflow := uint256.FromBig(amount)
	if overflow || amt.Gt(p.fewReserves[idx]) {
		return nil, ErrFewUnwrapExceeded
	}
	return amt.Sub(p.fewReserves[idx], amt), nil
}
