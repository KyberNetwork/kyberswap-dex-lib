package orderbook

import (
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	Gas         Gas
	levelsFroms [2][]Level
	tokens      [2]*entity.PoolToken
	minTrades   [2]float64
	fee         float64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	return NewPoolSimulatorWith(entityPool, MaxAge)
}

func NewPoolSimulatorWith(entityPool entity.Pool, maxAge time.Duration) (*PoolSimulator, error) {
	if time.Since(time.Unix(entityPool.Timestamp, 0)) > maxAge {
		return nil, ErrLevelsTooOld
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	firstLevelFrom0, firstLevelFrom1 := lo.FirstOrEmpty(extra.LevelsFrom[0]), lo.FirstOrEmpty(extra.LevelsFrom[1])
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
		Gas:         defaultGas,
		levelsFroms: extra.LevelsFrom,
		tokens:      [2]*entity.PoolToken(entity.ClonePoolTokens(entityPool.Tokens)),
		minTrades:   [2]float64{firstLevelFrom0.Size(), firstLevelFrom1.Size()},
		fee:         entityPool.SwapFee,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut, amtIn := params.TokenAmountIn.Token, params.TokenOut, params.TokenAmountIn.Amount
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	result, err := p.calcOut(amtIn, p.tokens[indexIn], p.tokens[indexOut], p.levelsFroms[indexIn], p.minTrades[indexIn])
	if err != nil {
		return nil, err
	}

	if limit := params.Limit; limit != nil {
		inventoryLimit := limit.GetLimit(tokenOut)
		if result.TokenAmountOut.Amount.Cmp(inventoryLimit) > 0 {
			return nil, ErrSwapLimitExceeded
		}
	}
	return result, nil
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenOut, amtOut := params.TokenIn, params.TokenAmountOut.Token, params.TokenAmountOut.Amount
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	if limit := params.Limit; limit != nil {
		inventoryLimitOut := limit.GetLimit(tokenOut)
		if amtOut.Cmp(inventoryLimitOut) > 0 {
			return nil, ErrSwapLimitExceeded
		}
	}

	return p.calcIn(amtOut, p.tokens[indexIn], p.tokens[indexOut], p.levelsFroms[indexIn], p.minTrades[indexIn])
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	tokens, reserves := p.GetTokens(), p.GetReserves()
	inventory := make(map[string]*big.Int, len(tokens))
	for i, token := range tokens {
		inventory[token] = reserves[i]
	}
	return inventory
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.levelsFroms = [2][]Level{
		slices.Clone(p.levelsFroms[0]),
		slices.Clone(p.levelsFroms[1]),
	}
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	amtIn := params.TokenAmountIn.Amount
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenAmountOut.Token
	indexIn := p.GetTokenIndex(tokenIn)

	amtInF, _ := amtIn.Float64()
	amtInAfterDecimals := amtInF / math.Pow10(int(p.tokens[indexIn].Decimals))
	p.levelsFroms[indexIn] = updateLevelsState(amtInAfterDecimals, p.levelsFroms[indexIn])

	if limit := params.SwapLimit; limit != nil {
		_, _, err := limit.UpdateLimit(tokenOut, tokenIn, params.TokenAmountOut.Amount, amtIn)
		if err != nil {
			log.Err(err).Msg("orderbook.UpdateBalance failed")
		}
	}
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return nil
}

func (p *PoolSimulator) calcOut(amountIn *big.Int, tokenIn, tokenOut *entity.PoolToken, priceLevel []Level,
	minTrade float64) (*pool.CalcAmountOutResult, error) {
	amtInF, _ := amountIn.Float64()
	amtInAfterDecimals := amtInF / math.Pow10(int(tokenIn.Decimals))
	amtOutAfterDecimals, levels, err := getAmountOut(amtInAfterDecimals, priceLevel, minTrade)
	if err != nil {
		return nil, err
	}

	amtOutF := amtOutAfterDecimals * math.Pow10(int(tokenOut.Decimals))
	amtOutF *= 1 - p.fee
	amountOut, _ := big.NewFloat(amtOutF).Int(nil)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut.Address, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: tokenIn.Address, Amount: bignumber.ZeroBI},
		Gas:            p.Gas.Base + levels*p.Gas.Level,
		SwapInfo: SwapInfo{
			BaseToken:          tokenIn.Address,
			BaseTokenAmount:    amountIn.String(),
			BaseTokenDecimals:  tokenIn.Decimals,
			QuoteToken:         tokenOut.Address,
			QuoteTokenAmount:   amountOut.String(),
			QuoteTokenDecimals: tokenOut.Decimals,
		},
	}, nil
}

func (p *PoolSimulator) calcIn(amountOut *big.Int, tokenIn, tokenOut *entity.PoolToken, priceLevel []Level,
	minTrade float64) (*pool.CalcAmountInResult, error) {
	amtOutF, _ := amountOut.Float64()
	amtOutF /= 1 - p.fee
	amtOutAfterDecimals := amtOutF / math.Pow10(int(tokenOut.Decimals))
	amtInAfterDecimals, levels, err := getAmountIn(amtOutAfterDecimals, priceLevel, minTrade)
	if err != nil {
		return nil, err
	}

	amtInF := amtInAfterDecimals * math.Pow10(int(tokenIn.Decimals))
	amountIn, _ := big.NewFloat(amtInF).Int(nil)

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn.Address, Amount: amountIn},
		Fee:           &pool.TokenAmount{Token: tokenIn.Address, Amount: bignumber.ZeroBI},
		Gas:           p.Gas.Base + levels*p.Gas.Level,
		SwapInfo: SwapInfo{
			BaseToken:          tokenIn.Address,
			BaseTokenAmount:    amountIn.String(),
			BaseTokenDecimals:  tokenIn.Decimals,
			QuoteToken:         tokenOut.Address,
			QuoteTokenAmount:   amountOut.String(),
			QuoteTokenDecimals: tokenOut.Decimals,
		},
	}, nil
}

func getAmountOut(amtIn float64, priceLevels []Level, minTrade float64) (amountOut float64, levels int64, err error) {
	if len(priceLevels) == 0 {
		return 0, 0, ErrEmptyLevels
	} else if amtIn < minTrade {
		return 0, 0, ErrInvalidAmountIn
	} else if amtIn > lo.SumBy(priceLevels, func(pl Level) float64 { return pl.Size() }) {
		return 0, 0, ErrInsufficientLiquidity
	}

	for _, currentLevel := range priceLevels {
		levels++
		currentLevelAmount := min(currentLevel.Size(), amtIn)
		amountOut += currentLevelAmount * currentLevel.Price()
		if amtIn -= currentLevelAmount; amtIn <= 0 {
			break
		}
	}

	return amountOut, levels, nil
}

func getAmountIn(amtOut float64, priceLevels []Level, minTrade float64) (amountIn float64, levels int64, err error) {
	if len(priceLevels) == 0 {
		return 0, 0, ErrEmptyLevels
	} else if amtOut < minTrade*priceLevels[0].Price() {
		return 0, 0, ErrInvalidAmountIn
	} else if amtOut > lo.SumBy(priceLevels, func(pl Level) float64 { return pl.Size() * pl.Price() }) {
		return 0, 0, ErrInsufficientLiquidity
	}

	for _, currentLevel := range priceLevels {
		levels++
		currentLevelAmount := min(currentLevel.Size()*currentLevel.Price(), amtOut)
		amountIn += currentLevelAmount / currentLevel.Price()
		if amtOut -= currentLevelAmount; amtOut <= 0 {
			break
		}
	}

	return amountIn, levels, nil
}

// updateLevelsState MAY MUTATE priceLevels
func updateLevelsState(amountIn float64, priceLevels []Level) []Level {
	for i, priceLevel := range priceLevels {
		if levelAmount := priceLevel.Size(); levelAmount > amountIn {
			priceLevels[i].SetSize(priceLevels[i].Size() - amountIn)
			return priceLevels[i:]
		} else if levelAmount == amountIn {
			return priceLevels[i+1:]
		} else {
			amountIn -= levelAmount
		}
	}

	return nil
}
