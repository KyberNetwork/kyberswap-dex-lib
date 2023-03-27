package synthetix

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Slot0 struct {
	SqrtPriceX96               *big.Int `json:"sqrtPriceX96"`
	Tick                       *big.Int `json:"tick"`
	ObservationIndex           uint16   `json:"observationIndex"`
	ObservationCardinality     uint16   `json:"observationCardinality"`
	ObservationCardinalityNext uint16   `json:"observationCardinalityNext"`
	FeeProtocol                uint8    `json:"feeProtocol"`
	Unlocked                   bool     `json:"unlocked"`
}

type OracleObservation struct {
	// the block timestamp of the observation
	BlockTimestamp uint32 `json:"blockTimestamp"`
	// the tick accumulator, i.e. tick * time elapsed since the pool was first initialized
	TickCumulative *big.Int `json:"tickCumulative"`
	// the seconds per liquidity, i.e. seconds elapsed / max(1, liquidity) since the pool was first initialized
	SecondsPerLiquidityCumulativeX128 *big.Int `json:"secondsPerLiquidityCumulativeX128"`
	// whether or not the observation is initialized
	Initialized bool `json:"initialized"`
}

type DexPriceAggregatorUniswapV3 struct {
	DefaultPoolFee         *big.Int                                `json:"defaultPoolFee"`
	UniswapV3Factory       common.Address                          `json:"uniswapV3Factory"`
	Weth                   common.Address                          `json:"weth"`
	BlockTimestamp         uint64                                  `json:"blockTimestamp"`
	OverriddenPoolForRoute map[string]common.Address               `json:"overriddenPoolForRoute"`
	UniswapV3Slot0         map[string]Slot0                        `json:"uniswapV3Slot0"`
	UniswapV3Observations  map[string]map[uint16]OracleObservation `json:"uniswapV3Observations"`
	TickCumulatives        map[string][]*big.Int                   `json:"tickCumulatives"`
}

func NewDexPriceAggregatorUniswapV3() *DexPriceAggregatorUniswapV3 {
	return &DexPriceAggregatorUniswapV3{
		OverriddenPoolForRoute: make(map[string]common.Address),
		UniswapV3Slot0:         make(map[string]Slot0),
		UniswapV3Observations:  make(map[string]map[uint16]OracleObservation),
		TickCumulatives:        make(map[string][]*big.Int),
	}
}
