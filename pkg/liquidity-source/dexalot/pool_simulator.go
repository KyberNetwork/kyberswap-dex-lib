package dexalot

import (
	"errors"
	"math"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrEmptyPriceLevels                       = errors.New("empty price levels")
	ErrAmountInIsLessThanLowestPriceLevel     = errors.New("amountIn is less than lowest price level")
	ErrAmountInIsGreaterThanHighestPriceLevel = errors.New("amountIn is greater than highest price level")
	ErrNoSwapLimit                            = errors.New("swap limit is required for dexalot pools")
)

type (
	PoolSimulator struct {
		pool.Pool
		Token0               entity.PoolToken
		Token1               entity.PoolToken
		ZeroToOnePriceLevels []PriceLevel
		OneToZeroPriceLevels []PriceLevel
		gas                  Gas
		Token0Original       string
		Token1Original       string
	}
	SwapInfo struct {
		BaseToken          string `json:"b" mapstructure:"b"`
		BaseTokenAmount    string `json:"bAmt" mapstructure:"bAmt"`
		QuoteToken         string `json:"q" mapstructure:"q"`
		QuoteTokenAmount   string `json:"qAmt" mapstructure:"qAmt"`
		MarketMaker        string `json:"mm,omitempty" mapstructure:"mm"`
		ExpirySecs         uint   `json:"exp,omitempty" mapstructure:"exp"`
		BaseTokenOriginal  string `json:"bo,omitempty" mapstructure:"bo"`
		QuoteTokenOriginal string `json:"qo,omitempty" mapstructure:"qo"`
		BaseTokenReserve   string `json:"br,omitempty" mapstructure:"br"`
		QuoteTokenReserve  string `json:"qr,omitempty" mapstructure:"qr"`
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
		Token0Address        string          `json:"token0"`
		Token1Address        string          `json:"token1"`
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
		Token0Original:       extra.Token0Address,
		Token1Original:       extra.Token1Address,
		ZeroToOnePriceLevels: zeroToOnePriceLevels,
		OneToZeroPriceLevels: oneToZeroPriceLevels,
		gas:                  defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.Limit == nil {
		return nil, ErrNoSwapLimit
	}

	tokenIn, tokenOut, tokenInOriginal, tokenOutOriginal, levels := p.Token0, p.Token1, p.Token0Original, p.Token1Original, p.ZeroToOnePriceLevels
	if params.TokenAmountIn.Token == p.Info.Tokens[1] {
		tokenIn, tokenOut, tokenInOriginal, tokenOutOriginal, levels = p.Token1, p.Token0, p.Token1Original, p.Token0Original, p.OneToZeroPriceLevels
	}
	result, _, err := p.swap(params.TokenAmountIn.Amount, tokenIn, tokenOut, tokenInOriginal, tokenOutOriginal, levels)
	if err != nil {
		return nil, err
	}

	inventoryLimit := params.Limit.GetLimit(tokenOut.Address)
	if result.TokenAmountOut.Amount.Cmp(inventoryLimit) > 0 {
		return nil, errors.New("not enough inventory")
	}
	return result, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenIn, tokenOut := p.Token0, p.Token1
	if params.TokenAmountIn.Token == p.Token1.Address {
		tokenIn, tokenOut = p.Token1, p.Token0
	}
	_, _, err := params.SwapLimit.UpdateLimit(tokenOut.Address, tokenIn.Address,
		params.TokenAmountOut.Amount, params.TokenAmountIn.Amount)
	if err != nil {
		logger.Errorf("unable to update dexalot limit, error: %v", err)
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) swap(amountIn *big.Int, baseToken, quoteToken entity.PoolToken,
	baseOriginal, quoteOriginal string, priceLevel []PriceLevel) (*pool.CalcAmountOutResult, string, error) {

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
	var baseTokenReserve, quoteTokenReserve *big.Int
	if strings.EqualFold(baseToken.Address, p.Info.Tokens[0]) {
		baseTokenReserve = p.Info.Reserves[0]
		quoteTokenReserve = p.Info.Reserves[1]
	} else {
		baseTokenReserve = p.Info.Reserves[1]
		quoteTokenReserve = p.Info.Reserves[0]
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: quoteToken.Address, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: baseToken.Address, Amount: bignumber.ZeroBI},
		Gas:            p.gas.Quote,
		SwapInfo: SwapInfo{
			BaseToken:          baseToken.Address,
			BaseTokenAmount:    amountIn.String(),
			QuoteToken:         quoteToken.Address,
			QuoteTokenAmount:   amountOut.String(),
			BaseTokenOriginal:  baseOriginal,
			QuoteTokenOriginal: quoteOriginal,
			BaseTokenReserve:   baseTokenReserve.String(),
			QuoteTokenReserve:  quoteTokenReserve.String(),
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

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	var pmmInventory = make(map[string]*big.Int, len(p.GetTokens()))
	tokens := p.GetTokens()
	rsv := p.GetReserves()
	for i, tok := range tokens {
		pmmInventory[tok] = big.NewInt(0).Set(rsv[i]) //clone here.
	}
	return pmmInventory
}
