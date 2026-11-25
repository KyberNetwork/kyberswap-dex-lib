package ringswap

import (
	"math/big"

	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	amountIn, overflow := uint256.FromBig(param.TokenAmountIn.Amount)
	if overflow {
		return nil, uniswapv2.ErrInvalidAmountIn
	}

	var getLimit func(string) *uint256.Int
	if param.Limit != nil {
		getLimit = func(key string) *uint256.Int { return toU256(param.Limit.GetLimit(key)) }
	}
	res, err := s.calcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{
			Token:     param.TokenAmountIn.Token,
			Amount:    *amountIn,
			AmountUsd: param.TokenAmountIn.AmountUsd,
		},
		TokenOut: param.TokenOut,
	}, getLimit)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: res.TokenAmountOut.Token, Amount: res.TokenAmountOut.Amount.ToBig()},
		Fee:            &pool.TokenAmount{Token: res.Fee.Token, Amount: res.Fee.Amount.ToBig()},
		Gas:            res.Gas,
		SwapInfo:       res.SwapInfo,
	}, nil
}
func toU256(b *big.Int) *uint256.Int {
	if b == nil {
		return nil
	}
	return uint256.MustFromBig(b)
}
