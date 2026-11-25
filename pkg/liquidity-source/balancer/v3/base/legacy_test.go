package base

import (
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut
	tokenIn := tokenAmountIn.Token
	if tokenIn == tokenOut {
		return nil, shared.ErrInvalidToken
	}

	if p.paused {
		return nil, shared.ErrPoolIsPaused
	}

	indexIn, isTokenInUnderlying, err := p.ResolveToken(tokenIn)
	if err != nil {
		return nil, err
	}
	indexOut, isTokenOutUnderlying, err := p.ResolveToken(tokenOut)
	if err != nil {
		return nil, err
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, shared.ErrInvalidAmountIn
	}

	// Check if this is a same-index underlying/wrapped token conversion
	if p.isBufferSwap(indexIn, indexOut, isTokenInUnderlying, isTokenOutUnderlying) {
		amountOut, err := p.handleBufferConversion(indexIn, amountIn, isTokenInUnderlying, false)
		if err != nil {
			return nil, err
		}

		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: amountOut.ToBig(),
			},
			Fee: &pool.TokenAmount{
				Token:  tokenIn,
				Amount: bignumber.ZeroBI, // No swap fee for direct conversions
			},
			SwapInfo: shared.SwapInfo{
				AggregateFee: bignumber.ZeroBI, // No aggregate fee for direct conversions
			},
			Gas: bufferGas,
		}, nil
	}

	gas := p.BaseGas()
	if isTokenInUnderlying {
		if indexIn >= len(p.buffers) || p.buffers[indexIn] == nil {
			return nil, fmt.Errorf("buffer not found for token %s at index %d", tokenIn, indexIn)
		}

		amountIn, err = p.buffers[indexIn].ConvertToShares(amountIn, false)
		if err != nil {
			return nil, err
		}
		gas += bufferGas
	}

	amountOut, totalSwapFee, aggregateFee, err := p.vault.Swap(shared.VaultSwapParams{
		Kind:           shared.ExactIn,
		IndexIn:        indexIn,
		IndexOut:       indexOut,
		AmountGivenRaw: amountIn,
	}, p.OnSwap)
	if err != nil {
		return nil, err
	}

	var amountOutBeforeConvert uint256.Int
	amountOutBeforeConvert.Set(amountOut)

	if isTokenOutUnderlying {
		if indexOut >= len(p.buffers) || p.buffers[indexOut] == nil {
			return nil, fmt.Errorf("buffer not found for token %s at index %d", tokenOut, indexOut)
		}

		amountOut, err = p.buffers[indexOut].ConvertToAssets(amountOut, false)
		if err != nil {
			return nil, err
		}
		gas += bufferGas
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: totalSwapFee.ToBig(),
		},
		SwapInfo: shared.SwapInfo{
			AggregateFee:           aggregateFee.ToBig(),
			AmountOutBeforeConvert: amountOutBeforeConvert.ToBig(),
		},
		Gas: gas,
	}, nil
}
