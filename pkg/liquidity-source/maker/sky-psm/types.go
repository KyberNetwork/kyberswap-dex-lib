package skypsm

import (
	"github.com/holiman/uint256"
)

type Extra struct {
	Rate           *uint256.Int `json:"rate"`
	BlockTimestamp uint64       `json:"blockTimestamp"`
}

type StaticExtra struct {
	RateProvider string `json:"rateProvider"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type Gas struct {
	SwapExactIn int64
}
