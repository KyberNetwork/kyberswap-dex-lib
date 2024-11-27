package deltaswapv1

import "github.com/holiman/uint256"

type Extra struct {
	DsFee                    uint8        `json:"dsFee"`
	DsFeeThreshold           uint8        `json:"dsFeeThreshold"`
	LiquidityEMA             *uint256.Int `json:"liquidityEMA"`
	LastLiquidityBlockNumber uint64       `json:"lastLiquidityBlockNumber"`
	TradeLiquidityEMA        *uint256.Int `json:"tradeLiquidityEMA"`
	LastTradeLiquiditySum    *uint256.Int `json:"lastTradeLiquiditySum"`
	LastTradeBlockNumber     uint64       `json:"lastTradeBlockNumber"`
}
