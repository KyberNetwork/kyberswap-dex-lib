package bebop

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
	Token0               entity.PoolToken
	Token1               entity.PoolToken
	ZeroToOnePriceLevels []PriceLevel
	OneToZeroPriceLevels []PriceLevel
	gas                  Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  strings.ToLower(entityPool.Address),
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
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
		gas:                  defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.Limit != nil && params.Limit.GetLimit("") != nil {
		return nil, pool.ErrNotEnoughInventory
	}

	if params.TokenAmountIn.Token == p.Info.Tokens[0] {
		return p.swap(params.TokenAmountIn.Amount, p.Token0, p.Token1, p.ZeroToOnePriceLevels)
	} else {
		return p.swap(params.TokenAmountIn.Amount, p.Token1, p.Token0, p.OneToZeroPriceLevels)
	}
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	amtInF, _ := params.TokenAmountIn.Amount.Float64()
	if params.TokenAmountIn.Token == p.Token0.Address {
		amtInAfterDecimals := amtInF / math.Pow10(int(p.Token0.Decimals))
		p.ZeroToOnePriceLevels = updatePriceLevelsState(amtInAfterDecimals, p.ZeroToOnePriceLevels)
	} else {
		amtInAfterDecimals := amtInF / math.Pow10(int(p.Token1.Decimals))
		p.OneToZeroPriceLevels = updatePriceLevelsState(amtInAfterDecimals, p.OneToZeroPriceLevels)
	}

	// to handle the "top levels of orderbook" issue
	// the swapLimit will be updated to 0, to limit using bebopRFQ once each route
	// ref:https://team-kyber.slack.com/archives/C061UNZDUVC/p1728974288547259
	if params.SwapLimit == nil {
		return
	}

	_, _, _ = params.SwapLimit.UpdateLimit(
		"", "",
		nil, nil,
	)
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	return nil
}

func (p *PoolSimulator) swap(amountIn *big.Int, baseToken, quoteToken entity.PoolToken,
	priceLevel []PriceLevel) (*pool.CalcAmountOutResult, error) {
	amtInF, _ := amountIn.Float64()
	amtInAfterDecimals := amtInF / math.Pow10(int(baseToken.Decimals))
	amtOutAfterDecimals, err := getAmountOut(amtInAfterDecimals, priceLevel)
	if err != nil {
		return nil, err
	}

	amtOutF := amtOutAfterDecimals * math.Pow10(int(quoteToken.Decimals))
	amountOut, _ := big.NewFloat(amtOutF).Int(nil)
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

func getAmountOut(amountIn float64, priceLevels []PriceLevel) (amountOut float64, err error) {
	if len(priceLevels) == 0 {
		return 0, ErrEmptyPriceLevels
	} else if amountIn > lo.SumBy(priceLevels, func(pl PriceLevel) float64 { return pl.Quote }) {
		return 0, ErrInsufficientLiquidity
	}

	for _, currentLevel := range priceLevels {
		currentLevelAmount := min(currentLevel.Quote, amountIn)
		amountOut += currentLevelAmount * currentLevel.Price
		amountIn -= currentLevelAmount
		if amountIn <= 0 {
			break
		}
	}

	return amountOut, nil
}

// updatePriceLevelsState MAY MUTATE priceLevels
func updatePriceLevelsState(amountIn float64, priceLevels []PriceLevel) []PriceLevel {
	for i, priceLevel := range priceLevels {
		if quote := priceLevel.Quote; quote > amountIn {
			priceLevels[i].Quote -= amountIn
			return priceLevels[i:]
		} else if quote == amountIn {
			return priceLevels[i+1:]
		} else {
			amountIn -= quote
		}
	}

	return nil
}
