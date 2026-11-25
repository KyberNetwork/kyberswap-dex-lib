package orderbook

import (
	"math"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut, amtIn := params.TokenAmountIn.Token, params.TokenOut, params.TokenAmountIn.Amount
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	result, err := p.calcOut(amtIn, p.tokens[indexIn], p.tokens[indexOut], p.levelsFroms[indexIn], p.minTrades[indexIn])
	if err != nil {
		return nil, err
	}

	if limit := params.Limit; limit != nil {
		inventoryLimit := limit.GetLimit(tokenOut)
		if result.TokenAmountOut.Amount.Cmp(inventoryLimit) > 0 {
			return nil, ErrSwapLimitExceeded
		}
	}
	return result, nil
}
func (p *PoolSimulator) calcOut(amountIn *big.Int, tokenIn, tokenOut *entity.PoolToken, priceLevel []Level,
	minTrade float64) (*pool.CalcAmountOutResult, error) {
	amtInF, _ := amountIn.Float64()
	amtInAfterDecimals := amtInF / math.Pow10(int(tokenIn.Decimals))
	amtOutAfterDecimals, levels, err := getAmountOut(amtInAfterDecimals, priceLevel, minTrade)
	if err != nil {
		return nil, err
	}

	amtOutF := amtOutAfterDecimals * math.Pow10(int(tokenOut.Decimals))
	amtOutF *= 1 - p.fee
	amountOut, _ := big.NewFloat(amtOutF).Int(nil)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut.Address, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: tokenIn.Address, Amount: bignumber.ZeroBI},
		Gas:            p.Gas.Base + levels*p.Gas.Level,
	}, nil
}
