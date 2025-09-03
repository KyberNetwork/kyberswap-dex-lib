package xpress

import (
	"encoding/json"
	"math/big"
	"slices"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

type PoolSimulator struct {
	pool.Pool
	OrderBook *OrderBook
	LobConfig *LobConfig
}

var _ = pool.RegisterFactory1(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var orderBook OrderBook
	if err := json.Unmarshal([]byte(entityPool.Extra), &orderBook); err != nil {
		return nil, err
	}

	var lobConfig LobConfig
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &lobConfig); err != nil {
		return nil, err
	}

	var swapFeeFl = new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), bignumber.BoneFloat)
	var swapFee, _ = swapFeeFl.Int(nil)

	info := pool.PoolInfo{
		Address:     entityPool.Address,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address},
		Reserves:    []*big.Int{bignumber.NewBig10(entityPool.Reserves[0]), bignumber.NewBig10(entityPool.Reserves[1])},
		SwapFee:     swapFee,
		BlockNumber: entityPool.BlockNumber,
	}

	return &PoolSimulator{
		Pool:      pool.Pool{Info: info},
		OrderBook: &orderBook,
		LobConfig: &lobConfig,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (swapResult *pool.CalcAmountOutResult, err error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	// TODO: check tokenIn and tokenOut is correct

	// tokenOut is tokenX means buy, tokenOut is tokenY means sell
	var isBuy = common.HexToAddress(param.TokenOut).Cmp(p.LobConfig.TokenX) == 0
	var levels *OrderBookLevels
	var scalingFactorIn *big.Int
	var scalingFactorOut *big.Int

	if isBuy {
		levels = &OrderBookLevels{
			ArrayPrices: slices.Clone(p.OrderBook.Asks.ArrayPrices),
			ArrayShares: slices.Clone(p.OrderBook.Asks.ArrayShares),
		}
		scalingFactorIn = p.LobConfig.ScalingFactorTokenY
		scalingFactorOut = p.LobConfig.ScalingFactorTokenX
	} else {
		levels = &OrderBookLevels{
			ArrayPrices: slices.Clone(p.OrderBook.Bids.ArrayPrices),
			ArrayShares: slices.Clone(p.OrderBook.Bids.ArrayShares),
		}
		scalingFactorIn = p.LobConfig.ScalingFactorTokenX
		scalingFactorOut = p.LobConfig.ScalingFactorTokenY
	}

	// for buys fees deducted from tokenIn (tokenY), for sells fees deducted from result tokenOut (tokenY)
	var availableAmountIn = new(big.Int).Set(tokenAmountIn.Amount)

	if isBuy {
		availableAmountIn.Mul(availableAmountIn, bignumber.BONE)
		availableAmountIn.Div(availableAmountIn, new(big.Int).Add(bignumber.BONE, p.Info.SwapFee))
	}

	var scaledAmountIn = new(big.Int).Div(availableAmountIn, scalingFactorIn)
	var executedScaledAmountOut = new(big.Int)
	var executedScaledAmountIn = new(big.Int)

	for i := 0; i < len(levels.ArrayPrices); i++ {
		price := levels.ArrayPrices[i]
		shares := levels.ArrayShares[i] // in tokenX

		var maxSharesIn *big.Int
		if isBuy {
			maxSharesIn = new(big.Int).Div(scaledAmountIn, price) // in tokenX
		} else {
			maxSharesIn = scaledAmountIn // in tokenX
		}

		executedShares := new(big.Int).Set(bignumber.Min(shares, maxSharesIn)) // in tokenX
		executedValue := new(big.Int).Mul(executedShares, price)               // in tokenY

		if isBuy {
			scaledAmountIn.Sub(scaledAmountIn, executedValue)                    // in tokenY
			executedScaledAmountOut.Add(executedScaledAmountOut, executedShares) // in tokenX
			executedScaledAmountIn.Add(executedScaledAmountIn, executedValue)    // in tokenY
		} else {
			scaledAmountIn.Sub(scaledAmountIn, executedShares)                  // in tokenX
			executedScaledAmountOut.Add(executedScaledAmountOut, executedValue) // in tokenY
			executedScaledAmountIn.Add(executedScaledAmountIn, executedShares)  // in tokenX
		}

		levels.ArrayShares[i] = new(big.Int).Sub(shares, executedShares)

		// check if amountIn is fully executed
		if scaledAmountIn.Cmp(bignumber.ZeroBI) == 0 {
			break
		}
	}

	executedAmountIn := new(big.Int).Mul(executedScaledAmountIn, scalingFactorIn)
	executedAmountOut := new(big.Int).Mul(executedScaledAmountOut, scalingFactorOut)

	var feesTokenY *big.Int
	var remainingAmountIn *big.Int
	if isBuy {
		feesTokenY = bignumber.MulWadUp(executedAmountIn, p.Info.SwapFee)
		remainingAmountIn = new(big.Int).Sub(tokenAmountIn.Amount, new(big.Int).Add(executedAmountIn, feesTokenY))
	} else {
		feesTokenY = bignumber.MulWadUp(executedAmountOut, p.Info.SwapFee)
		remainingAmountIn = new(big.Int).Sub(tokenAmountIn.Amount, executedAmountIn)

		executedAmountOut.Sub(executedAmountOut, feesTokenY)
	}

	var updatedOrderBook *OrderBook
	if isBuy {
		updatedOrderBook = &OrderBook{
			Bids: p.OrderBook.Bids,
			Asks: *p.removeExecutedLevels(levels),
		}
	} else {
		updatedOrderBook = &OrderBook{
			Bids: *p.removeExecutedLevels(levels),
			Asks: p.OrderBook.Asks,
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: tokenOut, Amount: executedAmountOut},
		Fee:                    &pool.TokenAmount{Token: p.LobConfig.TokenY.Hex(), Amount: feesTokenY},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: remainingAmountIn},
		Gas:                    DefaultGas,
		SwapInfo: SwapInfo{
			UpdatedOrderBook: updatedOrderBook,
		},
	}, nil
}

func (p *PoolSimulator) removeExecutedLevels(levels *OrderBookLevels) *OrderBookLevels {
	for len(levels.ArrayShares) > 0 {
		if levels.ArrayShares[0].Cmp(bignumber.ZeroBI) == 0 {
			levels.ArrayShares = levels.ArrayShares[1:]
			levels.ArrayPrices = levels.ArrayPrices[1:]
		} else {
			break
		}
	}
	return levels
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	clonedOrderBook := OrderBook{
		Bids: OrderBookLevels{
			ArrayPrices: slices.Clone(p.OrderBook.Bids.ArrayPrices),
			ArrayShares: slices.Clone(p.OrderBook.Bids.ArrayShares),
		},
		Asks: OrderBookLevels{
			ArrayPrices: slices.Clone(p.OrderBook.Asks.ArrayPrices),
			ArrayShares: slices.Clone(p.OrderBook.Asks.ArrayShares),
		},
	}
	clonedLobConfig := *p.LobConfig

	return &PoolSimulator{
		Pool:      p.Pool,
		OrderBook: &clonedOrderBook,
		LobConfig: &clonedLobConfig,
	}
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for OnchainClob pool, wrong swapInfo type")
		return
	}
	p.OrderBook = si.UpdatedOrderBook
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, _ string) any {
	return nil
}
