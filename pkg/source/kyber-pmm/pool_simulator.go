package kyberpmm

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/KyberNetwork/blockchain-toolkit/float"
	"github.com/KyberNetwork/blockchain-toolkit/integer"
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
	)

	if swapDirection == SwapDirectionBaseToQuote {
		result, err = p.swapBaseToQuote(tokenAmountIn, tokenOut)
	} else {
		result, err = p.swapQuoteToBase(tokenAmountIn, tokenOut)
	}
	if err != nil {
		return nil, err
	}

	var inventoryLimit *big.Int
	if swapDirection == SwapDirectionBaseToQuote {
		inventoryLimit = limit.GetLimit(p.quoteToken.Address)
	} else {
		inventoryLimit = limit.GetLimit(p.baseToken.Address)
	}

	if result.TokenAmountOut.Amount.Cmp(inventoryLimit) > 0 {
		return nil, errors.New("not enough inventory")
	}
	return result, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapDirection := p.getSwapDirection(params.TokenAmountIn.Token)

	if swapDirection == SwapDirectionBaseToQuote {
		amountInAfterDecimals := new(big.Float).Quo(
			new(big.Float).SetInt(params.TokenAmountIn.Amount),
			bignumber.TenPowDecimals(p.baseToken.Decimals),
		)

		p.baseToQuotePriceLevels = getNewPriceLevelsState(amountInAfterDecimals, p.baseToQuotePriceLevels)
		newQuoteInventory, newBaseInventory, err := params.SwapLimit.UpdateLimit(p.quoteToken.Address, p.baseToken.Address, params.TokenAmountOut.Amount, params.TokenAmountIn.Amount)
		if err != nil {
			fmt.Println("unable to update PMM info, error:", err)
		}
		p.QuoteBalance = newQuoteInventory
		p.BaseBalance = newBaseInventory
	} else {
		amountInAfterDecimals := new(big.Float).Quo(
			new(big.Float).SetInt(params.TokenAmountIn.Amount),
			bignumber.TenPowDecimals(p.quoteToken.Decimals),
		)

		p.quoteToBasePriceLevels = getNewPriceLevelsState(amountInAfterDecimals, p.quoteToBasePriceLevels)

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

func (p *PoolSimulator) swapBaseToQuote(tokenAmountIn pool.TokenAmount, tokenOut string) (*pool.CalcAmountOutResult, error) {
	amountInAfterDecimals := new(big.Float).Quo(
		new(big.Float).SetInt(tokenAmountIn.Amount),
		bignumber.TenPowDecimals(p.baseToken.Decimals),
	)

	amountOutAfterDecimals, err := getAmountOut(amountInAfterDecimals, p.baseToQuotePriceLevels)
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

func (p *PoolSimulator) swapQuoteToBase(tokenAmountIn pool.TokenAmount, tokenOut string) (*pool.CalcAmountOutResult, error) {
	amountInAfterDecimals := new(big.Float).Quo(
		new(big.Float).SetInt(tokenAmountIn.Amount),
		bignumber.TenPowDecimals(p.quoteToken.Decimals),
	)

	amountOutAfterDecimals, err := getAmountOut(amountInAfterDecimals, p.quoteToBasePriceLevels)
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

// Inventory implement Swap Limit for kyber-pmm
// key is tokenAddress, and the limit is its balance
// The balance is stored WITHOUT decimals
// DONOT directly modify it, use UpdateLimit if needed
type Inventory struct {
	lock    *sync.RWMutex
	Balance map[string]*big.Int
}

func NewInventory(balance map[string]*big.Int) *Inventory {
	return &Inventory{
		lock:    &sync.RWMutex{},
		Balance: balance,
	}
}

// GetLimit returns a copy of balance for the token in Inventory
func (i *Inventory) GetLimit(tokenAddress string) *big.Int {
	i.lock.RLock()
	defer i.lock.RUnlock()
	balance, avail := i.Balance[tokenAddress]
	if !avail {
		return big.NewInt(0)
	}
	return big.NewInt(0).Set(balance)
}

// CheckLimit returns a copy of balance for the token in Inventory
func (i *Inventory) CheckLimit(tokenAddress string, amount *big.Int) error {
	i.lock.RLock()
	defer i.lock.RUnlock()
	balance, avail := i.Balance[tokenAddress]
	if !avail {
		return ErrTokenNotFound
	}
	if balance.Cmp(amount) < 0 {
		return ErrInsufficientLiquidity
	}
	return nil
}

// UpdateLimit will reduce the limit to reflect the change in inventory
// note this delta is amount without Decimal
func (i *Inventory) UpdateLimit(decreaseTokenAddress, increaseTokenAddress string, decreaseDelta, increaseDelta *big.Int) (*big.Int, *big.Int, error) {
	i.lock.Lock()
	defer i.lock.Unlock()
	decreasedTokenBalance, avail := i.Balance[decreaseTokenAddress]
	if !avail {
		return big.NewInt(0), big.NewInt(0), pool.ErrTokenNotAvailable
	}
	if decreasedTokenBalance.Cmp(decreaseDelta) < 0 {
		return big.NewInt(0), big.NewInt(0), pool.ErrNotEnoughInventory
	}
	i.Balance[decreaseTokenAddress] = decreasedTokenBalance.Sub(decreasedTokenBalance, decreaseDelta)

	increasedTokenBalance, avail := i.Balance[increaseTokenAddress]
	if !avail {
		return big.NewInt(0), big.NewInt(0), pool.ErrTokenNotAvailable
	}
	i.Balance[increaseTokenAddress] = increasedTokenBalance.Add(increasedTokenBalance, increaseDelta)
	return big.NewInt(0).Set(i.Balance[decreaseTokenAddress]), big.NewInt(0).Set(i.Balance[increaseTokenAddress]), nil
}
