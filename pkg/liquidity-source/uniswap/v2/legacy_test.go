package uniswapv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	reserveIn := s.reserves[indexIn]
	reserveOut := s.reserves[indexOut]

	// Pair receives input net of sell tax; user receives output net of buy tax.
	// Reserves move by the pair-side amounts (effectiveAmountIn / grossAmountOut), carried in SwapInfo.
	effectiveAmountIn := s.taxHandler.ApplySellTax(s.Info.Tokens[indexIn], amountIn)

	// The pair keeps balances in uint112; a swap that pushes the input-side balance past 2^112-1
	// reverts on-chain with OVERFLOW. Headroom can be near zero for huge-supply tokens.
	if CheckUint112Overflow(reserveIn, effectiveAmountIn) {
		return nil, ErrUint112Overflow
	}

	grossAmountOut := s.getAmountOut(effectiveAmountIn, reserveIn, reserveOut)
	if !grossAmountOut.Lt(reserveOut) {
		return nil, ErrInsufficientLiquidity
	} else if grossAmountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutputAmount
	}

	amountOut := s.taxHandler.ApplyBuyTax(s.Info.Tokens[indexOut], grossAmountOut)
	if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutputAmount
	}

	result := &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: bignumber.ZeroBI},
		Gas:            lo.CoalesceOrEmpty(s.Gas, defaultGas+extraGasByExchange[s.GetExchange()]),
	}
	if s.taxHandler.HasSellTax(s.Info.Tokens[indexIn]) ||
		s.taxHandler.HasBuyTax(s.Info.Tokens[indexOut]) {
		result.SwapInfo = SwapInfo{
			EffectiveAmountIn: effectiveAmountIn.ToBig(),
			GrossAmountOut:    grossAmountOut.ToBig(),
		}
	}
	return result, nil
}
