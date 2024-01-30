package swaapv2

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/float"
	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/unit"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrEmptyPriceLevels      = errors.New("empty price levels")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)

type (
	PoolSimulator struct {
		pool.Pool

		baseToken              *entity.PoolToken
		quoteToken             *entity.PoolToken
		baseToQuotePriceLevels []PriceLevel
		quoteToBasePriceLevels []PriceLevel

		timestamp int64

		gas Gas
	}

	MetaInfo struct {
		Timestamp int64 `json:"timestamp"`
	}

	PriceLevel struct {
		Price  float64 `json:"price"`
		Amount float64 `json:"amount"`
	}

	Gas struct {
		Swap int64
	}

	PoolExtra struct {
		BaseToQuotePriceLevels []PriceLevel `json:"baseToQuotePriceLevels"`
		QuoteToBasePriceLevels []PriceLevel `json:"quoteToBasePriceLevels"`
	}

	SwapInfo struct {
		TokenIn  string
		TokenOut string
		AmountIn string
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
				Tokens:     lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
				Reserves:   lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			},
		},
		baseToken:              entityPool.Tokens[0],
		quoteToken:             entityPool.Tokens[1],
		baseToQuotePriceLevels: extra.BaseToQuotePriceLevels,
		quoteToBasePriceLevels: extra.QuoteToBasePriceLevels,

		timestamp: entityPool.Timestamp,

		gas: DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Token == p.baseToken.Address {
		return p.swapBaseToQuote(params.TokenAmountIn.Amount)
	} else {
		return p.swapQuoteToBase(params.TokenAmountIn.Amount)
	}
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if params.TokenAmountIn.Token == p.baseToken.Address {
		amountInAfterDecimals := unit.ToDecimal(params.TokenAmountIn.Amount, p.baseToken.Decimals)

		p.baseToQuotePriceLevels = getNewPriceLevelsState(amountInAfterDecimals, p.baseToQuotePriceLevels)
	} else {
		amountInAfterDecimals := unit.ToDecimal(params.TokenAmountIn.Amount, p.quoteToken.Decimals)

		p.quoteToBasePriceLevels = getNewPriceLevelsState(amountInAfterDecimals, p.quoteToBasePriceLevels)
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return MetaInfo{
		Timestamp: p.timestamp,
	}
}

func (p *PoolSimulator) swapBaseToQuote(amountIn *big.Int) (*pool.CalcAmountOutResult, error) {
	amountInAfterDecimals := unit.ToDecimal(amountIn, p.baseToken.Decimals)

	amountOutAfterDecimals, err := getAmountOut(amountInAfterDecimals, p.baseToQuotePriceLevels)
	if err != nil {
		return nil, err
	}

	amountOut, _ := new(big.Float).Mul(
		amountOutAfterDecimals,
		bignumber.TenPowDecimals(p.quoteToken.Decimals),
	).Int(nil)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: p.baseToken.Address, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: p.quoteToken.Address, Amount: integer.Zero()},
		Gas:            p.gas.Swap,
	}, nil
}

func (p *PoolSimulator) swapQuoteToBase(amountIn *big.Int) (*pool.CalcAmountOutResult, error) {
	amountInAfterDecimals := unit.ToDecimal(amountIn, p.quoteToken.Decimals)

	amountOutAfterDecimals, err := getAmountOut(amountInAfterDecimals, p.quoteToBasePriceLevels)
	if err != nil {
		return nil, err
	}

	amountOut, _ := new(big.Float).Mul(
		amountOutAfterDecimals,
		bignumber.TenPowDecimals(p.baseToken.Decimals),
	).Int(nil)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: p.quoteToken.Address, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: p.baseToken.Address, Amount: integer.Zero()},
		Gas:            p.gas.Swap,
	}, nil
}

func getAmountOut(amountIn *big.Float, priceLevels []PriceLevel) (*big.Float, error) {
	if len(priceLevels) == 0 {
		return nil, ErrEmptyPriceLevels
	}

	// Calculate the total available amount in the price levels
	availableAmount := lo.Reduce(priceLevels, func(acc float64, priceLevel PriceLevel, _ int) float64 {
		return acc + priceLevel.Amount
	}, 0.0)

	availableAmountBF := new(big.Float).SetFloat64(availableAmount)

	// If the amount in is greater than the available amount that price levels can provide, return error insufficient liquidity
	if amountIn.Cmp(availableAmountBF) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountOut := float.Zero()
	amountInLeft := amountIn
	currentLevelIdx := 0

	for {
		currentLevelAvailableAmount := new(big.Float).SetFloat64(priceLevels[currentLevelIdx].Amount)
		swappableAmount := currentLevelAvailableAmount

		if currentLevelAvailableAmount.Cmp(amountInLeft) > 0 {
			swappableAmount = amountInLeft
		}

		amountOut = new(big.Float).Add(
			amountOut,
			new(big.Float).Mul(
				swappableAmount, new(big.Float).SetFloat64(priceLevels[currentLevelIdx].Price),
			),
		)

		amountInLeft = new(big.Float).Sub(amountInLeft, swappableAmount)
		currentLevelIdx += 1

		if amountInLeft.Cmp(float.Zero()) == 0 || currentLevelIdx > len(priceLevels)-1 {
			break
		}
	}

	return amountOut, nil
}

func getNewPriceLevelsState(
	amountIn *big.Float,
	priceLevels []PriceLevel,
) []PriceLevel {
	if len(priceLevels) == 0 {
		return priceLevels
	}

	amountInLeft := amountIn
	currentLevelIdx := 0

	for {
		currentLevelAvailableAmount := new(big.Float).SetFloat64(priceLevels[currentLevelIdx].Amount)
		swappableAmount := currentLevelAvailableAmount

		if currentLevelAvailableAmount.Cmp(amountInLeft) > 0 {
			// Update the price level at the current step because it's partially filled
			priceLevels[currentLevelIdx].Amount, _ = new(big.Float).Sub(currentLevelAvailableAmount, amountInLeft).Float64()

			swappableAmount = amountInLeft
		} else {
			// Only increase the step if the current level is fully filled
			currentLevelIdx += 1
		}

		amountInLeft = new(big.Float).Sub(amountInLeft, swappableAmount)

		if amountInLeft.Cmp(float.Zero()) == 0 || currentLevelIdx > len(priceLevels)-1 {
			// Get the remaining price levels
			priceLevels = priceLevels[currentLevelIdx:]

			break
		}
	}

	return priceLevels
}
