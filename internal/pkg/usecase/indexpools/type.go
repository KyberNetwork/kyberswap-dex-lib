package indexpools

import (
	"fmt"
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	mapset "github.com/deckarep/golang-set/v2"
)

type TradesGenerationOutput struct {
	Successed map[string]map[TradePair][]TradeData
	Failed    map[string]map[TradePair][]TradeData

	Blacklist mapset.Set[string]
}

type price struct {
	buyPrice  float64
	sellPrice float64
}

func (p *price) getBuyPrice() float64 {
	if p.buyPrice == float64(0) {
		return p.sellPrice
	}

	return p.buyPrice
}

func (p *price) getSellPrice() float64 {
	if p.sellPrice == float64(0) {
		return p.buyPrice
	}

	return p.sellPrice
}

type TradePair struct {
	tokenIn  string
	tokenOut string
}

func (t TradePair) String() string {
	return fmt.Sprintf("%s-%s", t.tokenIn, t.tokenOut)
}

type TradeData struct {
	TokenIn      string  `json:"TokenIn"`
	TokenOut     string  `json:"TokenOut"`
	PriceImpact  float64 `json:"-"`
	AmountInUsd  float64 `json:"AmountInUsd"`
	AmountOutUsd float64 `json:"AmountOutUsd"`
	Pool         string  `json:"Pool"`

	// debug only fields when swaps error
	AmountIn   string `json:"AmountIn"`
	Err        error  `json:"-"`
	ErrMessage string `json:"error,omitempty"`
	Dex        string `json:"dex,omitempty"`
}

func (t *TradeData) hasError() bool {
	return t.Err != nil || t.AmountOutUsd == float64(0)
}

func (t *TradeData) getError() string {
	if t.Err != nil {
		return t.Err.Error()
	}
	return ErrAmountOutNotValid.Error()
}

func (t *TradeData) String() string {
	return fmt.Sprintf("tokenIn: %s, tokenOut: %s, amountInUsd: %f, AmountOutUsd: %f, pool: %s",
		t.TokenIn, t.TokenOut, t.AmountInUsd, t.AmountOutUsd, t.Pool)
}

func (t *TradeData) KeyError() string {
	return fmt.Sprintf("tokenIn: %s, tokenOut: %s, amountInUsd: %f, pool: %s",
		t.TokenIn, t.TokenOut, t.AmountInUsd, t.Pool)
}

// SwapLimit returned from this function format: map[PoolType]map[TokenAddress]*big.Int
type CalculateSwapLimit func(poolSimulator []poolpkg.IPoolSimulator) map[string]map[string]*big.Int

type UpdatePoolScores struct {
	rankingRepo IPoolRankRepository
	config      UpdateLiquidityScoreConfig
}

type BlacklistPoolIndex struct {
	repository IBlacklistIndexPoolRepository
}
