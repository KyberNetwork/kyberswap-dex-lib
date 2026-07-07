package skypsm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Extra struct {
	Rate           *uint256.Int `json:"rate"`
	BlockTimestamp uint64       `json:"blockTimestamp"`
}

type StaticExtra struct {
	RateProvider string         `json:"rateProvider"`
	Pocket       common.Address `json:"pocket"`
}

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type Gas struct {
	SwapExactIn int64
}
