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
	baseToken   entity.PoolToken
	quoteTokens []entity.PoolToken
	priceLevels []BaseQuotePriceLevels
	gas         Gas
	timestamp   int64
}

var _ = pool.RegisterFactory0(DexTypeKyberPMM, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var numTokens = len(entityPool.Tokens)
	var tokens = make([]string, numTokens)
	var reserves = make([]*big.Int, numTokens)

	if numTokens < 2 {
		return nil, fmt.Errorf("pool's number of tokens should equal or larger than 2")
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var (
		baseToken           entity.PoolToken
		quoteTokens         = make([]entity.PoolToken, 0, numTokens-1)
		priceLevels         = make([]BaseQuotePriceLevels, 0, numTokens-1)
		quoteAddresessesMap = make(map[string]struct{}, len(staticExtra.QuoteTokenAddresses))
	)
	for _, qAddr := range staticExtra.QuoteTokenAddresses {
		quoteAddresessesMap[strings.ToLower(qAddr)] = struct{}{}
	}

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		amount, ok := new(big.Int).SetString(entityPool.Reserves[i], 10)
		if !ok {
			return nil, errors.New("could not parse PMM reserve to big.Float")
		}
		if strings.EqualFold(staticExtra.BaseTokenAddress, entityPool.Tokens[i].Address) {
			baseToken = *entityPool.Tokens[i]
		}

		if _, exist := quoteAddresessesMap[strings.ToLower(entityPool.Tokens[i].Address)]; exist {
			quoteTokens = append(quoteTokens, *entityPool.Tokens[i])
		}

		reserves[i] = amount
	}
	for _, qToken := range quoteTokens {
		bqPriceLevel, exist := extra.PriceLevels[fmt.Sprintf("%s/%s", baseToken.Symbol, qToken.Symbol)]
		if !exist {
			continue
		}
		priceLevels = append(priceLevels, bqPriceLevel)
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
		baseToken:   baseToken,
		quoteTokens: quoteTokens,
		priceLevels: priceLevels,
		gas:         DefaultGas,
		timestamp:   entityPool.Timestamp,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	param pool.CalcAmountOutParams,
) (result *pool.CalcAmountOutResult, err error) {
	if param.Limit == nil {
		return nil, ErrNoSwapLimit
	}
	var (
		limit             = param.Limit
		inventoryLimitOut = limit.GetLimit(param.TokenOut)
		inventoryLimitIn  = limit.GetLimit(param.TokenAmountIn.Token)
	)
	if param.TokenAmountIn.Amount.Cmp(inventoryLimitIn) > 0 {
		return nil, fmt.Errorf("ErrNotEnoughInventoryIn: inv %s, req %s",
			inventoryLimitIn.String(), param.TokenAmountIn.Amount.String())
	}

	var (
		inToken, outToken entity.PoolToken
		priceLevels       []PriceLevel
		isBaseToQuote     bool
		quoteToken        string
	)

	if strings.EqualFold(param.TokenAmountIn.Token, p.baseToken.Address) {
		quoteToken = param.TokenOut
		isBaseToQuote = true
		inToken = p.baseToken
	} else {
		quoteToken = param.TokenAmountIn.Token
		isBaseToQuote = false
		outToken = p.baseToken
	}

	for i := range p.quoteTokens {
		if !strings.EqualFold(p.quoteTokens[i].Address, quoteToken) {
			continue
		}
		if isBaseToQuote {
			priceLevels = p.priceLevels[i].BaseToQuotePriceLevels
			outToken = p.quoteTokens[i]
		} else {
			priceLevels = p.priceLevels[i].QuoteToBasePriceLevels
			inToken = p.quoteTokens[i]
		}
		break
	}

	amountInAfterDecimals := amountAfterDecimals(param.TokenAmountIn.Amount, inToken.Decimals)
	amountOutAfterDecimals, err := getAmountOut(amountInAfterDecimals, priceLevels)
	if err != nil {
		return nil, err
	}
	amountOut, _ := amountOutAfterDecimals.Mul(
		amountOutAfterDecimals,
		bignumber.TenPowDecimals(outToken.Decimals),
	).Int(nil)

	if amountOut.Cmp(inventoryLimitOut) > 0 {
		return nil, errors.New("not enough inventory out")
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: outToken.Address, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: inToken.Address, Amount: bignumber.ZeroBI},
		Gas:            p.gas.Swap,
		SwapInfo: SwapExtra{
			TakerAsset:   inToken.Address,
			TakingAmount: param.TokenAmountIn.Amount.String(),
			MakerAsset:   outToken.Address,
			MakingAmount: amountOut.String(),
		},
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	// remove related base levels
	if strings.EqualFold(params.TokenAmountIn.Token, p.baseToken.Address) {
		baseAmountAfterDecimals := amountAfterDecimals(params.TokenAmountIn.Amount, p.baseToken.Decimals)
		for i := range p.priceLevels {
			p.priceLevels[i].BaseToQuotePriceLevels = getNewPriceLevelsStateByAmountIn(
				baseAmountAfterDecimals,
				p.priceLevels[i].BaseToQuotePriceLevels,
			)
		}
	} else {
		baseAmountAfterDecimals := amountAfterDecimals(params.TokenAmountOut.Amount, p.baseToken.Decimals)
		for i := range p.priceLevels {
			p.priceLevels[i].QuoteToBasePriceLevels = getNewPriceLevelsStateByAmountOut(
				baseAmountAfterDecimals,
				p.priceLevels[i].QuoteToBasePriceLevels,
			)
		}
	}

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

func (p *PoolSimulator) CanSwapTo(address string) []string {
	if strings.EqualFold(p.baseToken.Address, address) {
		result := make([]string, 0, len(p.quoteTokens))
		for i := range p.quoteTokens {
			result = append(result, p.quoteTokens[i].Address)
		}
		return result
	}

	if slices.ContainsFunc(p.quoteTokens, func(t entity.PoolToken) bool {
		return strings.EqualFold(t.Address, address)
	}) {
		return []string{p.baseToken.Address}
	}

	return nil
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	return p.CanSwapTo(address)
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

func getNewPriceLevelsStateByAmountIn(amountIn *big.Float, priceLevels []PriceLevel) []PriceLevel {
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

func getNewPriceLevelsStateByAmountOut(amountOut *big.Float, priceLevels []PriceLevel) []PriceLevel {
	if len(priceLevels) == 0 {
		return priceLevels
	}

	var tmpAmt, tmpPrice, amountOutLeft big.Float
	amountOutLeft.Set(amountOut)
	for currentLevelIdx, priceLevel := range priceLevels {
		swappableAmount := tmpAmt.SetFloat64(priceLevel.Amount)
		swappableAmount.Mul(swappableAmount, tmpPrice.SetFloat64(priceLevel.Price))

		cmp := swappableAmount.Cmp(&amountOutLeft)
		// full filled 1 level
		if cmp < 0 {
			amountOutLeft.Sub(&amountOutLeft, swappableAmount)
			continue
		}

		// fully filled amount out
		if cmp == 0 {
			return priceLevels[currentLevelIdx+1:]
		}

		// partially filled. Must clone so as not to mutate old price level
		priceLevels = slices.Clone(priceLevels[currentLevelIdx:])
		swappableAmount.Sub(swappableAmount, &amountOutLeft)
		swappableAmount.Quo(swappableAmount, tmpPrice.SetFloat64(priceLevels[0].Price))

		priceLevels[0].Amount, _ = swappableAmount.Float64()
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
