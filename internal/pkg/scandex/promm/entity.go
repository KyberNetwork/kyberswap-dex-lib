package promm

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/duration"
)

type Token struct {
	Address  string `json:"id"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals string `json:"decimals"`
}

type SubgraphPool struct {
	ID                 string `json:"id"`
	FeeTier            string `json:"feeTier"`
	PoolType           string `json:"poolType"`
	CreatedAtTimestamp string `json:"createdAtTimestamp"`
	Token0             Token  `json:"token0"`
	Token1             Token  `json:"token1"`
}

type TickResp struct {
	TickIdx        string `json:"tickIdx"`
	LiquidityGross string `json:"liquidityGross"`
	LiquidityNet   string `json:"liquidityNet"`
}

type SubgraphPoolTicks struct {
	ID    string     `json:"id"`
	Ticks []TickResp `json:"ticks"`
}

type StaticExtra struct {
	PoolId string `json:"poolId"`
}

type Tick struct {
	Index          int      `json:"index"`
	LiquidityGross *big.Int `json:"liquidityGross"`
	LiquidityNet   *big.Int `json:"liquidityNet"`
}

type Extra struct {
	Liquidity     *big.Int `json:"liquidity"`
	ReinvestL     *big.Int `json:"reinvestL"`
	ReinvestLLast *big.Int `json:"reinvestLLast"`
	SqrtPriceX96  *big.Int `json:"sqrtPriceX96"`
	Tick          *big.Int `json:"tick"`
	Ticks         []Tick   `json:"ticks"`
}

type PoolState struct {
	SqrtP              *big.Int `json:"sqrtP"`
	CurrentTick        *big.Int `json:"currentTick"`
	NearestCurrentTick *big.Int `json:"nearestCurrentTick"`
	Locked             bool     `json:"locked"`
}

type LiquidityState struct {
	BaseL         *big.Int `json:"baseL"`
	ReinvestL     *big.Int `json:"reinvestL"`
	ReinvestLLast *big.Int `json:"reinvestLLast"`
}

type Properties struct {
	SwapFee                   float64
	SubgraphAPI               string
	FactoryAddress            string
	NewPoolJobIntervalSec     int64
	ReserveJobInterval        duration.Duration `json:"reserveJobInterval"`
	TotalSupplyJobIntervalSec int64
	NewPoolBulk               int
	UpdateReserveBulk         int
	ConcurrentBatches         int `json:"concurrentBatches"`
}

func NewProperties(data map[string]interface{}) (properties Properties, err error) {
	bodyBytes, _ := json.Marshal(data)
	err = json.Unmarshal(bodyBytes, &properties)
	if properties.NewPoolBulk == 0 {
		properties.NewPoolBulk = 100
	}
	if properties.UpdateReserveBulk == 0 {
		properties.UpdateReserveBulk = 100
	}
	if properties.ReserveJobInterval.Duration == 0 {
		properties.ReserveJobInterval.Duration = 600 * time.Second
	}
	if properties.TotalSupplyJobIntervalSec == 0 {
		properties.TotalSupplyJobIntervalSec = 600
	}
	if properties.ConcurrentBatches == 0 {
		properties.ConcurrentBatches = constant.DefaultConcurrentBatches
	}

	return
}
