package tricryptong

import (
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
	if tokenIndexFrom < 0 || tokenIndexTo < 0 {
		return nil, fmt.Errorf("tokenIndexFrom %v or tokenIndexTo %v is not correct",
			tokenIndexFrom, tokenIndexTo)
	}

	var amountOut, fee, amount uint256.Int
	amount.SetFromBig(tokenAmountIn.Amount)
	var swapInfo SwapInfo
	err := t.GetDy(
		tokenIndexFrom,
		tokenIndexTo,
		&amount,
		&amountOut, &fee, &swapInfo.K0, swapInfo.Xp[:],
	)
	if err != nil {
		return nil, err
	} else if amountOut.IsZero() {
		return nil, ErrZero
	}
	A, gamma := t._A_gamma()
	if err = t.tweak_price(A, gamma, swapInfo.Xp, nil, &swapInfo.K0,
		swapInfo.LastPrices[:], swapInfo.PriceScale[:], &swapInfo.XcpProfit, &swapInfo.D,
		&swapInfo.VirtualPrice); err != nil {
		return nil, errors.WithMessage(err, "tweak price")
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: fee.ToBig(),
		},
		Gas:      t.gas,
		SwapInfo: swapInfo,
	}, nil
}
