package brownfiv3

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type GetReservesResult struct {
	Reserve0           *big.Int
	Reserve1           *big.Int
	BlockTimestampLast uint32
}

type PriceResult struct {
	Price       uint64
	Conf        uint64
	Expo        int32
	PublishTime uint64
}

// PairConfigResult decodes the getConfig(pair) tuple return.
type PairConfigResult struct {
	KB           *big.Int `abi:"kB"`
	KQ           *big.Int `abi:"kQ"`
	Lambda       uint64
	Fee          uint32
	FeeSplit     uint32
	Compress     uint32
	SSell        uint32 `abi:"sSell"`
	SBuy         uint32 `abi:"sBuy"`
	FixS         uint32 `abi:"fixS"`
	DisThreshold uint32
	SBound       uint32 `abi:"sBound"`
	PythWeight   uint32
	Gamma        uint32
}

// Extra holds dynamic per-block pool state.
type Extra struct {
	// AMM parameters
	KB    *uint256.Int `json:"kB,omitempty"`
	KQ    *uint256.Int `json:"kQ,omitempty"`
	Fee   uint32       `json:"f,omitempty"`
	Gamma uint32       `json:"g,omitempty"`

	// Spread / skew parameters from pairConfig
	Lambda       uint64 `json:"l,omitempty"`
	SSell        uint32 `json:"ss,omitempty"`
	SBuy         uint32 `json:"sb,omitempty"`
	FixS         uint32 `json:"fs,omitempty"`
	Compress     uint32 `json:"cp,omitempty"`
	SBound       uint32 `json:"sbd,omitempty"`
	PythWeight   uint32 `json:"pw,omitempty"`
	DisThreshold uint32 `json:"dt,omitempty"`

	// Raw oracle state (Q64 dollar prices)
	Price0   *uint256.Int `json:"p0,omitempty"`  // Pyth price of token0
	Price1   *uint256.Int `json:"p1,omitempty"`  // Pyth price of token1
	Conf0    *uint256.Int `json:"c0,omitempty"`  // Pyth confidence of token0
	Conf1    *uint256.Int `json:"c1,omitempty"`  // Pyth confidence of token1
	AmmPrice *uint256.Int `json:"am,omitempty"`  // on-chain AMM relative price Q64 (quote/base), 0 if no valid pool

	PriceUpdateData []byte `json:"u,omitempty"`
	PythTimestamp   int64  `json:"pt,omitempty"`
}

// StaticExtra holds infrequently-changing pool configuration (updated hourly).
type StaticExtra struct {
	PriceFeedIds    [2]common.Hash `json:"f,omitempty"`
	PriceOracle     string         `json:"o,omitempty"`
	PairConfig      string         `json:"pc,omitempty"`
	QuoteTokenIndex uint8          `json:"qi,omitempty"`
	LastUpdated     int64          `json:"lu,omitempty"`
}

// SwapInfo is returned to the on-chain executor.
type SwapInfo struct {
	PriceUpdateData []byte `json:"u,omitempty"`
}

// PoolMeta carries approval and fee metadata.
type PoolMeta struct {
	pool.ApprovalInfo
	Fee uint32 `json:"fee,omitempty"`
}

type PythUpdateData struct {
	Binary struct {
		Data []string `json:"data,omitempty"`
	} `json:"binary,omitempty"`
	Parsed []struct {
		Price struct {
			Price       string `json:"price,omitempty"`
			Conf        string `json:"conf,omitempty"`
			Expo        int    `json:"expo,omitempty"`
			PublishTime int64  `json:"publish_time,omitempty"`
		} `json:"price,omitempty"`
	} `json:"parsed,omitempty"`
}
