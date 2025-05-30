package swaapv2

import (
	"errors"
	"math"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrEmptyPriceLevels      = errors.New("empty price levels")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrPoolSwapped           = errors.New("pool swapped")
	ErrOutOfLiquidity        = errors.New("out of liquidity")
)

type (
	PoolSimulator struct {
		pool.Pool
		isBaseSwapped          bool
		isQuoteSwapped         bool
		baseToken              entity.PoolToken
		quoteToken             entity.PoolToken
		baseToQuotePriceLevels []PriceLevel
		quoteToBasePriceLevels []PriceLevel
		timestamp              int64
		priceTolerance         float64
		gas                    Gas
	}

	MetaInfo struct {
		Timestamp int64 `json:"timestamp"`
	}

	PriceLevel struct {
		Price float64 `json:"price"`
		Level float64 `json:"level"`
	}

	Gas struct {
		Swap int64
	}

	PoolExtra struct {
		BaseToQuotePriceLevels []PriceLevel `json:"baseToQuotePriceLevels"`
		QuoteToBasePriceLevels []PriceLevel `json:"quoteToBasePriceLevels"`
		PriceTolerance         uint         `json:"priceTolerance"`
	}

	SwapInfo struct {
		TokenIn  string `json:"tokenIn"`
		TokenOut string `json:"tokenOut"`
		AmountIn string `json:"amountIn"`
	}
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra PoolExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  strings.ToLower(entityPool.Address),
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
				Reserves: lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			},
		},
		isBaseSwapped:          false,
		isQuoteSwapped:         false,
		baseToken:              *entityPool.Tokens[0],
		quoteToken:             *entityPool.Tokens[1],
		baseToQuotePriceLevels: extra.BaseToQuotePriceLevels,
		quoteToBasePriceLevels: extra.QuoteToBasePriceLevels,
		timestamp:              entityPool.Timestamp,
		priceTolerance:         float64(extra.PriceTolerance),
		gas:                    DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Token == p.baseToken.Address {
		return p.swapBaseToQuote(params.TokenAmountIn.Amount)
	}

	return p.swapQuoteToBase(params.TokenAmountIn.Amount)
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	// if params.TokenAmountIn.Token == p.baseToken.Address {
	// 	p.isBaseSwapped = true
	// 	return
	// }
	// p.isQuoteSwapped = true
	amtIn, _ := params.TokenAmountIn.Amount, params.TokenAmountOut.Amount
	amtInF, _ := amtIn.Float64()
	if params.TokenAmountIn.Token == p.baseToken.Address {
		amountInAfterDecimalsF := amtInF / math.Pow10(int(p.baseToken.Decimals))
		p.baseToQuotePriceLevels = getNewPriceLevelsState(amountInAfterDecimalsF, p.baseToQuotePriceLevels)
	} else {
		amountInAfterDecimalsF := amtInF / math.Pow10(int(p.quoteToken.Decimals))
		p.quoteToBasePriceLevels = getNewPriceLevelsState(amountInAfterDecimalsF, p.quoteToBasePriceLevels)
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return MetaInfo{
		Timestamp: p.timestamp,
	}
}

func (p *PoolSimulator) swapBaseToQuote(amountIn *big.Int) (*pool.CalcAmountOutResult, error) {
	if p.isBaseSwapped {
		return nil, ErrPoolSwapped
	}

	amountInFl, _ := amountIn.Float64()
	amountInAfterDecimals := amountInFl / math.Pow10(int(p.baseToken.Decimals))

	amountOutAfterDecimals, err := getAmountOut(amountInAfterDecimals, p.baseToQuotePriceLevels)
	if err != nil {
		return nil, err
	}

	amountOutFl := amountOutAfterDecimals * math.Pow10(int(p.quoteToken.Decimals))
	amountOutFl = amountOutFl * (priceToleranceBps - p.priceTolerance) / priceToleranceBps

	amountOut, _ := new(big.Float).SetFloat64(amountOutFl).Int(nil)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: p.quoteToken.Address, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: p.quoteToken.Address, Amount: integer.Zero()},
		Gas:            p.gas.Swap,
		SwapInfo: SwapInfo{
			TokenIn:  p.baseToken.Address,
			TokenOut: p.quoteToken.Address,
			AmountIn: amountIn.String(),
		},
	}, nil
}

func (p *PoolSimulator) swapQuoteToBase(amountIn *big.Int) (*pool.CalcAmountOutResult, error) {
	if p.isQuoteSwapped {
		return nil, ErrPoolSwapped
	}

	amountInFl, _ := amountIn.Float64()
	amountInAfterDecimals := amountInFl / math.Pow10(int(p.quoteToken.Decimals))

	amountOutAfterDecimals, err := getAmountOut(amountInAfterDecimals, p.quoteToBasePriceLevels)
	if err != nil {
		return nil, err
	}

	amountOutFl := amountOutAfterDecimals * math.Pow10(int(p.baseToken.Decimals))
	amountOutFl = amountOutFl * (priceToleranceBps - p.priceTolerance) / priceToleranceBps

	amountOut, _ := new(big.Float).SetFloat64(amountOutFl).Int(nil)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: p.baseToken.Address, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: p.baseToken.Address, Amount: integer.Zero()},
		Gas:            p.gas.Swap,
		SwapInfo: SwapInfo{
			TokenIn:  p.quoteToken.Address,
			TokenOut: p.baseToken.Address,
			AmountIn: amountIn.String(),
		},
	}, nil
}

func getAmountOut(amountIn float64, priceLevels []PriceLevel) (float64, error) {
	if len(priceLevels) == 0 {
		return 0, ErrEmptyPriceLevels
	}

	if amountIn > priceLevels[len(priceLevels)-1].Level {
		return 0, ErrOutOfLiquidity
	}

	var (
		amountOut    = float64(0)
		amountInLeft = amountIn
		levelIdx     = 0
	)

	for {
		availableAmount := priceLevels[levelIdx].Level
		if levelIdx > 0 {
			availableAmount -= priceLevels[levelIdx-1].Level
		}
		swappableAmount := math.Min(availableAmount, amountInLeft)
		amountOut += swappableAmount * priceLevels[levelIdx].Price
		amountInLeft -= swappableAmount
		levelIdx += 1
		if amountInLeft == 0 || levelIdx >= len(priceLevels) {
			break
		}
	}

	return amountOut, nil
}

func getNewPriceLevelsState(amountIn float64, priceLevels []PriceLevel) []PriceLevel {
	if len(priceLevels) == 0 {
		return priceLevels
	}

	var newPriceLevels []PriceLevel

	// Reduce every price level by amountIn
	for idx, priceLevel := range priceLevels {
		if priceLevel.Level <= amountIn {
			continue
		}

		if newPriceLevels == nil {
			newPriceLevels = make([]PriceLevel, 0, len(priceLevels)-idx)
		}

		newPriceLevels = append(newPriceLevels, PriceLevel{
			Price: priceLevel.Price,
			Level: priceLevel.Level - amountIn,
		})
	}

	return newPriceLevels
}
