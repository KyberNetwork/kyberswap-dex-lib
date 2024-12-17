package mxtrading

import (
	"errors"
	"math"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
)

var (
	ErrEmptyPriceLevels                    = errors.New("empty price levels")
	ErrAmountInIsLessThanLowestPriceLevel  = errors.New("amountIn is less than lowest price level")
	ErrAmountInIsGreaterThanTotalLevelSize = errors.New("amountIn is greater than total level size")
	ErrAmountOutIsGreaterThanInventory     = errors.New("amountOut is greater than inventory")
)

type (
	PoolSimulator struct {
		pool.Pool

		ZeroToOnePriceLevels []PriceLevel `json:"0to1"`
		OneToZeroPriceLevels []PriceLevel `json:"1to0"`

		token0, token1 entity.PoolToken

		timestamp int64
		gas       Gas
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens: lo.Map(entityPool.Tokens,
					func(item *entity.PoolToken, index int) string { return item.Address }),
				Reserves: lo.Map(entityPool.Reserves,
					func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			},
		},
		ZeroToOnePriceLevels: extra.ZeroToOnePriceLevels,
		OneToZeroPriceLevels: extra.OneToZeroPriceLevels,

		token0:    *entityPool.Tokens[0],
		token1:    *entityPool.Tokens[1],
		timestamp: entityPool.Timestamp,
		gas:       defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Token == p.token0.Address {
		return p.swap(params.TokenAmountIn.Amount, p.token0, p.token1,
			params.Limit.GetLimit(p.token1.Address), p.ZeroToOnePriceLevels,
		)
	} else {
		return p.swap(params.TokenAmountIn.Amount, p.token1, p.token0,
			params.Limit.GetLimit(p.token0.Address), p.OneToZeroPriceLevels,
		)
	}
}

func (p *PoolSimulator) swap(
	amountIn *big.Int,
	baseToken, quoteToken entity.PoolToken,
	inventoryLimit *big.Int,
	priceLevel []PriceLevel,
) (*pool.CalcAmountOutResult, error) {
	amountInF, _ := amountIn.Float64()
	amountInAfterDecimalsF := amountInF / math.Pow10(int(baseToken.Decimals))
	fillPrice, err := findFillPrice(amountInAfterDecimalsF, priceLevel)
	if err != nil {
		return nil, err
	}
	amountOutAfterDecimalsF := amountInAfterDecimalsF * fillPrice
	amountOutF := amountOutAfterDecimalsF * math.Pow10(int(quoteToken.Decimals))
	amountOut, _ := big.NewFloat(amountOutF).Int(nil)

	if amountOut.Cmp(inventoryLimit) > 0 {
		return nil, ErrAmountOutIsGreaterThanInventory
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: quoteToken.Address, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: baseToken.Address, Amount: bignumber.ZeroBI},
		Gas:            p.gas.FillOrderArgs,
		SwapInfo: SwapInfo{
			BaseToken:       baseToken.Address,
			BaseTokenAmount: amountIn.String(),
			QuoteToken:      quoteToken.Address,
		},
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenAmountOut.Token
	amountIn, amountOut := params.TokenAmountIn.Amount, params.TokenAmountOut.Amount
	amountInF, _ := amountIn.Float64()

	if tokenIn == p.token0.Address {
		amountInAfterDecimalsF := amountInF / math.Pow10(int(p.token0.Decimals))
		p.ZeroToOnePriceLevels = getNewPriceLevelsState(amountInAfterDecimalsF, p.ZeroToOnePriceLevels)
	} else {
		amountInAfterDecimalsF := amountInF / math.Pow10(int(p.token1.Decimals))
		p.OneToZeroPriceLevels = getNewPriceLevelsState(amountInAfterDecimalsF, p.OneToZeroPriceLevels)
	}

	if _, _, err := params.SwapLimit.UpdateLimit(tokenOut, tokenIn, amountOut, amountIn); err != nil {
		logger.Errorf("unable to update mx-trading limit, error: %v", err)
	}
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	tokens, reserves := p.GetTokens(), p.GetReserves()
	inventory := make(map[string]*big.Int, len(tokens))
	for i, token := range tokens {
		var reducedReserve big.Float
		reducedReserve.SetInt(reserves[i]).Mul(&reducedReserve, big.NewFloat(0.95))
		// Reduce each token's reserve by 5% to prevent the total quote amount of a token
		// from potentially exceeding the maker's balance when building the route
		inventory[token], _ = reducedReserve.Int(nil)
	}

	return inventory
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return MetaInfo{Timestamp: p.timestamp}
}

func findFillPrice(amountInF float64, levels []PriceLevel) (float64, error) {
	if len(levels) == 0 {
		return 0, ErrEmptyPriceLevels
	}

	if amountInF < levels[0].Size {
		return 0, ErrAmountInIsLessThanLowestPriceLevel
	}

	var sizeFilled, price float64
	for _, level := range levels {
		partFillSize := amountInF - sizeFilled
		if level.Size >= partFillSize {
			price += (level.Price * partFillSize) / amountInF
			sizeFilled += partFillSize
			break
		}

		price += level.Price * level.Size / amountInF
		sizeFilled += level.Size
	}

	if sizeFilled == amountInF {
		return price, nil
	}

	return 0, ErrAmountInIsGreaterThanTotalLevelSize
}

func getNewPriceLevelsState(amountIn float64, priceLevels []PriceLevel) []PriceLevel {
	for i, priceLevel := range priceLevels {
		if amountIn < priceLevel.Size {
			priceLevel.Size -= amountIn
			priceLevels[i] = priceLevel
			return priceLevels[i:]
		}
		amountIn -= priceLevel.Size
	}

	return nil
}
