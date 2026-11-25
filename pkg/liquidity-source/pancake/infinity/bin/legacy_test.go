package bin

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := params.TokenAmountIn, params.TokenOut
	indexIn, indexOut := p.GetTokenIndex(tokenIn.Token), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, shared.ErrInvalidToken
	}

	if p.activeId == 0 {
		return nil, shared.ErrUninitializedPool
	}

	amountIn := tokenIn.Amount
	// A native-flagged side trades at native's own decimals on-chain; only Arc's differ (18 vs 6).
	if p.isNative[indexIn] {
		amountIn = valueobject.UnwrapNativeAmount(p.chainID, amountIn)
	}

	res, err := p.swap(true, indexIn == 0, amountIn)
	if err != nil {
		return nil, err
	}

	amountOut := res.Amount.ToBig()
	if p.isNative[indexOut] {
		amountOut = valueobject.WrapNativeAmount(p.chainID, amountOut)
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn.Token,
			Amount: res.Fee.ToBig(),
		},
		Gas: p.gas(),
		SwapInfo: SwapInfo{
			NewActiveID:        res.NewActiveID,
			BinsReserveChanges: res.BinsReserveChanges,
		},
	}, nil
}
