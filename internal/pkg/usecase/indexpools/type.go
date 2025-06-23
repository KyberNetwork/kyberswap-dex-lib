package indexpools

import (
	"fmt"
	"math/big"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	mapset "github.com/deckarep/golang-set/v2"
)

type LiquidityScoreCalcInput struct {
	TradeData []TradeData `json:"trade_data"`
	Liquidity float64     `json:"liquidity"`
}

func (i *LiquidityScoreCalcInput) AddTradeData(tradeData TradeData) {
	i.TradeData = append(i.TradeData, tradeData)
}

type TradesGenerationOutput struct {
	Successed map[TradeDataId]*LiquidityScoreCalcInput
	Failed    map[TradeDataId]*LiquidityScoreCalcInput

	Blacklist      mapset.Set[string]
	ZeroScorePools []entity.PoolScore
}

type TradeDataGenerationResult struct {
	OutputFileNames mapset.Set[string]
	Blacklist       mapset.Set[string]
	ZeroScorePools  []entity.PoolScore
}

type TradesGenerationInput struct {
	Pool     string
	Exchange string
}

type TradeDataId struct {
	Pool string
	Type valueobject.TradeDataType
}

type TradeData struct {
	/*
	 * Key value will be the exact key which is sorted set key in Redis
	 */
	Key string `json:"key"`
	// Type in trade data is whitelist-whitelist, token-whitelist, whitelist-token or direct
	Type         valueobject.TradeDataType `json:"-"`
	TokenIn      string                    `json:"token_in"`
	TokenOut     string                    `json:"token_out"`
	PriceImpact  float64                   `json:"-"`
	AmountInUsd  float64                   `json:"amount_in_usd"`
	AmountOutUsd float64                   `json:"amount_out_usd"`
	Pool         string                    `json:"pool"`

	// debug only fields when swaps error
	AmountIn   string `json:"amount_in"`
	Err        error  `json:"error,omitempty"`
	ErrMessage string `json:"error_message,omitempty"`
	Exchange   string `json:"-"`
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
