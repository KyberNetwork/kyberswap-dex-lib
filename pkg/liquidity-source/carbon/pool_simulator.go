package carbon

import (
	"math/big"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra
}

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},
		Extra:       extra,
		StaticExtra: staticExtra,
	}, nil
}

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
			logger.WithFields(logger.Fields{
				"pool":          s.Info.Address,
				"bestAmountOut": tradeResults.Best.AmountOutAfterFee.String(),
				"fastAmountOut": tradeResults.Fast.AmountOutAfterFee.String(),
			}).Warn("Fast result has greater output than Best result")

			swapInfo.FastTradeActions = tradeResults.Fast.TradeActions
			swapInfo.FastAmount = tradeResults.Fast.AmountOutAfterFee

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

func (s *PoolSimulator) trade(amountIn *uint256.Int, ordersMap EncodedOrderMap,
	strategyIdxMap map[string]int, isToken0To1 bool, matchTypes []MatchType) (*TradeResults, error) {

	targetOrderIdx := lo.Ternary(isToken0To1, orderIdxToken1, orderIdxToken0)
	sourceOrderIdx := lo.Ternary(isToken0To1, orderIdxToken0, orderIdxToken1)

	matchResult := MatchBySourceAmount(amountIn, ordersMap, matchTypes, nil)

	results := &TradeResults{}
	ppmMinusFee := uint256.NewInt(ppmResolution - uint64(s.TradingFeePpm))

	for _, matchType := range matchTypes {
		var actions []*MatchAction
		switch matchType {
		case MatchTypeBest:
			actions = matchResult.Best
		case MatchTypeFast:
			actions = matchResult.Fast
		}

		if len(actions) == 0 {
			continue
		}

		output := s.processMatchActions(actions, strategyIdxMap, targetOrderIdx, sourceOrderIdx, isToken0To1, ppmMinusFee)
		if output == nil {
			continue
		}

		switch matchType {
		case MatchTypeBest:
			results.Best = output
		case MatchTypeFast:
			results.Fast = output
		}
	}

	if results.Best == nil && results.Fast == nil {
		return nil, ErrInvalidSwap
	}

	return results, nil
}

func (s *PoolSimulator) processMatchActions(
	actions []*MatchAction,
	strategyIdxMap map[string]int,
	targetOrderIdx, sourceOrderIdx int,
	isToken0To1 bool,
	ppmMinusFee *uint256.Int,
) *TradeOutput {
	totalAmountOut := u256.New0()

	var tradeActions []TradeAction
	for _, action := range actions {
		strategyIdStr := action.Id

		strategyIdx, ok := strategyIdxMap[strategyIdStr]
		if !ok {
			continue
		}

		strategy := &s.Strategies[strategyIdx]
		targetOrder := &strategy.Orders[targetOrderIdx]
		sourceOrder := &strategy.Orders[sourceOrderIdx]

		newTargetY := new(uint256.Int).Sub(targetOrder.Y, action.Output)

		newSourceY := new(uint256.Int).Add(sourceOrder.Y, action.Input)
		newSourceZ := new(uint256.Int).Set(sourceOrder.Z)
		if newSourceY.Gt(sourceOrder.Z) {
			newSourceZ.Set(newSourceY)
		}

		tradeActions = append(tradeActions, TradeAction{
			StrategyId:      strategyIdStr,
			strategyIdx:     strategyIdx,
			isToken0To1:     isToken0To1,
			SourceAmount:    action.Input,
			TargetAmount:    action.Output,
			newTargetOrderY: newTargetY,
			newSourceOrderY: newSourceY,
			newSourceOrderZ: newSourceZ,
		})

		totalAmountOut.Add(totalAmountOut, action.Output)
	}

	if totalAmountOut.Sign() <= 0 {
		return nil
	}

	amountOutAfterFee := new(uint256.Int).Mul(totalAmountOut, ppmMinusFee)
	amountOutAfterFee.Div(amountOutAfterFee, uPmmResolution)

	feeAmount := new(uint256.Int).Sub(totalAmountOut, amountOutAfterFee)

	return &TradeOutput{
		AmountOutAfterFee: amountOutAfterFee,
		FeeAmount:         feeAmount,
		TradeActions:      tradeActions,
	}
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}

	for _, info := range swapInfo.TradeActions {
		if info.strategyIdx < 0 || info.strategyIdx >= len(s.Strategies) {
			continue
		}

		targetIdx := lo.Ternary(info.isToken0To1, orderIdxToken1, orderIdxToken0)
		sourceIdx := lo.Ternary(info.isToken0To1, orderIdxToken0, orderIdxToken1)

		s.Strategies[info.strategyIdx].Orders[targetIdx].Y = info.newTargetOrderY
		s.Strategies[info.strategyIdx].Orders[sourceIdx].Y = info.newSourceOrderY
		s.Strategies[info.strategyIdx].Orders[sourceIdx].Z = info.newSourceOrderZ
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Strategies = make([]Strategy, len(s.Strategies))
	for i := range s.Strategies {
		cloned.Strategies[i] = s.Strategies[i].Clone()
	}

	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	isToken0In := s.GetTokenIndex(tokenIn) == orderIdxToken0
	return Meta{
		BlockNumber:     s.Info.BlockNumber,
		IsNativeIn:      valueobject.IsNative(lo.Ternary(isToken0In, s.Token0, s.Token1)),
		IsNativeOut:     valueobject.IsNative(lo.Ternary(isToken0In, s.Token1, s.Token0)),
		ApprovalAddress: s.Controller,
	}
}
