package feltir

import (
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenOut := strings.ToLower(params.TokenOut)
	amountOut, err := s.quote(params.TokenAmountIn.Token, tokenOut, params.TokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}

	if limit := params.Limit; limit != nil {
		if amountOut.Cmp(limit.GetLimit(tokenOut)) > 0 {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: strings.ToLower(params.TokenAmountIn.Token), Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
	}, nil
}
