package carbon

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
	"github.com/rs/zerolog/log"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := params.TokenAmountIn.Token
	tokenOut := params.TokenOut

	if params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmount
	}

	if len(s.Strategies) == 0 {
		return nil, ErrNoStrategies
	}

	tokenInIdx := s.GetTokenIndex(tokenIn)
	tokenOutIdx := s.GetTokenIndex(tokenOut)

	if tokenInIdx < 0 || tokenOutIdx < 0 {
		return nil, ErrInvalidToken
	}

	isToken0To1 := tokenOutIdx == orderIdxToken1
	targetOrderIdx := tokenOutIdx

	ordersMap := make(EncodedOrderMap)
	strategyIdxMap := make(map[string]int)
	for strategyIdx, strategy := range s.Strategies {
		targetOrder := &strategy.Orders[targetOrderIdx]

		if targetOrder.Y != nil && targetOrder.Y.Sign() > 0 {
			strategyIdStr := strategy.Id.String()
			ordersMap[strategyIdStr] = targetOrder
			strategyIdxMap[strategyIdStr] = strategyIdx
		}
	}
	if len(ordersMap) == 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)

	tradeResults, err := s.trade(
		amountIn, ordersMap, strategyIdxMap,
		isToken0To1,
		[]MatchType{MatchTypeBest, MatchTypeFast},
	)
	if err != nil {
		return nil, err
	}

	output := tradeResults.Best
	if output == nil {
		output = tradeResults.Fast
	}

	if output == nil {
		return nil, ErrInsufficientLiquidity
	}

	swapInfo := SwapInfo{}
	swapInfo.TradeActions = output.TradeActions

	if tradeResults.Fast != nil {
		// If fast amount is greater than best amount
		if tradeResults.Best != nil && tradeResults.Fast.AmountOutAfterFee.Gt(tradeResults.Best.AmountOutAfterFee) {
			log.Debug().Str("pool", s.Info.Address).
				Stringer("bestAmountOut", tradeResults.Best.AmountOutAfterFee).
				Stringer("fastAmountOut", tradeResults.Fast.AmountOutAfterFee).
				Msg("Fast result has greater output than Best result")

			output = tradeResults.Fast
			swapInfo.TradeActions = output.TradeActions
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: output.AmountOutAfterFee.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: output.FeeAmount.ToBig(),
		},
		SwapInfo: swapInfo,
		Gas:      defaultTradeBySourceAmountGas + int64(defaultSingleTradeActionGas*len(output.TradeActions)),
	}, nil
}
