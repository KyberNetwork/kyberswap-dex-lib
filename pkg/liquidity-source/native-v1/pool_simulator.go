package nativev1

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
	ErrAmountOutIsGreaterThanInventory        = errors.New("amountOut is greater than inventory")
	ErrInsufficientFundInTreasury             = errors.New("insufficient fund in treasury")
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

		BalanceTreasury0, BalanceTreasury1 *big.Int

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
		BalanceTreasury0     *big.Int     `json:"balanceTreasury0"`
		BalanceTreasury1     *big.Int     `json:"balanceTreasury1"`
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
		BalanceTreasury0:     extra.BalanceTreasury0,
		BalanceTreasury1:     extra.BalanceTreasury1,

		timestamp:      entityPool.Timestamp,
		priceTolerance: extra.PriceTolerance,
		expirySecs:     extra.ExpirySecs,
		gas:            defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Token == p.Token0.Address {
		return p.swap(params.TokenAmountIn.Amount, p.Token0, p.Token1,
			p.MinIn0, params.Limit.GetLimit(p.Token1.Address), p.BalanceTreasury1, p.ZeroToOnePriceLevels)
	} else {
		return p.swap(params.TokenAmountIn.Amount, p.Token1, p.Token0,
			p.MinIn1, params.Limit.GetLimit(p.Token0.Address), p.BalanceTreasury0, p.OneToZeroPriceLevels)
	}
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	amtIn, amtOut := params.TokenAmountIn.Amount, params.TokenAmountOut.Amount
	amtInF, _ := amtIn.Float64()
	if params.TokenAmountIn.Token == p.Token0.Address {
		amountInAfterDecimalsF := amtInF / math.Pow10(int(p.Token0.Decimals))
		p.ZeroToOnePriceLevels = getNewPriceLevelsState(amountInAfterDecimalsF, p.ZeroToOnePriceLevels)
		_, _, err := params.SwapLimit.UpdateLimit(p.Token1.Address, p.Token0.Address, amtOut, amtIn)
		if err != nil {
			logger.Errorf("unable to update native limit, error: %v", err)
		}

		p.BalanceTreasury0 = new(big.Int).Add(p.BalanceTreasury0, amtIn)
		p.BalanceTreasury1 = new(big.Int).Add(p.BalanceTreasury1, amtOut)
	} else {
		amountInAfterDecimalsF := amtInF / math.Pow10(int(p.Token1.Decimals))
		p.OneToZeroPriceLevels = getNewPriceLevelsState(amountInAfterDecimalsF, p.OneToZeroPriceLevels)
		_, _, err := params.SwapLimit.UpdateLimit(p.Token0.Address, p.Token1.Address, amtOut, amtIn)
		if err != nil {
			logger.Errorf("unable to update native limit, error: %v", err)
		}

		p.BalanceTreasury1 = new(big.Int).Add(p.BalanceTreasury1, amtIn)
		p.BalanceTreasury0 = new(big.Int).Add(p.BalanceTreasury0, amtOut)
	}
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	tokens, reserves := p.GetTokens(), p.GetReserves()
	nativeTreasury := make(map[string]*big.Int, len(tokens))
	for i, token := range tokens {
		nativeTreasury[token] = new(big.Int).Set(reserves[i])
	}
	return nativeTreasury
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return MetaInfo{Timestamp: p.timestamp}
}

func (p *PoolSimulator) swap(amountIn *big.Int, baseToken, quoteToken entity.PoolToken, minBase float64,
	inventoryLimit, balanceTreasuryInitial *big.Int, priceLevel []PriceLevel) (*pool.CalcAmountOutResult, error) {
	amountInF, _ := amountIn.Float64()
	amountInAfterDecimalsF := amountInF / math.Pow10(int(baseToken.Decimals))
	maxQuoteF, _ := inventoryLimit.Float64()
	maxQuoteAfterDecimalsF := maxQuoteF / math.Pow10(int(quoteToken.Decimals))
	amountOutAfterDecimalsF, err := getAmountOut(amountInAfterDecimalsF, minBase, maxQuoteAfterDecimalsF, priceLevel)
	if err != nil {
		return nil, err
	}
	amountOutF := amountOutAfterDecimalsF * math.Pow10(int(quoteToken.Decimals))
	amountOutF = amountOutF * (1 - float64(p.priceTolerance)/bps)
	amountOut, _ := big.NewFloat(amountOutF).Int(nil)

	if amountOut.Cmp(balanceTreasuryInitial) > 0 {
		return nil, ErrInsufficientFundInTreasury
	}

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

func getAmountOut(amtIn, minAmtIn, maxAmtOut float64, priceLevels []PriceLevel) (amountOut float64, err error) {
	if len(priceLevels) == 0 {
		return 0, ErrEmptyPriceLevels
	}

	if amtIn < minAmtIn {
		return 0, ErrAmountInIsLessThanLowestPriceLevel
	}

	for _, priceLevel := range priceLevels {
		if amtIn <= priceLevel.Quote {
			amountOut += amtIn * priceLevel.Price
			if amountOut > maxAmtOut {
				return 0, ErrAmountOutIsGreaterThanInventory
			}
			return amountOut, nil
		}
		amountOut += priceLevel.Quote * priceLevel.Price
		amtIn -= priceLevel.Quote
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
