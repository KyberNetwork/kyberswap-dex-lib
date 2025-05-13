package hashflowv3

import (
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	MarketMaker            string
	Token0, Token1         entity.PoolToken
	ZeroToOnePriceLevels   []PriceLevel
	OneToZeroPriceLevels   []PriceLevel
	MinAmt0In, MinAmt1In   float64
	MinAmt0Out, MinAmt1Out float64

	timestamp      int64
	priceTolerance float64
	gas            Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	zeroToOnePriceLevels, err := parsePriceLevelRaw(extra.ZeroToOnePriceLevels)
	if err != nil {
		return nil, err
	}
	oneToZeroPriceLevels, err := parsePriceLevelRaw(extra.OneToZeroPriceLevels)
	if err != nil {
		return nil, err
	}

	var minAmt0In, minAmt1In, minAmt0Out, minAmt1Out float64
	if len(zeroToOnePriceLevels) > 0 {
		minAmt0In = zeroToOnePriceLevels[0].Quote
		minAmt1Out = zeroToOnePriceLevels[0].Quote * zeroToOnePriceLevels[0].Price
	}
	if len(oneToZeroPriceLevels) > 0 {
		minAmt1In = oneToZeroPriceLevels[0].Quote
		minAmt0Out = oneToZeroPriceLevels[0].Quote * oneToZeroPriceLevels[0].Price
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
		MarketMaker:          staticExtra.MarketMaker,
		Token0:               *entityPool.Tokens[0],
		Token1:               *entityPool.Tokens[1],
		ZeroToOnePriceLevels: zeroToOnePriceLevels,
		OneToZeroPriceLevels: oneToZeroPriceLevels,
		MinAmt0In:            minAmt0In,
		MinAmt1In:            minAmt1In,
		MinAmt1Out:           minAmt1Out,
		MinAmt0Out:           minAmt0Out,

		timestamp:      entityPool.Timestamp,
		priceTolerance: float64(extra.PriceTolerance) / Bps,
		gas:            defaultGas,
	}, nil
}

func parsePriceLevelRaw(rawLevels []PriceLevelRaw) ([]PriceLevel, error) {
	priceLevels := make([]PriceLevel, len(rawLevels))
	for i, rawLevel := range rawLevels {
		quote, err := strconv.ParseFloat(rawLevel.Quote, 64)
		if err != nil {
			return nil, err
		}
		price, err := strconv.ParseFloat(rawLevel.Price, 64)
		if err != nil {
			return nil, err
		}

		priceLevels[i] = PriceLevel{
			Quote: quote,
			Price: price,
		}
	}
	return priceLevels, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenOut == p.Token1.Address {
		return p.swap(params.TokenAmountIn.Amount, p.MinAmt0In, p.Token0, p.Token1, p.ZeroToOnePriceLevels)
	} else {
		return p.swap(params.TokenAmountIn.Amount, p.MinAmt1In, p.Token1, p.Token0, p.OneToZeroPriceLevels)
	}
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if params.TokenIn == p.Token0.Address {
		return p.swapExactOut(params.TokenAmountOut.Amount, p.MinAmt1Out, p.Token0, p.Token1, p.ZeroToOnePriceLevels)
	} else {
		return p.swapExactOut(params.TokenAmountOut.Amount, p.MinAmt0Out, p.Token1, p.Token0, p.OneToZeroPriceLevels)
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.ZeroToOnePriceLevels = slices.Clone(p.ZeroToOnePriceLevels)
	cloned.OneToZeroPriceLevels = slices.Clone(p.OneToZeroPriceLevels)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	amtIn, _ := params.TokenAmountIn.Amount.Float64()
	if params.TokenAmountIn.Token == p.Token0.Address {
		amtIn /= math.Pow10(int(p.Token0.Decimals))
		p.ZeroToOnePriceLevels = getNewPriceLevelsState(amtIn, p.ZeroToOnePriceLevels)
	} else {
		amtIn /= math.Pow10(int(p.Token1.Decimals))
		p.OneToZeroPriceLevels = getNewPriceLevelsState(amtIn, p.OneToZeroPriceLevels)
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return MetaInfo{Timestamp: p.timestamp}
}

func (p *PoolSimulator) swap(amountIn *big.Int, minAmtIn float64, baseToken, quoteToken entity.PoolToken,
	priceLevel []PriceLevel) (*pool.CalcAmountOutResult, error) {
	amtIn, _ := amountIn.Float64()
	amtIn /= math.Pow10(int(baseToken.Decimals))

	amtOut, err := getAmountOut(amtIn, minAmtIn, priceLevel)
	if err != nil {
		return nil, err
	}

	amtOut *= math.Pow10(int(quoteToken.Decimals))
	amtOut -= amtOut * p.priceTolerance

	amountOut, _ := big.NewFloat(amtOut).Int(nil)

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
		},
	}, nil
}

func (p *PoolSimulator) swapExactOut(amountOut *big.Int, minAmtOut float64, baseToken, quoteToken entity.PoolToken,
	priceLevel []PriceLevel) (*pool.CalcAmountInResult, error) {
	amtOut, _ := amountOut.Float64()
	amtOut /= math.Pow10(int(quoteToken.Decimals))

	amtIn, err := getAmountIn(amtOut, minAmtOut, priceLevel)
	if err != nil {
		return nil, err
	}

	amtIn *= math.Pow10(int(baseToken.Decimals))
	amtIn += amtIn * p.priceTolerance

	amountIn, _ := big.NewFloat(math.Floor(amtIn)).Int(nil)

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: baseToken.Address, Amount: amountIn},
		Fee:           &pool.TokenAmount{Token: baseToken.Address, Amount: bignumber.ZeroBI},
		Gas:           p.gas.Quote,
		SwapInfo: SwapInfo{
			BaseToken:        baseToken.Address,
			BaseTokenAmount:  amountIn.String(),
			QuoteToken:       quoteToken.Address,
			QuoteTokenAmount: amountOut.String(),
			MarketMaker:      p.MarketMaker,
		},
	}, nil
}

func getAmountOut(amtIn, minAmtIn float64, priceLevels []PriceLevel) (amtOut float64, err error) {
	if len(priceLevels) == 0 {
		return 0, ErrEmptyPriceLevels
	} else if amtIn < minAmtIn {
		return 0, ErrAmtInLessThanMinAllowed
	} else if amtIn > lo.SumBy(priceLevels, func(p PriceLevel) float64 { return p.Quote }) {
		return 0, ErrInsufficientLiquidity
	}

	for _, priceLevel := range priceLevels {
		levelAmount := min(amtIn, priceLevel.Quote)
		amtOut += levelAmount * priceLevel.Price
		if amtIn -= levelAmount; amtIn <= 0 {
			return amtOut, nil
		}
	}
	return 0, ErrInsufficientLiquidity
}

func getAmountIn(amtOut, minAmtOut float64, priceLevels []PriceLevel) (amtIn float64, err error) {
	if len(priceLevels) == 0 {
		return 0, ErrEmptyPriceLevels
	} else if amtOut < minAmtOut {
		return 0, ErrAmtOutLessThanMinAllowed
	} else if amtOut > lo.SumBy(priceLevels, func(p PriceLevel) float64 { return p.Quote * p.Price }) {
		return 0, ErrInsufficientLiquidity
	}

	for _, priceLevel := range priceLevels {
		swappableAmount := min(amtOut, priceLevel.Quote*priceLevel.Price)
		amtIn += swappableAmount / priceLevel.Price
		if amtOut -= swappableAmount; amtOut <= 0 {
			return amtIn, nil
		}
	}
	return 0, ErrInsufficientLiquidity
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
