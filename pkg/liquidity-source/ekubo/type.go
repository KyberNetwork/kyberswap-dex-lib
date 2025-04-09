package ekubo

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
)

type ExtensionType int

type QuoteData struct {
	Tick      int32          `json:"tick"`
	SqrtRatio *big.Int       `json:"sqrtRatio"`
	Liquidity *big.Int       `json:"liquidity"`
	MinTick   int32          `json:"minTick"`
	MaxTick   int32          `json:"maxTick"`
	Ticks     []quoting.Tick `json:"ticks"`
}

type PoolData struct {
	CoreAddress string `json:"core_address"`
	Token0      string `json:"token0"`
	Token1      string `json:"token1"`
	Fee         string `json:"fee"`
	TickSpacing uint32 `json:"tick_spacing"`
	Extension   string `json:"extension"`
}

type GetAllPoolsResult = []PoolData

type Extra struct {
	quoting.PoolState
}

type StaticExtra struct {
	ExtensionType ExtensionType   `json:"extensionType"`
	PoolKey       quoting.PoolKey `json:"poolKey"`
}

type nextInitializedTick struct {
	*quoting.Tick
	Index     int
	SqrtRatio *big.Int
}

type SwapInfo struct {
	SkipAhead       uint32
	SqrtRatio       *big.Int
	Liquidity       *big.Int
	ActiveTickIndex int
}
