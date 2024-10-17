package bebop

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
	if params.Limit.GetLimit("") != nil {
		return nil, pool.ErrNotEnoughInventory
	}

	if params.TokenAmountIn.Token == p.Info.Tokens[0] {
		return p.swap(params.TokenAmountIn.Amount, p.Token0, p.Token1, p.ZeroToOnePriceLevels)
	} else {
		return p.swap(params.TokenAmountIn.Amount, p.Token1, p.Token0, p.OneToZeroPriceLevels)
	}
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

	// to handle the "top levels of orderbook" issue
	// the swapLimit will be updated to 0, to limit using bebopRFQ once each route
	// ref:https://team-kyber.slack.com/archives/C061UNZDUVC/p1728974288547259
	_, _, _ = params.SwapLimit.UpdateLimit(
		"", "",
		nil, nil,
	)
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	var pmmInventory = make(map[string]*big.Int, len(p.GetTokens()))
	tokens := p.GetTokens()
	rsv := p.GetReserves()
	if len(tokens) != len(rsv) {
		return pmmInventory
	}

	for i, tok := range tokens {
		// rsv of a token can be set to 1 wei to bypass the aggregator check
		if rsv[i].Int64() == 1 {
			continue
		}

		pmmInventory[tok] = big.NewInt(0).Set(rsv[i]) //clone here.
	}

	return pmmInventory
}

func (p *PoolSimulator) swap(amountIn *big.Int, baseToken, quoteToken entity.PoolToken,
	priceLevel []PriceLevel) (*pool.CalcAmountOutResult, error) {

	var amountInAfterDecimals, decimalsPow, amountInBF, amountOutBF big.Float

	amountInBF.SetInt(amountIn)
	decimalsPow.SetFloat64(math.Pow10(int(baseToken.Decimals)))
	amountInAfterDecimals.Quo(&amountInBF, &decimalsPow)
	var amountOutAfterDecimals big.Float
	err := getAmountOut(&amountInAfterDecimals, priceLevel, &amountOutAfterDecimals)
	if err != nil {
		return nil, err
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
	}, nil
}

func getAmountOut(amountIn *big.Float, priceLevels []PriceLevel, amountOut *big.Float) error {
	if len(priceLevels) == 0 {
		return ErrEmptyPriceLevels
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
