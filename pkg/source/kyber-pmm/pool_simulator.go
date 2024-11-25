package kyberpmm

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/float"
	"github.com/KyberNetwork/blockchain-toolkit/integer"
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
	QuoteBalance           *big.Int
	BaseBalance            *big.Int
	timestamp              int64
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
		baseBalance           = big.NewInt(0)
		quoteBalance          = big.NewInt(0)
	)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		amount, ok := big.NewInt(0).SetString(entityPool.Reserves[i], 10)
		if !ok {
			return nil, errors.New("could not parse PMM reserve to big.Float")
		}
		if strings.EqualFold(staticExtra.BaseTokenAddress, entityPool.Tokens[i].Address) {
			baseToken = *entityPool.Tokens[i]
			baseBalance.Set(amount)
		}

		if strings.EqualFold(staticExtra.QuoteTokenAddress, entityPool.Tokens[i].Address) {
			quoteToken = *entityPool.Tokens[i]
			quoteBalance.Set(amount)
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
				SwapFee:    integer.Zero(), // fee is added in the price levels already
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
		BaseBalance:            baseBalance,
		QuoteBalance:           quoteBalance,
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
		swapDirection = p.getSwapDirection(tokenAmountIn.Token)
		swapped       = limit.GetSwapped()
	)

	if swapDirection == SwapDirectionBaseToQuote {
		baseToQuotePriceLevels := p.baseToQuotePriceLevels
		swappedBaseAmount, ok := swapped[p.baseToken.Address]
		if ok && swappedBaseAmount.Sign() > 0 {
			swappedBaseAmountAfterDecimals := new(big.Float).Quo(
				new(big.Float).SetInt(swappedBaseAmount),
				bignumber.TenPowDecimals(p.baseToken.Decimals),
			)
			baseToQuotePriceLevels = getNewPriceLevelsState(swappedBaseAmountAfterDecimals, p.baseToQuotePriceLevels)
		}

		result, err = p.swapBaseToQuote(tokenAmountIn, tokenOut, baseToQuotePriceLevels)
	} else {
		quoteToBasePriceLevels := p.quoteToBasePriceLevels
		swappedQuoteAmount, ok := swapped[p.quoteToken.Address]
		if ok && swappedQuoteAmount.Sign() > 0 {
			swappedQuoteAmountAfterDecimals := new(big.Float).Quo(
				new(big.Float).SetInt(param.TokenAmountIn.Amount),
				bignumber.TenPowDecimals(p.quoteToken.Decimals),
			)

			quoteToBasePriceLevels = getNewPriceLevelsState(swappedQuoteAmountAfterDecimals, p.quoteToBasePriceLevels)
		}
		result, err = p.swapQuoteToBase(tokenAmountIn, tokenOut, quoteToBasePriceLevels)
	}
	if err != nil {
		return nil, err
	}

	var (
		inventoryLimitOut *big.Int
		// inventoryLimitIn  *big.Int
	)
	if swapDirection == SwapDirectionBaseToQuote {
		inventoryLimitOut = limit.GetLimit(p.quoteToken.Address)
		// inventoryLimitIn = limit.GetLimit(p.baseToken.Address)
	} else {
		inventoryLimitOut = limit.GetLimit(p.baseToken.Address)
		// inventoryLimitIn = limit.GetLimit(p.quoteToken.Address)
	}

	// log.Println("[DEBUG] limit", param.TokenAmountIn.Token, inventoryLimitOut)
	// if tokenAmountIn.Amount.Cmp(inventoryLimitIn) > 0 {
	// 	log.Println("[DEBUG] not enough inventory in", param.TokenAmountIn.Token, inventoryLimitOut)
	// 	return nil, errors.New("not enough inventory in")
	// }
	if result.TokenAmountOut.Amount.Cmp(inventoryLimitOut) > 0 {
		return nil, errors.New("not enough inventory out")
	}

	return result, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapDirection := p.getSwapDirection(params.TokenAmountIn.Token)

	if swapDirection == SwapDirectionBaseToQuote {
		newQuoteInventory, newBaseInventory, err := params.SwapLimit.UpdateLimit(p.quoteToken.Address, p.baseToken.Address, params.TokenAmountOut.Amount, params.TokenAmountIn.Amount)
		if err != nil {
			fmt.Println("unable to update PMM info, error:", err)
		}
		p.QuoteBalance = newQuoteInventory
		p.BaseBalance = newBaseInventory
	} else {
		newBaseInventory, newQuoteInventory, err := params.SwapLimit.UpdateLimit(p.baseToken.Address, p.quoteToken.Address, params.TokenAmountOut.Amount, params.TokenAmountIn.Amount)
		if err != nil {
			fmt.Println("unable to update PMM info, error:", err)
		}
		p.QuoteBalance = newQuoteInventory
		p.BaseBalance = newBaseInventory
	}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return RFQMeta{
		Timestamp: p.timestamp,
	}
}

func (p *PoolSimulator) getSwapDirection(tokenIn string) SwapDirection {
	if strings.EqualFold(tokenIn, p.baseToken.Address) {
		return SwapDirectionBaseToQuote
	}

	return SwapDirectionQuoteToBase
}

func (p *PoolSimulator) swapBaseToQuote(tokenAmountIn pool.TokenAmount, tokenOut string, priceLevels []PriceLevel) (*pool.CalcAmountOutResult, error) {
	amountInAfterDecimals := new(big.Float).Quo(
		new(big.Float).SetInt(tokenAmountIn.Amount),
		bignumber.TenPowDecimals(p.baseToken.Decimals),
	)

	amountOutAfterDecimals, err := getAmountOut(amountInAfterDecimals, priceLevels)
	if err != nil {
		return nil, err
	}

	amountOut, _ := new(big.Float).Mul(
		amountOutAfterDecimals,
		bignumber.TenPowDecimals(p.quoteToken.Decimals),
	).Int(nil)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: integer.Zero()},
		Gas:            p.gas.Swap,
		SwapInfo: SwapExtra{
			TakerAsset:   tokenAmountIn.Token,
			TakingAmount: tokenAmountIn.Amount.String(),
			MakerAsset:   tokenOut,
			MakingAmount: amountOut.String(),
		},
	}, nil
}

func (p *PoolSimulator) swapQuoteToBase(tokenAmountIn pool.TokenAmount, tokenOut string, priceLevels []PriceLevel) (*pool.CalcAmountOutResult, error) {
	amountInAfterDecimals := new(big.Float).Quo(
		new(big.Float).SetInt(tokenAmountIn.Amount),
		bignumber.TenPowDecimals(p.quoteToken.Decimals),
	)

	amountOutAfterDecimals, err := getAmountOut(amountInAfterDecimals, priceLevels)
	if err != nil {
		return nil, err
	}

	amountOut, _ := new(big.Float).Mul(
		amountOutAfterDecimals,
		bignumber.TenPowDecimals(p.baseToken.Decimals),
	).Int(nil)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: tokenAmountIn.Token, Amount: integer.Zero()},
		Gas:            p.gas.Swap,
		SwapInfo: SwapExtra{
			TakerAsset:   tokenAmountIn.Token,
			TakingAmount: tokenAmountIn.Amount.String(),
			MakerAsset:   tokenOut,
			MakingAmount: amountOut.String(),
		},
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
