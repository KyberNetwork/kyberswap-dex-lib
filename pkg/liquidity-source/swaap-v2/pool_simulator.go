package swaapv2

import (
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/float"
	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/unit"
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
		baseToken              *entity.PoolToken
		quoteToken             *entity.PoolToken
		baseToQuotePriceLevels []PriceLevel
		quoteToBasePriceLevels []PriceLevel
		timestamp              int64
		priceTolerance         *big.Int
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
		isBaseSwapped:          false,
		isQuoteSwapped:         false,
		baseToken:              entityPool.Tokens[0],
		quoteToken:             entityPool.Tokens[1],
		baseToQuotePriceLevels: extra.BaseToQuotePriceLevels,
		quoteToBasePriceLevels: extra.QuoteToBasePriceLevels,
		timestamp:              entityPool.Timestamp,
		priceTolerance:         big.NewInt(int64(extra.PriceTolerance)),
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
	if params.TokenAmountIn.Token == p.baseToken.Address {
		p.isBaseSwapped = true
		return
	}
	p.isQuoteSwapped = true
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

	amountInAfterDecimals := unit.ToDecimal(amountIn, p.baseToken.Decimals)

	amountOutAfterDecimals, err := getAmountOut(amountInAfterDecimals, p.baseToQuotePriceLevels)
	if err != nil {
		return nil, err
	}

	amountOut, _ := new(big.Float).Mul(
		amountOutAfterDecimals,
		bignumber.TenPowDecimals(p.quoteToken.Decimals),
	).Int(nil)

	amountOut = new(big.Int).Quo(
		new(big.Int).Mul(
			amountOut,
			new(big.Int).Sub(priceToleranceBps, p.priceTolerance),
		),
		priceToleranceBps,
	)

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

	amountInAfterDecimals := unit.ToDecimal(amountIn, p.quoteToken.Decimals)

	amountOutAfterDecimals, err := getAmountOut(amountInAfterDecimals, p.quoteToBasePriceLevels)
	if err != nil {
		return nil, err
	}

	amountOut, _ := new(big.Float).Mul(
		amountOutAfterDecimals,
		bignumber.TenPowDecimals(p.baseToken.Decimals),
	).Int(nil)

	amountOut = new(big.Int).Quo(
		new(big.Int).Mul(
			amountOut,
			new(big.Int).Sub(priceToleranceBps, p.priceTolerance),
		),
		priceToleranceBps,
	)

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

func getAmountOut(amountIn *big.Float, priceLevels []PriceLevel) (*big.Float, error) {
	if len(priceLevels) == 0 {
		return nil, ErrEmptyPriceLevels
	}

	var (
		amountOut    = float.Zero()
		amountInLeft = new(big.Float).Set(amountIn)
		levelIdx     = 0
	)

	for {
		availableAmount := new(big.Float).SetFloat64(priceLevels[levelIdx].Level)
		if levelIdx > 0 {
			availableAmount.Sub(availableAmount, new(big.Float).SetFloat64(priceLevels[levelIdx-1].Level))
		}
		var swappableAmount *big.Float
		if availableAmount.Cmp(amountInLeft) > 0 {
			swappableAmount = amountInLeft
		} else {
			swappableAmount = availableAmount
		}

		amountOut = new(big.Float).Add(
			amountOut,
			new(big.Float).Mul(
				swappableAmount, new(big.Float).SetFloat64(priceLevels[levelIdx].Price),
			),
		)

		amountInLeft = new(big.Float).Sub(amountInLeft, swappableAmount)
		levelIdx += 1

		if amountInLeft.Cmp(float.Zero()) == 0 || levelIdx >= len(priceLevels) {
			break
		}
	}

	if amountInLeft.Cmp(float.Zero()) != 0 {
		return nil, ErrOutOfLiquidity
	}

	return amountOut, nil
}
