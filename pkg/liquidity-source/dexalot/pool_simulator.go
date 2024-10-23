package dexalot

import (
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/samber/lo"
)

var (
	ErrEmptyPriceLevels                       = errors.New("empty price levels")
	ErrAmountInIsLessThanLowestPriceLevel     = errors.New("amountIn is less than lowest price level")
	ErrAmountInIsGreaterThanHighestPriceLevel = errors.New("amountIn is greater than highest price level")
)

type (
	PoolSimulator struct {
		pool.Pool
		Token0               entity.PoolToken
		Token1               entity.PoolToken
		ZeroToOnePriceLevels []PriceLevel
		OneToZeroPriceLevels []PriceLevel
		gas                  Gas
	}
	SwapInfo struct {
		BaseToken        string `json:"b" mapstructure:"b"`
		BaseTokenAmount  string `json:"bAmt" mapstructure:"bAmt"`
		QuoteToken       string `json:"q" mapstructure:"q"`
		QuoteTokenAmount string `json:"qAmt" mapstructure:"qAmt"`
		MarketMaker      string `json:"mm,omitempty" mapstructure:"mm"`
		ExpirySecs       uint   `json:"exp,omitempty" mapstructure:"exp"`
	}

	Gas struct {
		Quote int64
	}

	PriceLevel struct {
		Quote *big.Float
		Price *big.Float
	}

	PriceLevelRaw struct {
		Price float64 `json:"p"`
		Quote float64 `json:"q"`
	}

	Extra struct {
		ZeroToOnePriceLevels []PriceLevelRaw `json:"0to1"`
		OneToZeroPriceLevels []PriceLevelRaw `json:"1to0"`
	}

	MetaInfo struct {
		Timestamp int64 `json:"timestamp"`
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	zeroToOnePriceLevels := lo.Map(extra.ZeroToOnePriceLevels, func(item PriceLevelRaw, index int) PriceLevel {
		return PriceLevel{
			Quote: big.NewFloat(item.Quote),
			Price: big.NewFloat(item.Price),
		}
	})
	oneToZeroPriceLevels := lo.Map(extra.OneToZeroPriceLevels, func(item PriceLevelRaw, index int) PriceLevel {
		return PriceLevel{
			Quote: big.NewFloat(item.Quote),
			Price: big.NewFloat(item.Price),
		}
	})

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
		Token0:               *entityPool.Tokens[0],
		Token1:               *entityPool.Tokens[1],
		ZeroToOnePriceLevels: zeroToOnePriceLevels,
		OneToZeroPriceLevels: oneToZeroPriceLevels,
		gas:                  defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut, levels := p.Token0, p.Token1, p.ZeroToOnePriceLevels
	if params.TokenAmountIn.Token == p.Info.Tokens[1] {
		tokenIn, tokenOut, levels = p.Token1, p.Token0, p.OneToZeroPriceLevels
	}
	amountOut, _, err := p.swap(params.TokenAmountIn.Amount, tokenIn, tokenOut, levels)
	return amountOut, err
}
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	// Ignore for now cause logic not exposed
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) swap(amountIn *big.Int, baseToken, quoteToken entity.PoolToken,
	priceLevel []PriceLevel) (*pool.CalcAmountOutResult, string, error) {

	var amountInAfterDecimals, decimalsPow, amountInBF, amountOutBF big.Float

	amountInBF.SetInt(amountIn)
	decimalsPow.SetFloat64(math.Pow10(int(baseToken.Decimals)))
	amountInAfterDecimals.Quo(&amountInBF, &decimalsPow)
	var amountOutAfterDecimals big.Float
	err := getAmountOut(&amountInAfterDecimals, priceLevel, &amountOutAfterDecimals)
	if err != nil {
		return nil, "", err
	}
	decimalsPow.SetFloat64(math.Pow10(int(quoteToken.Decimals)))
	amountOutBF.Mul(&amountOutAfterDecimals, &decimalsPow)

	amountOut, _ := amountOutBF.Int(nil)
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: quoteToken.Address, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: baseToken.Address, Amount: bignumber.ZeroBI},
		Gas:            p.gas.Quote,
		SwapInfo: SwapInfo{
			BaseToken:        baseToken.Address,
			BaseTokenAmount:  amountIn.String(),
			QuoteToken:       quoteToken.Address,
			QuoteTokenAmount: amountOut.String(),
		},
	}, amountOutAfterDecimals.String(), nil
}

func getAmountOut(amountIn *big.Float, priceLevels []PriceLevel, amountOut *big.Float) error {
	if len(priceLevels) == 0 {
		return ErrEmptyPriceLevels
	}
	// Check lower bound
	if amountIn.Cmp(priceLevels[0].Quote) < 0 {
		return ErrAmountInIsLessThanLowestPriceLevel
	}

	if amountIn.Cmp(priceLevels[len(priceLevels)-1].Quote) > 0 {
		return ErrAmountInIsGreaterThanHighestPriceLevel
	}
	left := 0
	right := len(priceLevels)
	var qty *big.Float

	for left < right {
		mid := (left + right) / 2
		qty = priceLevels[mid].Quote
		if qty.Cmp(amountIn) <= 0 {
			left = mid + 1
		} else {
			right = mid
		}
	}

	var price *big.Float
	if amountIn.Cmp(qty) == 0 {
		price = priceLevels[left-1].Price // TODO: check with https://docs.dexalot.com/apiv2/SimpleSwap.html#_3b-request-batched-quotes-optional
	} else if left == 0 {
		price = big.NewFloat(0)
	} else if left < len(priceLevels) {
		price = priceLevels[left-1].Price.Add(
			priceLevels[left-1].Price,
			new(big.Float).Quo(
				new(big.Float).Mul(
					new(big.Float).Sub(priceLevels[left].Price, priceLevels[left-1].Price),
					new(big.Float).Sub(amountIn, priceLevels[left-1].Quote),
				),
				new(big.Float).Sub(priceLevels[left].Quote, priceLevels[left-1].Quote),
			),
		)
	}
	amountOut.Mul(amountIn, price)
	return nil
}
