package xpress

import (
	"math/big"
	"slices"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	swapFee *uint256.Int
	*OrderBook
	*StaticExtra
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
		levels = &OrderBookLevels{
			ArrayPrices: slices.Clone(p.Asks.ArrayPrices),
			ArrayShares: slices.Clone(p.Asks.ArrayShares),
		}
		scalingFactorIn = p.ScalingFactorY
		scalingFactorOut = p.ScalingFactorX
	} else {
		levels = &OrderBookLevels{
			ArrayPrices: slices.Clone(p.Bids.ArrayPrices),
			ArrayShares: slices.Clone(p.Bids.ArrayShares),
		}
		scalingFactorIn = p.ScalingFactorX
		scalingFactorOut = p.ScalingFactorY
	}

	// for buys fees deducted from tokenIn (tokenY), for sells fees deducted from result tokenOut (tokenY)
	var availableAmountIn, tmp uint256.Int
	availableAmountIn.Set(amtIn)
	if isBuy {
		availableAmountIn.MulDivOverflow(&availableAmountIn, big256.BONE, tmp.Add(big256.BONE, p.swapFee))
	}

	scaledAmountIn := availableAmountIn.Div(&availableAmountIn, scalingFactorIn)
	var executedValue, executedScaledAmountOut, executedScaledAmountIn uint256.Int

	for i, price := range levels.ArrayPrices {
		shares := levels.ArrayShares[i] // in tokenX

		var maxSharesIn *uint256.Int
		if isBuy {
			maxSharesIn = tmp.Div(scaledAmountIn, price) // in tokenX
		} else {
			maxSharesIn = tmp.Set(scaledAmountIn) // in tokenX
		}
		executedShares := big256.Min(shares, maxSharesIn) // in tokenX
		executedValue.Mul(executedShares, price)          // in tokenY

		if isBuy {
			scaledAmountIn.Sub(scaledAmountIn, &executedValue)                    // in tokenY
			executedScaledAmountOut.Add(&executedScaledAmountOut, executedShares) // in tokenX
			executedScaledAmountIn.Add(&executedScaledAmountIn, &executedValue)   // in tokenY
		} else {
			scaledAmountIn.Sub(scaledAmountIn, executedShares)                    // in tokenX
			executedScaledAmountOut.Add(&executedScaledAmountOut, &executedValue) // in tokenY
			executedScaledAmountIn.Add(&executedScaledAmountIn, executedShares)   // in tokenX
		}

		levels.ArrayShares[i].Sub(shares, executedShares)

		if scaledAmountIn.Sign() == 0 { // fully executed
			break
		}
	}

	executedAmountIn := executedScaledAmountIn.Mul(&executedScaledAmountIn, scalingFactorIn)
	executedAmountOut := executedScaledAmountOut.Mul(&executedScaledAmountOut, scalingFactorOut)

	var feesTokenY, remainingAmountIn *uint256.Int
	if isBuy {
		feesTokenY = big256.MulWadUp(&tmp, executedAmountIn, p.swapFee)
		remainingAmountIn = amtIn.Sub(amtIn, executedAmountIn).Sub(amtIn, feesTokenY)
	} else {
		feesTokenY = big256.MulWadUp(&tmp, executedAmountOut, p.swapFee)
		remainingAmountIn = amtIn.Sub(amtIn, executedAmountIn)
		executedAmountOut.Sub(executedAmountOut, feesTokenY)
	}

	var updatedOrderBook *OrderBook
	if isBuy {
		updatedOrderBook = &OrderBook{
			Bids: p.Bids,
			Asks: *p.removeExecutedLevels(levels),
		}
	} else {
		updatedOrderBook = &OrderBook{
			Bids: *p.removeExecutedLevels(levels),
			Asks: p.Asks,
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: tokenOut, Amount: executedAmountOut.ToBig()},
		Fee:                    &pool.TokenAmount{Token: p.Info.Tokens[1], Amount: feesTokenY.ToBig()},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: remainingAmountIn.ToBig()},
		Gas:                    DefaultGas,
		SwapInfo: SwapInfo{
			UpdatedOrderBook: updatedOrderBook,
		},
	}, nil
}

func (p *PoolSimulator) removeExecutedLevels(levels *OrderBookLevels) *OrderBookLevels {
	for len(levels.ArrayShares) > 0 {
		if levels.ArrayShares[0].IsZero() {
			levels.ArrayShares = levels.ArrayShares[1:]
			levels.ArrayPrices = levels.ArrayPrices[1:]
		} else {
			break
		}
	}
	return levels
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.OrderBook = &OrderBook{
		Bids: OrderBookLevels{
			ArrayPrices: slices.Clone(p.Bids.ArrayPrices),
			ArrayShares: slices.Clone(p.Bids.ArrayShares),
		},
		Asks: OrderBookLevels{
			ArrayPrices: slices.Clone(p.Asks.ArrayPrices),
			ArrayShares: slices.Clone(p.Asks.ArrayShares),
		},
	}
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for OnchainClob pool, wrong swapInfo type")
		return
	}
	p.OrderBook = si.UpdatedOrderBook
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return nil
}
