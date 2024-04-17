package nativev1

import (
	"errors"
	"math"
	"math/big"
	"strings"

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
)

type (
	PoolSimulator struct {
		pool.Pool

		MarketMaker          string
		Token0               entity.PoolToken
		Token1               entity.PoolToken
		ZeroToOnePriceLevels []PriceLevel
		OneToZeroPriceLevels []PriceLevel
		MinIn0, MinIn1       float64

		timestamp      int64
		priceTolerance uint
		expirySecs     uint
		gas            Gas
	}

	StaticExtra struct {
		MarketMaker string `json:"marketMaker"`
	}

	Extra struct {
		ZeroToOnePriceLevels []PriceLevel `json:"0to1"`
		OneToZeroPriceLevels []PriceLevel `json:"1to0"`
		MinIn0               float64      `json:"min0"`
		MinIn1               float64      `json:"min1"`
		PriceTolerance       uint         `json:"tlrnce,omitempty"`
		ExpirySecs           uint         `json:"exp,omitempty"`
	}
	PriceLevel struct {
		Quote float64 `json:"q"`
		Price float64 `json:"p"`
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

	MetaInfo struct {
		Timestamp int64 `json:"timestamp"`
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
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
		Token0:               *entityPool.Tokens[0],
		Token1:               *entityPool.Tokens[1],
		ZeroToOnePriceLevels: extra.ZeroToOnePriceLevels,
		OneToZeroPriceLevels: extra.OneToZeroPriceLevels,
		MinIn0:               extra.MinIn0,
		MinIn1:               extra.MinIn1,

		timestamp:      entityPool.Timestamp,
		priceTolerance: extra.PriceTolerance,
		expirySecs:     extra.ExpirySecs,
		gas:            defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Token == p.Token0.Address {
		return p.swap(params.TokenAmountIn.Amount, p.Token0, p.Token1, p.MinIn0, p.ZeroToOnePriceLevels)
	} else {
		return p.swap(params.TokenAmountIn.Amount, p.Token1, p.Token0, p.MinIn1, p.OneToZeroPriceLevels)
	}
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	amountInF, _ := params.TokenAmountIn.Amount.Float64()
	if params.TokenAmountIn.Token == p.Token0.Address {
		amountInAfterDecimalsF := amountInF / math.Pow10(int(p.Token0.Decimals))
		p.ZeroToOnePriceLevels = getNewPriceLevelsState(amountInAfterDecimalsF, p.ZeroToOnePriceLevels)
	} else {
		amountInAfterDecimalsF := amountInF / math.Pow10(int(p.Token1.Decimals))
		p.OneToZeroPriceLevels = getNewPriceLevelsState(amountInAfterDecimalsF, p.OneToZeroPriceLevels)
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return MetaInfo{Timestamp: p.timestamp}
}

func (p *PoolSimulator) swap(amountIn *big.Int, baseToken, quoteToken entity.PoolToken, minBase float64,
	priceLevel []PriceLevel) (*pool.CalcAmountOutResult, error) {
	amountInF, _ := amountIn.Float64()
	amountInAfterDecimalsF := amountInF / math.Pow10(int(baseToken.Decimals))
	amountOutAfterDecimalsF, err := getAmountOut(amountInAfterDecimalsF, minBase, priceLevel)
	if err != nil {
		return nil, err
	}
	amountOutF := amountOutAfterDecimalsF * math.Pow10(int(quoteToken.Decimals))
	amountOutF = amountOutF * (1 - float64(p.priceTolerance)/bps)
	amountOut, _ := big.NewFloat(amountOutF).Int(nil)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: quoteToken.Address, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: baseToken.Address, Amount: bignumber.ZeroBI},
		Gas:            p.gas.Quote,
		SwapInfo: SwapInfo{
			BaseToken:        baseToken.Address,
			BaseTokenAmount:  amountIn.String(),
			QuoteToken:       quoteToken.Address,
			QuoteTokenAmount: amountOut.String(),
			MarketMaker:      p.MarketMaker,
			ExpirySecs:       p.expirySecs,
		},
	}, nil
}

func getAmountOut(amountIn, minAmount float64, priceLevels []PriceLevel) (amountOut float64, err error) {
	if len(priceLevels) == 0 {
		return 0, ErrEmptyPriceLevels
	}

	if amountIn < minAmount {
		return 0, ErrAmountInIsLessThanLowestPriceLevel
	}

	for _, priceLevel := range priceLevels {
		if amountIn <= priceLevel.Quote {
			return amountOut + amountIn*priceLevel.Price, nil
		}
		amountIn -= priceLevel.Quote
		amountOut += priceLevel.Quote * priceLevel.Price
	}
	return 0, ErrAmountInIsGreaterThanHighestPriceLevel
}

func getNewPriceLevelsState(amountIn float64, priceLevels []PriceLevel) []PriceLevel {
	if len(priceLevels) == 0 {
		return priceLevels
	}

	for i, priceLevel := range priceLevels {
		if amountIn < priceLevel.Quote {
			priceLevel.Quote -= amountIn
			priceLevels[i] = priceLevel
			return priceLevels[i:]
		}
		amountIn -= priceLevel.Quote
	}

	return nil
}
