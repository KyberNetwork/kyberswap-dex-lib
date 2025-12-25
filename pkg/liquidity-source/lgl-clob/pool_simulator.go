package lglclob

import (
	"math"
	"math/big"
	"slices"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	swapFee *uint256.Int
	*OrderBook
	*StaticExtra
	cumAmtOutF float64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var orderBook OrderBook
	if err := json.Unmarshal([]byte(entityPool.Extra), &orderBook); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens:   []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address},
			Reserves: []*big.Int{bignumber.NewBig10(entityPool.Reserves[0]),
				bignumber.NewBig10(entityPool.Reserves[1])},
			BlockNumber: entityPool.BlockNumber,
		}},
		swapFee:     uint256.NewInt(uint64(entityPool.SwapFee * 1e18)),
		OrderBook:   &orderBook,
		StaticExtra: &staticExtra,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (swapResult *pool.CalcAmountOutResult,
	err error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenIn := tokenAmountIn.Token
	amtIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmount
	}
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	// tokenOut is tokenX means buy, tokenOut is tokenY means sell
	var levels *OrderBookLevels
	var scalingFactorIn, scalingFactorOut *uint256.Int
	isBuy := indexOut == 0
	if isBuy {
		levels = &p.Asks
		scalingFactorIn = p.ScalingFactorY
		scalingFactorOut = p.ScalingFactorX
	} else {
		levels = &p.Bids
		scalingFactorIn = p.ScalingFactorX
		scalingFactorOut = p.ScalingFactorY
	}
	if len(levels.ArrayPrices) == 0 {
		return nil, ErrEmptyOrders
	}

	// for buys fees deducted from tokenIn (tokenY), for sells fees deducted from result tokenOut (tokenY)
	var availableAmountIn, tmp uint256.Int
	availableAmountIn.Set(amtIn)
	if isBuy {
		availableAmountIn.MulDivOverflow(&availableAmountIn, big256.BONE, tmp.Add(big256.BONE, p.swapFee))
	}

	scaledAmountIn := availableAmountIn.Div(&availableAmountIn, scalingFactorIn)
	var executedValue, executedScaledAmountOut, executedScaledAmountIn uint256.Int

	var executedLevels int
	var executionDone bool
	var executedShares uint256.Int
	for i, price := range levels.ArrayPrices {
		shares := levels.ArrayShares[i] // in tokenX
		executedLevels++

		if isBuy {
			executedShares.Div(scaledAmountIn, price) // in tokenX
		} else {
			executedShares.Set(scaledAmountIn) // in tokenX
		}
		if executedShares.IsZero() { // scaledAmountIn/price can be 0 i.e. remaining>0
			break
		}
		if executionDone = !executedShares.Gt(shares); !executionDone { // level not enough shares, fully used
			executedShares.Set(shares) // in tokenX
		}
		executedValue.Mul(&executedShares, price) // in tokenY

		if isBuy {
			scaledAmountIn.Sub(scaledAmountIn, &executedValue)                     // in tokenY
			executedScaledAmountOut.Add(&executedScaledAmountOut, &executedShares) // in tokenX
			executedScaledAmountIn.Add(&executedScaledAmountIn, &executedValue)    // in tokenY
		} else {
			scaledAmountIn.Sub(scaledAmountIn, &executedShares)                   // in tokenX
			executedScaledAmountOut.Add(&executedScaledAmountOut, &executedValue) // in tokenY
			executedScaledAmountIn.Add(&executedScaledAmountIn, &executedShares)  // in tokenX
		}

		if executionDone {
			break
		}
	}
	// 1:190576 2:236653 3:267273 4:269108 5:272492 6:244971 8:241112 9:227946 10:237471
	// 1:494222 2:497402 after latest update
	gas := int64(197346*math.Log(float64(executedLevels+1)/2) + 494222)
	if executedShares.Eq(levels.ArrayShares[executedLevels-1]) {
		executedShares.Clear()
		executedLevels++
	}

	executedAmountIn := executedScaledAmountIn.Mul(&executedScaledAmountIn, scalingFactorIn)
	executedAmountOut := executedScaledAmountOut.Mul(&executedScaledAmountOut, scalingFactorOut)
	reserveOutF, _ := p.GetReserves()[indexOut].Float64()
	if p.cumAmtOutF+executedAmountOut.Float64() > reserveOutF*safetyBuffer {
		return nil, ErrExceededSafetyBuffer
	}

	var feesTokenY, remainingAmountIn *uint256.Int
	priceLimit := levels.ArrayPrices[len(levels.ArrayPrices)-1]
	if isBuy {
		feesTokenY = big256.MulWadUp(&tmp, executedAmountIn, p.swapFee)
		remainingAmountIn = amtIn.Sub(amtIn, executedAmountIn).Sub(amtIn, feesTokenY)
		priceLimit = big256.MulDivUp(&executedValue, priceLimit, uPriceLimitMultiplier, big256.UBasisPoint)
	} else {
		feesTokenY = big256.MulWadUp(&tmp, executedAmountOut, p.swapFee)
		executedAmountOut.Sub(executedAmountOut, feesTokenY)
		remainingAmountIn = amtIn.Sub(amtIn, executedAmountIn)
		priceLimit = big256.MulDivDown(&executedValue, priceLimit, big256.UBasisPoint, uPriceLimitMultiplier)
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: tokenOut, Amount: executedAmountOut.ToBig()},
		Fee:                    &pool.TokenAmount{Token: p.Info.Tokens[1], Amount: feesTokenY.ToBig()},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: remainingAmountIn.ToBig()},
		Gas:                    gas,
		SwapInfo: SwapInfo{
			executedLevels:     executedLevels,
			lastExecutedShares: &executedShares,
			HasNative:          p.SupportsNativeEth,
			PriceLimit:         priceLimit,
		},
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.OrderBook = lo.ToPtr(*p.OrderBook)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for OnchainClob pool, wrong swapInfo type")
		return
	}

	isBuy := p.GetTokenIndex(params.TokenAmountOut.Token) == 0
	levels := lo.Ternary(isBuy, &p.Asks, &p.Bids)
	fullyExecutedLevels := si.executedLevels - 1
	levels.ArrayPrices = levels.ArrayPrices[fullyExecutedLevels:]
	levels.ArrayShares = levels.ArrayShares[fullyExecutedLevels:]
	if !si.lastExecutedShares.IsZero() {
		levels.ArrayShares = slices.Clone(levels.ArrayShares)
		levels.ArrayShares[0] = new(uint256.Int).Sub(levels.ArrayShares[0], si.lastExecutedShares)
	}
	amtOutF, _ := params.TokenAmountOut.Amount.Float64()
	p.cumAmtOutF += amtOutF
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return nil
}
