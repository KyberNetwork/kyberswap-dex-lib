package hashflowv3

import (
	"math"
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	MarketMaker          string
	Token0               entity.PoolToken
	Token1               entity.PoolToken
	ZeroToOnePriceLevels []PriceLevel
	OneToZeroPriceLevels []PriceLevel

	timestamp      int64
	priceTolerance int64
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

	zeroToOnePriceLevels := make([]PriceLevel, len(extra.ZeroToOnePriceLevels))
	for i, priceLevel := range extra.ZeroToOnePriceLevels {
		convertQuote, success := new(big.Float).SetString(priceLevel.Quote)
		if !success {
			return nil, ErrParsingBigFloat
		}
		convertPrice, success := new(big.Float).SetString(priceLevel.Price)
		if !success {
			return nil, ErrParsingBigFloat
		}

		zeroToOnePriceLevels[i] = PriceLevel{
			Quote: convertQuote,
			Price: convertPrice,
		}
	}

	oneToZeroPriceLevels := make([]PriceLevel, len(extra.OneToZeroPriceLevels))
	for i, priceLevel := range extra.OneToZeroPriceLevels {
		convertQuote, success := new(big.Float).SetString(priceLevel.Quote)
		if !success {
			return nil, ErrParsingBigFloat
		}
		convertPrice, success := new(big.Float).SetString(priceLevel.Price)
		if !success {
			return nil, ErrParsingBigFloat
		}

		oneToZeroPriceLevels[i] = PriceLevel{
			Quote: convertQuote,
			Price: convertPrice,
		}
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
		MarketMaker:          staticExtra.MarketMaker,
		Token0:               *entityPool.Tokens[0],
		Token1:               *entityPool.Tokens[1],
		ZeroToOnePriceLevels: zeroToOnePriceLevels,
		OneToZeroPriceLevels: oneToZeroPriceLevels,

		timestamp:      entityPool.Timestamp,
		priceTolerance: extra.PriceTolerance,
		gas:            defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Token == p.Token0.Address {
		return p.swap(params.TokenAmountIn.Amount, p.Token0, p.Token1, p.ZeroToOnePriceLevels)
	} else {
		return p.swap(params.TokenAmountIn.Amount, p.Token1, p.Token0, p.OneToZeroPriceLevels)
	}
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if params.TokenAmountOut.Token == p.Token1.Address {
		return p.swapExactOut(params.TokenAmountOut.Amount, p.Token0, p.Token1, p.ZeroToOnePriceLevels)
	} else {
		return p.swapExactOut(params.TokenAmountOut.Amount, p.Token1, p.Token0, p.OneToZeroPriceLevels)
	}
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.ZeroToOnePriceLevels = lo.Map(p.ZeroToOnePriceLevels, func(v PriceLevel, i int) PriceLevel {
		v.Quote = new(big.Float).Set(v.Quote)
		return v
	})
	cloned.OneToZeroPriceLevels = lo.Map(p.OneToZeroPriceLevels, func(v PriceLevel, i int) PriceLevel {
		v.Quote = new(big.Float).Set(v.Quote)
		return v
	})
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	var amountInAfterDecimals, decimalsPow, amountInBF big.Float
	amountInBF.SetInt(params.TokenAmountIn.Amount)

	if params.TokenAmountIn.Token == p.Token0.Address {
		decimalsPow.SetFloat64(math.Pow10(int(p.Token0.Decimals)))
		amountInAfterDecimals.Quo(&amountInBF, &decimalsPow)

		p.ZeroToOnePriceLevels = getNewPriceLevelsState(&amountInAfterDecimals, p.ZeroToOnePriceLevels)
	} else {
		decimalsPow.SetFloat64(math.Pow10(int(p.Token1.Decimals)))
		amountInAfterDecimals.Quo(&amountInBF, &decimalsPow)

		p.OneToZeroPriceLevels = getNewPriceLevelsState(&amountInAfterDecimals, p.OneToZeroPriceLevels)
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return MetaInfo{Timestamp: p.timestamp}
}

func (p *PoolSimulator) swap(amountIn *big.Int, baseToken, quoteToken entity.PoolToken, priceLevel []PriceLevel) (*pool.CalcAmountOutResult, error) {
	var amountInAfterDecimals, decimalsPow, amountInBF, amountOutBF, priceToleranceBF, amountOutToleranceBF big.Float

	amountInBF.SetInt(amountIn)
	decimalsPow.SetFloat64(math.Pow10(int(baseToken.Decimals)))
	amountInAfterDecimals.Quo(&amountInBF, &decimalsPow)

	var amountOutAfterDecimals big.Float
	// Passing amountOutAfterDecimals to the function to avoid allocation
	err := getAmountOut(&amountInAfterDecimals, priceLevel, &amountOutAfterDecimals)
	if err != nil {
		return nil, err
	}

	decimalsPow.SetFloat64(math.Pow10(int(quoteToken.Decimals)))
	amountOutBF.Mul(&amountOutAfterDecimals, &decimalsPow)

	priceToleranceBF.SetFloat64(float64(p.priceTolerance) / float64(priceToleranceBps))
	amountOutToleranceBF.Mul(&priceToleranceBF, &amountOutBF)
	amountOutBF.Sub(&amountOutBF, &amountOutToleranceBF)

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
			MarketMaker:      p.MarketMaker,
		},
	}, nil
}

func (p *PoolSimulator) swapExactOut(amountOut *big.Int, baseToken, quoteToken entity.PoolToken, priceLevel []PriceLevel) (*pool.CalcAmountInResult, error) {
	var amountOutAfterDecimals, decimalsPow, amountInBF, amountOutBF, priceToleranceBF, amountInToleranceBF big.Float

	amountOutBF.SetInt(amountOut)
	decimalsPow.SetFloat64(math.Pow10(int(quoteToken.Decimals)))
	amountOutAfterDecimals.Quo(&amountOutBF, &decimalsPow)

	var amountInAfterDecimals big.Float
	// Passing amountInAfterDecimals to the function to avoid allocation
	err := getAmountIn(&amountOutAfterDecimals, priceLevel, &amountInAfterDecimals)
	if err != nil {
		return nil, err
	}

	decimalsPow.SetFloat64(math.Pow10(int(baseToken.Decimals)))
	amountInBF.Mul(&amountInAfterDecimals, &decimalsPow)

	priceToleranceBF.SetFloat64(float64(p.priceTolerance) / float64(priceToleranceBps))
	amountInToleranceBF.Mul(&priceToleranceBF, &amountInBF)
	amountInBF.Add(&amountInBF, &amountInToleranceBF) // amountIn = amountIn + tolerance

	amountIn, _ := amountInBF.Int(nil)

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

func getAmountOut(amountIn *big.Float, priceLevels []PriceLevel, amountOut *big.Float) error {
	if len(priceLevels) == 0 {
		return ErrEmptyPriceLevels
	}

	// Check lower bound
	if amountIn.Cmp(priceLevels[0].Quote) < 0 {
		return ErrAmountInIsLessThanLowestPriceLevel
	}

	// Check upper bound
	var supportedAmount big.Float
	for _, priceLevel := range priceLevels {
		supportedAmount.Add(&supportedAmount, priceLevel.Quote)
	}
	if amountIn.Cmp(&supportedAmount) > 0 {
		return ErrAmountInIsGreaterThanHighestPriceLevel
	}

	var currentLevelAmount, tmp big.Float // Use tmp for temporary calculation
	amountLeft := amountIn
	currentLevelIdx := 0

	for {
		currentLevel := priceLevels[currentLevelIdx]
		if amountLeft.Cmp(currentLevel.Quote) < 0 {
			currentLevelAmount.Set(amountLeft)
		} else {
			currentLevelAmount.Set(currentLevel.Quote)
		}

		amountOut.Add(amountOut, tmp.Mul(&currentLevelAmount, currentLevel.Price))
		amountLeft.Sub(amountLeft, &currentLevelAmount)
		currentLevelIdx++

		if amountLeft.Cmp(zeroBF) == 0 || currentLevelIdx == len(priceLevels) {
			break
		}
	}

	return nil
}

func getAmountIn(amountOut *big.Float, priceLevels []PriceLevel, amountIn *big.Float) error {
	if len(priceLevels) == 0 {
		return ErrEmptyPriceLevels
	}

	// Check lower bound
	if amountOut.Cmp(new(big.Float).Mul(priceLevels[0].Quote, priceLevels[0].Price)) < 0 {
		return ErrAmountOutIsLessThanLowestPriceLevel
	}

	// Check upper bound
	var supportedAmountOut big.Float
	for _, priceLevel := range priceLevels {
		supportedAmountOut.Add(&supportedAmountOut, new(big.Float).Mul(priceLevel.Quote, priceLevel.Price))
	}
	if amountOut.Cmp(&supportedAmountOut) > 0 {
		return ErrAmountOutIsGreaterThanHighestPriceLevel
	}

	amountLeft := new(big.Float).Set(amountOut)

	for _, priceLevel := range priceLevels {
		swappableAmount := new(big.Float).Mul(priceLevel.Quote, priceLevel.Price)
		if swappableAmount.Cmp(amountLeft) > 0 {
			swappableAmount = new(big.Float).Set(amountLeft)
		}

		amountIn.Add(amountIn, new(big.Float).Quo(swappableAmount, priceLevel.Price))
		amountLeft = amountLeft.Sub(amountLeft, swappableAmount)

		if amountLeft.Cmp(zeroBF) == 0 {
			break
		}
	}

	if amountLeft.Cmp(zeroBF) != 0 {
		return ErrInsufficientLiquidity // Should not happen
	}

	return nil
}

func getNewPriceLevelsState(amountIn *big.Float, priceLevels []PriceLevel) []PriceLevel {
	if len(priceLevels) == 0 {
		return priceLevels
	}

	amountLeft := amountIn
	currentLevelIdx := 0

	for {
		currentLevelAvailableAmount := priceLevels[currentLevelIdx].Quote

		if currentLevelAvailableAmount.Cmp(amountLeft) > 0 {
			// Update the price level at the current step because it's partially filled
			priceLevels[currentLevelIdx].Quote.Sub(currentLevelAvailableAmount, amountLeft)
			amountLeft.Set(zeroBF)
		} else {
			// Only increase the step if the current level is fully filled
			amountLeft.Sub(amountLeft, priceLevels[currentLevelIdx].Quote)
			priceLevels[currentLevelIdx].Quote.Set(zeroBF)
			currentLevelIdx += 1
		}

		if amountLeft.Cmp(zeroBF) == 0 || currentLevelIdx == len(priceLevels) {
			// We don't skip the used price levels, but just reset its quote to zero.
			break
		}
	}

	return priceLevels
}
