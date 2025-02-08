package kyberpmm

import (
	"errors"
	"fmt"
	"math/big"
	"slices"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	baseToken              entity.PoolToken
	quoteToken             entity.PoolToken
	baseToQuotePriceLevels []PriceLevel
	quoteToBasePriceLevels []PriceLevel
	gas                    Gas
	timestamp              int64
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var numTokens = len(entityPool.Tokens)
	var tokens = make([]string, numTokens)
	var reserves = make([]*big.Int, numTokens)

	if numTokens != 2 {
		return nil, fmt.Errorf("pool's number of tokens should equal 2")
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var (
		baseToken, quoteToken entity.PoolToken
	)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		amount, ok := new(big.Int).SetString(entityPool.Reserves[i], 10)
		if !ok {
			return nil, errors.New("could not parse PMM reserve to big.Float")
		}
		if strings.EqualFold(staticExtra.BaseTokenAddress, entityPool.Tokens[i].Address) {
			baseToken = *entityPool.Tokens[i]
		}

		if strings.EqualFold(staticExtra.QuoteTokenAddress, entityPool.Tokens[i].Address) {
			quoteToken = *entityPool.Tokens[i]
		}
		reserves[i] = amount
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    bignumber.ZeroBI, // fee is added in the price levels already
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		baseToken:              baseToken,
		quoteToken:             quoteToken,
		baseToQuotePriceLevels: extra.BaseToQuotePriceLevels,
		quoteToBasePriceLevels: extra.QuoteToBasePriceLevels,
		gas:                    DefaultGas,
		timestamp:              entityPool.Timestamp,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	param pool.CalcAmountOutParams,
) (result *pool.CalcAmountOutResult, err error) {
	if param.Limit == nil {
		return nil, ErrNoSwapLimit
	}
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
		limit         = param.Limit
		swapped       = limit.GetSwapped()
	)

	priceLevels := p.baseToQuotePriceLevels
	inToken, outToken := p.baseToken, p.quoteToken
	if strings.EqualFold(tokenAmountIn.Token, p.quoteToken.Address) {
		priceLevels = p.quoteToBasePriceLevels
		inToken, outToken = p.quoteToken, p.baseToken
	}

	swappedAmount, ok := swapped[inToken.Address]
	if ok && swappedAmount.Sign() > 0 {
		swappedAmountAfterDecimals := amountAfterDecimals(swappedAmount, inToken.Decimals)
		priceLevels = getNewPriceLevelsState(swappedAmountAfterDecimals, priceLevels)
	}

	amountInAfterDecimals := amountAfterDecimals(tokenAmountIn.Amount, inToken.Decimals)
	amountOutAfterDecimals, err := getAmountOut(amountInAfterDecimals, priceLevels)
	if err != nil {
		return nil, err
	}
	amountOut, _ := amountOutAfterDecimals.Mul(
		amountOutAfterDecimals,
		bignumber.TenPowDecimals(outToken.Decimals),
	).Int(nil)

	inventoryLimitOut := limit.GetLimit(outToken.Address)
	// inventoryLimitIn = limit.GetLimit(inToken.Address)

	// log.Println("[DEBUG] limit", param.TokenAmountIn.Token, inventoryLimitOut)
	// if tokenAmountIn.Amount.Cmp(inventoryLimitIn) > 0 {
	// 	log.Println("[DEBUG] not enough inventory in", param.TokenAmountIn.Token, inventoryLimitOut)
	// 	return nil, errors.New("not enough inventory in")
	// }
	if amountOut.Cmp(inventoryLimitOut) > 0 {
		return nil, ErrNotEnoughInventoryOut
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: bignumber.ZeroBI},
		Gas:            p.gas.Swap,
		SwapInfo: SwapExtra{
			TakerAsset:   tokenAmountIn.Token,
			TakingAmount: tokenAmountIn.Amount.String(),
			MakerAsset:   tokenOut,
			MakingAmount: amountOut.String(),
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(
	param pool.CalcAmountInParams,
) (result *pool.CalcAmountInResult, err error) {
	if param.Limit == nil {
		return nil, ErrNoSwapLimit
	}

	var (
		tokenAmountOut = param.TokenAmountOut
		tokenIn        = param.TokenIn
		limit          = param.Limit
	)

	priceLevels := p.baseToQuotePriceLevels
	inToken, outToken := p.baseToken, p.quoteToken
	if strings.EqualFold(tokenIn, p.quoteToken.Address) {
		priceLevels = p.quoteToBasePriceLevels
		inToken, outToken = p.quoteToken, p.baseToken
	}

	inventoryLimitOut := limit.GetLimit(outToken.Address)
	if tokenAmountOut.Amount.Cmp(inventoryLimitOut) > 0 {
		return nil, ErrNotEnoughInventoryOut
	}

	swapped := limit.GetSwapped()
	swappedAmount, ok := swapped[inToken.Address]
	if ok && swappedAmount.Sign() > 0 {
		swappedAmountAfterDecimals := amountAfterDecimals(swappedAmount, inToken.Decimals)
		priceLevels = getNewPriceLevelsState(swappedAmountAfterDecimals, priceLevels)
	}

	amountOutAfterDecimals := amountAfterDecimals(tokenAmountOut.Amount, outToken.Decimals)
	amountInAfterDecimals, err := getAmountIn(amountOutAfterDecimals, priceLevels)
	if err != nil {
		return nil, err
	}
	amountIn, _ := amountInAfterDecimals.Mul(
		amountInAfterDecimals,
		bignumber.TenPowDecimals(inToken.Decimals),
	).Int(nil)

	inventoryLimitIn := limit.GetLimit(inToken.Address)
	if amountIn.Cmp(inventoryLimitIn) > 0 {
		return nil, ErrNotEnoughInventoryIn
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		Fee:           &pool.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
		Gas:           p.gas.Swap,
		SwapInfo: SwapExtra{
			TakerAsset:   tokenIn,
			TakingAmount: amountIn.String(),
			MakerAsset:   tokenAmountOut.Token,
			MakingAmount: tokenAmountOut.Amount.String(),
		},
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	_, _, err := params.SwapLimit.UpdateLimit(params.TokenAmountOut.Token,
		params.TokenAmountIn.Token, params.TokenAmountOut.Amount, params.TokenAmountIn.Amount)
	if err != nil {
		logger.Errorf("kyberpmm.UpdateBalance failed: %v", err)
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return RFQMeta{
		Timestamp: p.timestamp,
	}
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	var pmmInventory = make(map[string]*big.Int, len(p.GetTokens()))
	tokens := p.GetTokens()
	rsv := p.GetReserves()
	for i, tok := range tokens {
		pmmInventory[tok] = new(big.Int).Set(rsv[i]) // clone here.
	}
	return pmmInventory
}

func getAmountOut(amountIn *big.Float, priceLevels []PriceLevel) (*big.Float, error) {
	if len(priceLevels) == 0 {
		return nil, ErrEmptyPriceLevels
	}

	var availableAmountBF big.Float
	availableAmountBF.SetFloat64(lo.SumBy(priceLevels, func(priceLevel PriceLevel) float64 {
		return priceLevel.Amount
	}))

	if amountIn.Cmp(&availableAmountBF) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountOut := new(big.Float)
	amountInLeft := availableAmountBF.Set(amountIn)
	var tmp, price big.Float
	for _, priceLevel := range priceLevels {
		swappableAmount := tmp.SetFloat64(priceLevel.Amount)
		if swappableAmount.Cmp(amountInLeft) > 0 {
			swappableAmount = amountInLeft
		}

		amountOut = amountOut.Add(
			amountOut,
			price.Mul(swappableAmount, price.SetFloat64(priceLevel.Price)),
		)

		if amountInLeft.Cmp(swappableAmount) == 0 {
			break
		}

		amountInLeft = amountInLeft.Sub(amountInLeft, swappableAmount)
	}

	return amountOut, nil
}

func getAmountIn(amountOut *big.Float, priceLevels []PriceLevel) (*big.Float, error) {
	if len(priceLevels) == 0 {
		return nil, ErrEmptyPriceLevels
	}

	var availableAmountBF big.Float
	availableAmountBF.SetFloat64(lo.SumBy(priceLevels, func(priceLevel PriceLevel) float64 {
		return priceLevel.Amount * priceLevel.Price
	}))

	if amountOut.Cmp(&availableAmountBF) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountIn := new(big.Float)
	amountOutLeft := availableAmountBF.Set(amountOut)
	var tmp, price big.Float
	for _, priceLevel := range priceLevels {
		swappableAmount := tmp.SetFloat64(priceLevel.Amount * priceLevel.Price)
		if swappableAmount.Cmp(amountOutLeft) > 0 {
			swappableAmount = amountOutLeft
		}

		amountIn = amountIn.Add(
			amountIn,
			price.Quo(swappableAmount, price.SetFloat64(priceLevel.Price)),
		)

		if amountOutLeft.Cmp(swappableAmount) == 0 {
			break
		}

		amountOutLeft = amountOutLeft.Sub(amountOutLeft, swappableAmount)
	}

	return amountIn, nil
}

func getNewPriceLevelsState(amountIn *big.Float, priceLevels []PriceLevel) []PriceLevel {
	if len(priceLevels) == 0 {
		return priceLevels
	}

	var tmp, amountInLeft big.Float
	amountInLeft.Set(amountIn)
	for currentLevelIdx, priceLevel := range priceLevels {
		swappableAmount := tmp.SetFloat64(priceLevel.Amount)
		if cmp := swappableAmount.Cmp(&amountInLeft); cmp < 0 {
			amountInLeft.Sub(&amountInLeft, swappableAmount)
			continue
		} else if cmp == 0 { // fully filled
			return priceLevels[currentLevelIdx+1:]
		}

		// partially filled. Must clone so as not to mutate old price level
		priceLevels = slices.Clone(priceLevels[currentLevelIdx:])
		priceLevels[0].Amount, _ = swappableAmount.Sub(swappableAmount, &amountInLeft).Float64()
		return priceLevels
	}

	return nil
}

func amountAfterDecimals(amount *big.Int, decimals uint8) *big.Float {
	ret := new(big.Float)
	return ret.Quo(
		ret.SetInt(amount),
		bignumber.TenPowDecimals(decimals),
	)
}
