package nabla

import (
	"math/big"
	"slices"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/int256"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

type PoolSimulator struct {
	pool.Pool
	Extra
	decimals []uint8
}

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
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
		Extra:    extra,
		decimals: lo.Map(ep.Tokens, func(item *entity.PoolToken, _ int) uint8 { return item.Decimals }),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenIn  = params.TokenAmountIn.Token
		tokenOut = params.TokenOut
	)
	idxIn, idxOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if idxIn < 0 || idxOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn := int256.MustFromBig(params.TokenAmountIn.Amount)
	amountOut, swapInfo, err := sell(s.Pools[idxIn], s.Pools[idxOut], amountIn, s.decimals[idxIn], s.decimals[idxOut])
	if err != nil {
		return nil, err
	}

	if amountOut.Gt(s.Pools[idxOut].State.Reserve) {
		return nil, ErrInsufficientReserves
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: bignumber.ZeroBI,
		},
		SwapInfo: SwapInfo{
			frPoolNewState: swapInfo.frPoolNewState,
			toPoolNewState: swapInfo.toPoolNewState,
		},
	}, nil
}

func sell(fr, to NablaPool, amountIn *int256.Int, frDecimals, toDecimals uint8) (*int256.Int, *SwapInfo, error) {
	if fr.State.Price == nil || to.State.Price == nil {
		return nil, nil, ErrStalePrice
	}

	lpFee := to.Meta.LpFee
	protocolFee := to.Meta.ProtocolFee
	backstopFee := to.Meta.BackstopFee

	var scalingFactor *int256.Int
	if frDecimals > toDecimals {
		scalingFactor = new(int256.Int).Mul(priceScalingFactor, i256.TenPow(uint64(frDecimals-toDecimals)))
	} else if toDecimals > frDecimals {
		scalingFactor = new(int256.Int).Quo(priceScalingFactor, i256.TenPow(uint64(toDecimals-frDecimals)))
	} else {
		scalingFactor = priceScalingFactor
	}

	price := new(int256.Int).Mul(fr.State.Price, pricePrecision)
	price.Quo(price, to.State.Price)

	curveIn := NewCurve(fr.Meta.CurveBeta, fr.Meta.CurveC)
	curveOut := NewCurve(to.Meta.CurveBeta, to.Meta.CurveC)

	effectiveAmountIn := curveIn.InverseHorizontal(fr.State.Reserve, fr.State.TotalLiabilities,
		new(int256.Int).Add(fr.State.ReserveWithSlippage, amountIn), int64(frDecimals))

	temp0 := new(int256.Int).Add(fr.State.Reserve, effectiveAmountIn)

	temp1 := new(int256.Int).Mul(i1990, fr.State.TotalLiabilities)
	temp1.Quo(temp1, i1e3)

	if temp0.Gt(temp1) {
		return nil, nil, ErrZeroSwap
	}

	rawAmountOut := new(int256.Int).Mul(effectiveAmountIn, price)
	rawAmountOut.Quo(rawAmountOut, scalingFactor)

	bspFeeAmount := new(int256.Int).Mul(rawAmountOut, backstopFee)
	bspFeeAmount.Quo(bspFeeAmount, feePrecision)

	protocolFeeAmount := new(int256.Int).Mul(rawAmountOut, protocolFee)
	protocolFeeAmount.Quo(protocolFeeAmount, feePrecision)

	maxLpFee := new(int256.Int).Mul(rawAmountOut, lpFee)
	maxLpFee.Quo(maxLpFee, feePrecision)

	reducedReserveOut := new(int256.Int).Sub(to.State.Reserve, rawAmountOut)
	reducedReserveOut.Add(reducedReserveOut, bspFeeAmount)
	reducedReserveOut.Add(reducedReserveOut, protocolFeeAmount)

	actualLpFeeAmount := curveOut.InverseDiagonal(
		reducedReserveOut, to.State.TotalLiabilities, to.State.ReserveWithSlippage, int64(toDecimals),
	)
	if actualLpFeeAmount.Gt(maxLpFee) {
		actualLpFeeAmount = maxLpFee
	}

	actualReducedReserveOut := new(int256.Int).Add(reducedReserveOut, actualLpFeeAmount)
	actualTotalLiabilitiesOut := new(int256.Int).Add(to.State.TotalLiabilities, actualLpFeeAmount)

	reserveWithSlippageAfterAmountOut := curveOut.Psi(
		actualReducedReserveOut, actualTotalLiabilitiesOut, int64(toDecimals),
	)

	if reserveWithSlippageAfterAmountOut.Gt(to.State.ReserveWithSlippage) {
		reserveWithSlippageAfterAmountOut = to.State.ReserveWithSlippage
	}

	minReserveWithSlippageAfterAmountOut := new(int256.Int).Mul(i1e4, to.State.TotalLiabilities)
	minReserveWithSlippageAfterAmountOut.Quo(minReserveWithSlippageAfterAmountOut, i1e6)

	if reserveWithSlippageAfterAmountOut.Lte(minReserveWithSlippageAfterAmountOut) {
		return nil, nil, ErrZeroSwap
	}

	amountOut := new(int256.Int).Sub(to.State.ReserveWithSlippage, reserveWithSlippageAfterAmountOut)

	newInputReserve := new(int256.Int).Add(fr.State.Reserve, effectiveAmountIn)
	newInputReserveWithSlippage := curveIn.Psi(newInputReserve, fr.State.TotalLiabilities, int64(frDecimals))

	return amountOut, &SwapInfo{
		frPoolNewState: NablaPoolState{
			Reserve:             newInputReserve,
			ReserveWithSlippage: newInputReserveWithSlippage,
			TotalLiabilities:    fr.State.TotalLiabilities,
			Price:               fr.State.Price,
		},
		toPoolNewState: NablaPoolState{
			Reserve:             actualReducedReserveOut,
			ReserveWithSlippage: reserveWithSlippageAfterAmountOut,
			TotalLiabilities:    actualTotalLiabilitiesOut,
			Price:               to.State.Price,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warnf("failed to update balance for nabla pool")
		return
	}

	idxIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	idxOut := s.GetTokenIndex(params.TokenAmountOut.Token)
	if idxIn < 0 || idxOut < 0 {
		return
	}

	poolIn := s.Pools[idxIn]
	poolIn.State = swapInfo.frPoolNewState
	s.Pools[idxIn] = poolIn

	poolOut := s.Pools[idxOut]
	poolOut.State = swapInfo.toPoolNewState
	s.Pools[idxOut] = poolOut
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return Meta{
		BlockNumber:     s.Info.BlockNumber,
		PriceUpdateData: s.PriceFeedData,
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Pools = slices.Clone(s.Pools)
	return &cloned
}
